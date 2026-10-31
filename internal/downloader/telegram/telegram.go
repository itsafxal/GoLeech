package telegram

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/downloader"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
	tg "github.com/amarnathcjd/gogram/telegram"
)

type Downloader struct {
	client     *tg.Client
	userClient *tg.Client
}

func NewDownloader(client, userClient *tg.Client) *Downloader {
	return &Downloader{
		client:     client,
		userClient: userClient,
	}
}

// DownloadMedia downloads a Telegram message's attached media directly via MTProto
func (d *Downloader) DownloadMedia(ctx context.Context, msg *tg.NewMessage, t *task.Task, onProgress downloader.ProgressCallback) (string, error) {
	if msg == nil || !msg.IsMedia() {
		return "", fmt.Errorf("message does not contain any downloadable media")
	}

	cfg := config.Get()
	taskDir := filepath.Join(cfg.DownloadDir, t.GID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}
	t.LocalPath = taskDir

	filename := t.Options.Name
	var fileSize int64
	if doc := msg.Document(); doc != nil {
		fileSize = doc.Size
		if filename == "" {
			for _, attr := range doc.Attributes {
				if fName, ok := attr.(*tg.DocumentAttributeFilename); ok {
					filename = fName.FileName
					break
				}
			}
		}
	}
	if fileSize > 0 {
		t.TotalSize = fileSize
	}

	if filename == "" {
		filename = fmt.Sprintf("tg_%d_%d.bin", msg.ChatID(), msg.ID)
	}

	filename = utils.CleanMediaFilename(filename, "")
	t.SetName(filename)
	destPath := filepath.Join(taskDir, filename)

	// Gogram native MTProto download with progress callback and context
	opts := &tg.DownloadOptions{
		FileName: destPath,
		Threads:  4, // 4 concurrent MTProto chunk downloaders
		Ctx:      ctx,
		ProgressCallback: func(p *tg.ProgressInfo) {
			if onProgress != nil && p != nil {
				onProgress(p.Current, p.TotalSize, p.CurrentSpeed)
			}
		},
	}

	var savedPath string
	var err error

	// If file is > 2GB (Premium size) and user client is connected, prefer user client
	if fileSize > 2097152000 && d.userClient != nil {
		savedPath, err = d.userClient.DownloadMedia(msg.Media, opts)
	} else {
		savedPath, err = msg.Download(opts)
		if err != nil && d.userClient != nil {
			// Fallback to user client if bot download hit restriction
			savedPath, err = d.userClient.DownloadMedia(msg.Media, opts)
		}
	}

	if err != nil {
		return "", fmt.Errorf("telegram download failed: %w", err)
	}

	return savedPath, nil
}
