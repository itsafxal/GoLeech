package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/GoLeech/GoLeech/internal/downloader"
)

type UphosterUploader struct {
	client *http.Client
}

func NewUphosterUploader() *UphosterUploader {
	return &UphosterUploader{
		client: &http.Client{Timeout: 0},
	}
}

// UploadToPixeldrain uploads a file to Pixeldrain API
func (u *UphosterUploader) UploadToPixeldrain(ctx context.Context, filePath string, onProgress downloader.ProgressCallback) (*UploadResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, _ := file.Stat()
	totalSize := info.Size()

	req, err := http.NewRequestWithContext(ctx, "PUT", "https://pixeldrain.com/api/file/"+filepath.Base(filePath), file)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return nil, fmt.Errorf("pixeldrain upload failed with status: %s", resp.Status)
	}

	var res struct {
		ID      string `json:"id"`
		Success bool   `json:"success"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	downloadURL := fmt.Sprintf("https://pixeldrain.com/u/%s", res.ID)
	return &UploadResult{
		URL:      downloadURL,
		FileSize: totalSize,
	}, nil
}

// UploadToGofile uploads a file to Gofile API
func (u *UphosterUploader) UploadToGofile(ctx context.Context, filePath string, onProgress downloader.ProgressCallback) (*UploadResult, error) {
	// 1. Get best server
	reqSrv, _ := http.NewRequestWithContext(ctx, "GET", "https://api.gofile.io/servers", nil)
	respSrv, err := u.client.Do(reqSrv)
	if err != nil {
		return nil, err
	}
	defer respSrv.Body.Close()

	var srvData struct {
		Status string `json:"status"`
		Data   struct {
			Servers []struct {
				Name string `json:"name"`
			} `json:"servers"`
		} `json:"data"`
	}
	_ = json.NewDecoder(respSrv.Body).Decode(&srvData)

	serverName := "store1"
	if len(srvData.Data.Servers) > 0 {
		serverName = srvData.Data.Servers[0].Name
	}

	// 2. Upload file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}
	_ = writer.Close()

	upURL := fmt.Sprintf("https://%s.gofile.io/contents/uploadfile", serverName)
	reqUp, err := http.NewRequestWithContext(ctx, "POST", upURL, body)
	if err != nil {
		return nil, err
	}
	reqUp.Header.Set("Content-Type", writer.FormDataContentType())

	respUp, err := u.client.Do(reqUp)
	if err != nil {
		return nil, err
	}
	defer respUp.Body.Close()

	var resUp struct {
		Status string `json:"status"`
		Data   struct {
			DownloadPage string `json:"downloadPage"`
		} `json:"data"`
	}
	if err := json.NewDecoder(respUp.Body).Decode(&resUp); err != nil {
		return nil, err
	}

	return &UploadResult{
		URL: resUp.Data.DownloadPage,
	}, nil
}
