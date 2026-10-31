package torrent

import (
	"context"

	"github.com/GoLeech/GoLeech/internal/downloader"
	"github.com/GoLeech/GoLeech/internal/downloader/aria2"
	"github.com/GoLeech/GoLeech/internal/task"
)

type Aria2Downloader struct {
	d *aria2.Downloader
}

func NewAria2Downloader(rpcURL, secret string) *Aria2Downloader {
	return &Aria2Downloader{
		d: aria2.NewDownloader(rpcURL, secret),
	}
}

func (d *Aria2Downloader) Download(ctx context.Context, t *task.Task, onProgress downloader.ProgressCallback) (string, error) {
	return d.d.Download(ctx, t, onProgress)
}
