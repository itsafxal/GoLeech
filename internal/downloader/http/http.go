package http

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

	// Send HEAD or initial GET request to check headers, content length, and filename
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "GoLeech/1.0 (Mozilla/5.0 Compatible)")
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	totalSize := resp.ContentLength
	if totalSize > 0 {
		t.TotalSize = totalSize
		if cfg.DirectLimit > 0 && totalSize > cfg.DirectLimit {
			return "", fmt.Errorf("file size (%s) exceeds direct download limit (%s)", utils.FormatBytes(totalSize), utils.FormatBytes(cfg.DirectLimit))
		}
	}

	// Resolve filename
	filename := determineFilename(resp, rawURL, t.Options.Name)
	t.SetName(filename)
	destPath := filepath.Join(taskDir, filename)

	// Single connection streaming with progress updates
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer outFile.Close()

	var downloaded int64
	startTime := time.Now()
	lastUpdate := time.Now()
	var lastDownloaded int64

	buf := make([]byte, 64*1024) // 64KB buffer
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		n, rErr := resp.Body.Read(buf)
		if n > 0 {
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

	// Final progress update
	duration := time.Since(startTime).Seconds()
	avgSpeed := float64(downloaded) / duration
	if onProgress != nil {
		onProgress(downloaded, downloaded, avgSpeed)
	}

	return destPath, nil
}

func determineFilename(resp *http.Response, rawURL, customName string) string {
	if customName != "" {
		return utils.CleanFilename(customName)
	}

	// Check Content-Disposition
	cd := resp.Header.Get("Content-Disposition")
	if cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if name, ok := params["filename"]; ok && name != "" {
				return utils.CleanFilename(name)
			}
		}
	}

	// Fallback to URL path
	parsed, err := url.Parse(rawURL)
	if err == nil {
		base := filepath.Base(parsed.Path)
		if base != "" && base != "/" && base != "." {
			return utils.CleanFilename(base)
		}
	}

	return fmt.Sprintf("download_%d.bin", time.Now().Unix())
}

// MultiChunkDownloader handles parallel range downloads for servers supporting Accept-Ranges
type MultiChunkDownloader struct {
	concurrency int
}

func NewMultiChunkDownloader(concurrency int) *MultiChunkDownloader {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &MultiChunkDownloader{concurrency: concurrency}
}

func (m *MultiChunkDownloader) DownloadChunk(ctx context.Context, url string, start, end int64, file *os.File, counter *int64) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	buf := make([]byte, 32*1024)
	offset := start
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := file.WriteAt(buf[:n], offset); wErr != nil {
				return wErr
			}
			offset += int64(n)
			atomic.AddInt64(counter, int64(n))
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			return rErr
		}
	}
	return nil
}
