package aria2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/downloader"
	dlHTTP "github.com/GoLeech/GoLeech/internal/downloader/http"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
)

type Downloader struct {
	rpcURL string
	secret string
	client *http.Client
}

func NewDownloader(rpcURL, secret string) *Downloader {
	if rpcURL == "" {
		rpcURL = "http://localhost:6800/jsonrpc"
	}
	return &Downloader{
		rpcURL: rpcURL,
		secret: secret,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

type aria2Request struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type aria2Response struct {
	ID     string                 `json:"id"`
	Result map[string]interface{} `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// IsAvailable checks if the Aria2 RPC server is responsive
func (d *Downloader) IsAvailable(ctx context.Context) bool {
	var params []interface{}
	if d.secret != "" {
		params = append(params, "token:"+d.secret)
	}
	resp, err := d.callRPC(ctx, "aria2.getVersion", params)
	return err == nil && resp != nil
}

func (d *Downloader) Download(ctx context.Context, t *task.Task, onProgress downloader.ProgressCallback) (string, error) {
	link := t.Options.Link
	isTorrent := strings.HasPrefix(strings.ToLower(link), "magnet:?") || strings.HasSuffix(strings.ToLower(link), ".torrent") || link == "tg://torrent"

	// If Aria2 is not running, fall back to native HTTP downloader for direct URLs
	if !d.IsAvailable(ctx) {
		if isTorrent {
			return "", fmt.Errorf("aria2 daemon is required for torrent downloads (start aria2c with --enable-rpc)")
		}
		// Fallback to Native HTTP Downloader
		nativeHTTP := dlHTTP.NewDownloader()
		return nativeHTTP.Download(ctx, t, onProgress)
	}

	cfg := config.Get()
	taskDir := filepath.Join(cfg.DownloadDir, t.GID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create aria2 task directory: %w", err)
	}
	t.LocalPath = taskDir

	var params []interface{}
	if d.secret != "" {
		params = append(params, "token:"+d.secret)
	}

	options := map[string]string{
		"dir":                        taskDir,
		"max-connection-per-server": "16",
		"split":                      "16",
		"min-split-size":             "1M",
		"stream-piece-selector":      "in-order",
		"continue":                   "true",
		"allow-overwrite":            "true",
	}

	if t.Options.Name != "" {
		options["out"] = t.Options.Name
	}
	if t.Options.CustomHeader != "" {
		options["header"] = t.Options.CustomHeader
	}
	if t.Options.SelectFiles {
		options["pause"] = "true"
	}

	uris := []string{link}
	params = append(params, uris, options)

	resp, err := d.callRPC(ctx, "aria2.addUri", params)
	if err != nil {
		if !isTorrent {
			// Fallback to native HTTP if addUri fails
			nativeHTTP := dlHTTP.NewDownloader()
			return nativeHTTP.Download(ctx, t, onProgress)
		}
		return "", fmt.Errorf("failed to add task to aria2: %w", err)
	}

	ariaGID, _ := resp["result"].(string)
	if ariaGID == "" {
		return "", fmt.Errorf("aria2 did not return a valid download GID")
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			var removeParams []interface{}
			if d.secret != "" {
				removeParams = append(removeParams, "token:"+d.secret)
			}
			removeParams = append(removeParams, ariaGID)
			_, _ = d.callRPC(context.Background(), "aria2.forceRemove", removeParams)
			return "", ctx.Err()

		case <-ticker.C:
			var statusParams []interface{}
			if d.secret != "" {
				statusParams = append(statusParams, "token:"+d.secret)
			}
			statusParams = append(statusParams, ariaGID)

			st, err := d.callRPC(ctx, "aria2.tellStatus", statusParams)
			if err != nil {
				continue
			}

			res, _ := st["result"].(map[string]interface{})
			if res == nil {
				continue
			}

			status, _ := res["status"].(string)
			totStr, _ := res["totalLength"].(string)
			curStr, _ := res["completedLength"].(string)
			spdStr, _ := res["downloadSpeed"].(string)

			total, _ := strconv.ParseInt(totStr, 10, 64)
			current, _ := strconv.ParseInt(curStr, 10, 64)
			speed, _ := strconv.ParseFloat(spdStr, 64)

			if total > 0 {
				t.TotalSize = total
			}

			if onProgress != nil {
				onProgress(current, total, speed)
			}

			// If files list has path, update task name
			if files, ok := res["files"].([]interface{}); ok && len(files) > 0 {
				if f0, ok := files[0].(map[string]interface{}); ok {
					if p, ok := f0["path"].(string); ok && p != "" {
						t.SetName(filepath.Base(p))
					}
				}
			}

			if status == "complete" {
				files, _ := os.ReadDir(taskDir)
				for _, f := range files {
					if !f.IsDir() || len(files) == 1 {
						foundPath := filepath.Join(taskDir, f.Name())
						// Sniff magic bytes if file lacks extension
						ext := filepath.Ext(f.Name())
						if ext == "" || ext == ".bin" || ext == ".php" {
							if data, rErr := os.ReadFile(foundPath); rErr == nil && len(data) >= 12 {
								if detExt := utils.SniffFileExtension(data[:min(len(data), 512)], ""); detExt != "" {
									newP := strings.TrimSuffix(foundPath, ext) + detExt
									_ = os.Rename(foundPath, newP)
									foundPath = newP
									t.SetName(filepath.Base(newP))
								}
							}
						}
						return foundPath, nil
					}
				}
				return taskDir, nil
			}

			if status == "error" {
				errMsg, _ := res["errorMessage"].(string)
				return "", fmt.Errorf("aria2 error: %s", errMsg)
			}
		}
	}
}

func (d *Downloader) callRPC(ctx context.Context, method string, params []interface{}) (map[string]interface{}, error) {
	reqBody := aria2Request{
		JSONRPC: "2.0",
		ID:      task.GenerateGID(),
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.rpcURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
