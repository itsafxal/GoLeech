package providers

import (
	"context"

	"github.com/GoLeech/GoLeech/internal/downloader"
	"github.com/GoLeech/GoLeech/internal/task"
)

type UploadResult struct {
	URL       string `json:"url"`
	MessageID int32  `json:"message_id"`
	FileSize  int64  `json:"file_size"`
}

type Uploader interface {
	Upload(ctx context.Context, t *task.Task, filePath string, onProgress downloader.ProgressCallback) (*UploadResult, error)
}
