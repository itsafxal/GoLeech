package torrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/downloader"
	"github.com/GoLeech/GoLeech/internal/task"
)

type Aria2Downloader struct {
	rpcURL string
	secret string
	client *http.Client
}

func NewAria2Downloader(rpcURL, secret string) *Aria2Downloader {
	if rpcURL == "" {
		rpcURL = "http://localhost:6800/jsonrpc"
	}
	return &Aria2Downloader{
		rpcURL: rpcURL,
		secret: secret,
		client: &http.Client{Timeout: 10 * time.Second},
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

func (d *Aria2Downloader) Download(ctx context.Context, t *task.Task, onProgress downloader.ProgressCallback) (string, error) {
	cfg := config.Get()
	taskDir := filepath.Join(cfg.DownloadDir, t.GID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create torrent directory: %w", err)
	}
	t.LocalPath = taskDir

	// Add URI to Aria2
	var params []interface{}
	if d.secret != "" {
		params = append(params, "token:"+d.secret)
	}
	uris := []string{t.Options.Link}
	options := map[string]string{
		"dir": taskDir,
	}
	if t.Options.SelectFiles {
		options["pause"] = "true" // allow user to select files first
	}

	params = append(params, uris, options)

	resp, err := d.callRPC(ctx, "aria2.addUri", params)
	if err != nil {
		return "", fmt.Errorf("failed to add torrent to aria2: %w (ensure aria2c daemon is running)", err)
	}

	ariaGID, _ := resp["result"].(string)
	if ariaGID == "" {
		return "", fmt.Errorf("aria2 did not return a valid download GID")
	}

	// Poll status until complete or cancelled
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Cancel in Aria2
			_, _ = d.callRPC(context.Background(), "aria2.forceRemove", []interface{}{"token:" + d.secret, ariaGID})
			return "", ctx.Err()
		case <-ticker.C:
			st, err := d.callRPC(ctx, "aria2.tellStatus", []interface{}{"token:" + d.secret, ariaGID})
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

			if status == "complete" {
				// Find downloaded file or directory
				files, _ := os.ReadDir(taskDir)
				for _, f := range files {
					if !f.IsDir() || len(files) == 1 {
						t.SetName(f.Name())
						return filepath.Join(taskDir, f.Name()), nil
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

func (d *Aria2Downloader) callRPC(ctx context.Context, method string, params []interface{}) (map[string]interface{}, error) {
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
