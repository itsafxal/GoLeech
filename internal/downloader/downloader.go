package downloader

import (
	"context"
	"strings"

	"github.com/GoLeech/GoLeech/internal/task"
)

type ProgressCallback func(current, total int64, speed float64)

type Downloader interface {
	Download(ctx context.Context, t *task.Task, onProgress ProgressCallback) (string, error)
}

// DetectEngine selects the appropriate downloader engine name for an input link
func DetectEngine(link string) string {
	linkLower := strings.ToLower(link)
	if strings.HasPrefix(linkLower, "magnet:?") || strings.HasSuffix(linkLower, ".torrent") {
		return "Torrent"
	}
	if strings.Contains(linkLower, "t.me/") || strings.HasPrefix(linkLower, "tg://") {
		return "Telegram"
	}
	if strings.Contains(linkLower, "youtube.com/") || strings.Contains(linkLower, "youtu.be/") ||
		strings.Contains(linkLower, "instagram.com/") || strings.Contains(linkLower, "tiktok.com/") ||
		strings.Contains(linkLower, "twitter.com/") || strings.Contains(linkLower, "x.com/") {
		return "yt-dlp"
	}
	if strings.Contains(linkLower, "drive.google.com") {
		return "GDrive"
	}
	if strings.Contains(linkLower, "mega.nz") {
		return "Mega"
	}
	return "HTTP"
}
