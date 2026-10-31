package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/downloader"
	"github.com/GoLeech/GoLeech/internal/downloader/resolvers"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
)

type Downloader struct {
	client *http.Client
}

func NewDownloader() *Downloader {
	return &Downloader{
		client: &http.Client{
			Timeout: 0, // No global timeout for large streaming downloads
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 15 {
					return fmt.Errorf("stopped after 15 redirects")
				}
				return nil
			},
		},
	}
}

func (d *Downloader) Download(ctx context.Context, t *task.Task, onProgress downloader.ProgressCallback) (string, error) {
	rawURL := t.Options.Link
	directURL, err := resolvers.ResolveLink(rawURL)
	if err == nil && directURL != "" {
		rawURL = directURL
	}

	cfg := config.Get()
	taskDir := filepath.Join(cfg.DownloadDir, t.GID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create task download directory: %w", err)
	}
	t.LocalPath = taskDir

	// Prepare initial request
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("invalid download URL: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")

	if t.Options.CustomHeader != "" {
		parts := strings.SplitN(t.Options.CustomHeader, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
	if t.Options.AuthUser != "" {
		req.SetBasicAuth(t.Options.AuthUser, t.Options.AuthPass)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to connect to download source: %w", err)
	}
	defer resp.Body.Close()

	// Handle standard HTTP error statuses
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return "", fmt.Errorf("download authentication required (HTTP 401)")
	case http.StatusForbidden:
		return "", fmt.Errorf("access forbidden by remote host (HTTP 403)")
	case http.StatusNotFound:
		return "", fmt.Errorf("file not found on remote server (HTTP 404)")
	case http.StatusTooManyRequests:
		return "", fmt.Errorf("rate limited by download server (HTTP 429)")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download server returned HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	totalSize := resp.ContentLength
	if totalSize > 0 {
		t.TotalSize = totalSize
		if cfg.DirectLimit > 0 && totalSize > cfg.DirectLimit {
			return "", fmt.Errorf("file size (%s) exceeds direct download limit (%s)", utils.FormatBytes(totalSize), utils.FormatBytes(cfg.DirectLimit))
		}
	}

	// Resolve filename intelligently from Content-Disposition, final redirect URL, and query params
	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	filename := DetermineFilename(resp, finalURL, t.Options.Name)
	destPath := filepath.Join(taskDir, filename)

	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	var downloaded int64
	startTime := time.Now()
	lastUpdate := time.Now()
	var lastDownloaded int64

	buf := make([]byte, 128*1024) // 128KB buffer for high throughput
	var sniffedHeader []byte
	headerChecked := false

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			if !headerChecked {
				sniffedHeader = append(sniffedHeader, buf[:min(n, 512-len(sniffedHeader))]...)
				if len(sniffedHeader) >= 512 || rErr == io.EOF {
					headerChecked = true
					// Check if file has missing or script extension and fix it
					contentType := resp.Header.Get("Content-Type")
					ext := filepath.Ext(filename)
					isScriptExt := ext == "" || ext == ".php" || ext == ".aspx" || ext == ".bin" || ext == ".cgi" || ext == ".do" || ext == ".jsp" || ext == ".html"
					if isScriptExt {
						detectedExt := utils.SniffFileExtension(sniffedHeader, contentType)
						if detectedExt != "" && detectedExt != ext {
							cleanBase := strings.TrimSuffix(filename, ext)
							if cleanBase == "" || cleanBase == "attachment" || cleanBase == "download" || cleanBase == "index" {
								cleanBase = fmt.Sprintf("file_%s", t.GID)
							}
							filename = cleanBase + detectedExt
						}
					}
				}
			}

			if _, wErr := outFile.Write(buf[:n]); wErr != nil {
				return "", fmt.Errorf("error writing to disk: %w", wErr)
			}
			downloaded += int64(n)

			now := time.Now()
			elapsed := now.Sub(lastUpdate)
			if elapsed >= 1*time.Second {
				speed := float64(downloaded-lastDownloaded) / elapsed.Seconds()
				if onProgress != nil {
					onProgress(downloaded, totalSize, speed)
				}
				lastUpdate = now
				lastDownloaded = downloaded
			}
		}

		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			return "", fmt.Errorf("connection error during download: %w", rErr)
		}
	}

	// Close output file so we can rename if needed
	_ = outFile.Close()

	// If filename changed after sniffing, rename file
	finalPath := filepath.Join(taskDir, filename)
	if finalPath != destPath {
		_ = os.Rename(destPath, finalPath)
	}
	t.SetName(filename)

	duration := time.Since(startTime).Seconds()
	avgSpeed := float64(downloaded) / duration
	if onProgress != nil {
		onProgress(downloaded, downloaded, avgSpeed)
	}

	return finalPath, nil
}

// DetermineFilename extracts the best human-readable filename from Content-Disposition, URL path, query params, or fallback
func DetermineFilename(resp *http.Response, rawURL, customName string) string {
	if customName != "" {
		return utils.CleanFilename(customName)
	}

	// 1. Content-Disposition Header (Standard & RFC 5987/6266 UTF-8)
	if resp != nil {
		cd := resp.Header.Get("Content-Disposition")
		if cd != "" {
			if parsed := utils.ParseContentDisposition(cd); parsed != "" {
				return parsed
			}
		}
	}

	// 2. URL Path and Query Parsing
	if parsed, err := url.Parse(rawURL); err == nil {
		base := filepath.Base(parsed.Path)
		ext := strings.ToLower(filepath.Ext(base))
		isScriptExt := ext == ".php" || ext == ".aspx" || ext == ".asp" || ext == ".jsp" || ext == ".do" || ext == ".html" || ext == ".htm"

		// If path base is not a script name, use it
		if base != "" && base != "/" && base != "." && !isScriptExt {
			return utils.CleanFilename(base)
		}

		// If path is a dynamic script, check query params: ?file=... or ?name=... or ?filename=...
		query := parsed.Query()
		for _, param := range []string{"file", "name", "filename", "title", "download"} {
			if val := query.Get(param); val != "" {
				cleanVal := filepath.Base(val)
				if cleanVal != "" && cleanVal != "." {
					return utils.CleanFilename(cleanVal)
				}
			}
		}
	}

	// 3. Fallback to Content-Type extension
	if resp != nil {
		ct := resp.Header.Get("Content-Type")
		ext := utils.SniffFileExtension(nil, ct)
		if ext != "" {
			return fmt.Sprintf("file_%d%s", time.Now().Unix(), ext)
		}
	}

	return fmt.Sprintf("download_%d.bin", time.Now().Unix())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
