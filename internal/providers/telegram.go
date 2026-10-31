package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/internal/downloader"
	"github.com/GoLeech/GoLeech/internal/media"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
	tg "github.com/amarnathcjd/gogram/telegram"
)

type TelegramUploader struct {
	client     *tg.Client
	userClient *tg.Client
}

func NewTelegramUploader(botClient, userClient *tg.Client) *TelegramUploader {
	return &TelegramUploader{
		client:     botClient,
		userClient: userClient,
	}
}

func (u *TelegramUploader) Upload(ctx context.Context, t *task.Task, filePath string, onProgress downloader.ProgressCallback) (*UploadResult, error) {
	cfg := config.Get()
	targetChatID := t.ChatID
	if t.Options.DumpChat != "" {
		// Could parse custom dump chat
	} else if cfg.LeechLogChat != 0 {
		targetChatID = cfg.LeechLogChat
	}

	// Fetch user settings
	var uSettings *db.UserSettings
	if db.GlobalStore != nil {
		uSettings, _ = db.GlobalStore.GetUserSettings(t.UserID)
	}

	// Split size check
	splitLimit := cfg.LeechSplitSize
	if u.userClient != nil {
		splitLimit = 4194304000 // 4GB for Telegram Premium User session
	}
	if t.Options.SplitSize > 0 {
		splitLimit = t.Options.SplitSize
	}

	equalSplits := false
	if uSettings != nil && uSettings.Get("EQUAL_SPLITS") == "true" {
		equalSplits = true
	}

	parts, err := media.SplitFileCustom(filePath, splitLimit, equalSplits)
	if err != nil {
		return nil, fmt.Errorf("splitting file failed: %w", err)
	}

	var lastMsgID int32
	var totalUploaded int64

	for idx, partPath := range parts {
		if t.IsCancelled() {
			return nil, fmt.Errorf("upload cancelled by user")
		}

		info, err := os.Stat(partPath)
		if err != nil {
			return nil, err
		}

		isDoc := t.Options.AsDocument
		if uSettings != nil && uSettings.AsDoc && !t.Options.AsMedia {
			isDoc = true
		}

		partName := filepath.Base(partPath)
		caption := buildCaption(partName, info.Size(), idx+1, len(parts), t, uSettings, cfg)

		// Upload options with progress tracking
		upOpts := &tg.UploadOptions{
			FileName:  partName,
			Threads:   4,
			ChunkSize: 512 * 1024, // 512 KB chunks for high throughput
			Ctx:       ctx,
			ProgressCallback: func(p *tg.ProgressInfo) {
				if onProgress != nil && p != nil {
					onProgress(totalUploaded+p.Current, t.TotalSize, p.CurrentSpeed)
				}
			},
		}

		mediaOpts := &tg.MediaOptions{
			Caption:   caption,
			ParseMode: "html",
			Upload:    upOpts,
		}

		// Handle video metadata and thumbnail if sending as video
		ext := strings.ToLower(filepath.Ext(partPath))
		isVideo := !isDoc && (ext == ".mp4" || ext == ".mkv" || ext == ".avi" || ext == ".mov" || ext == ".webm")

		// Apply custom thumbnail if available
		customThumb := t.Options.CustomThumb
		if customThumb == "" && uSettings != nil {
			customThumb = uSettings.ThumbPath
		}
		if customThumb != "" {
			if _, err := os.Stat(customThumb); err == nil {
				mediaOpts.Thumb = customThumb
			}
		}

		if isVideo {
			meta, _ := media.GetVideoMetadata(ctx, partPath)
			if mediaOpts.Thumb == nil {
				thumbPath, _ := media.GenerateThumbnail(ctx, partPath, filepath.Dir(partPath))
				if thumbPath != "" {
					mediaOpts.Thumb = thumbPath
				}
			}
			if meta != nil {
				mediaOpts.Attributes = append(mediaOpts.Attributes, &tg.DocumentAttributeVideo{
					Duration:          meta.Duration,
					W:                 int32(meta.Width),
					H:                 int32(meta.Height),
					SupportsStreaming: true,
				})
			}
		} else {
			mediaOpts.ForceDocument = true
		}

		// Use user client if available and premium file size, else bot client
		clientToUse := u.client
		if u.userClient != nil && info.Size() > 2097152000 {
			clientToUse = u.userClient
		}

		sentMsg, err := clientToUse.SendMedia(targetChatID, partPath, mediaOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to upload part %s to telegram: %w", partName, err)
		}

		if sentMsg != nil {
			lastMsgID = sentMsg.ID
		}
		totalUploaded += info.Size()
	}

	return &UploadResult{
		MessageID: lastMsgID,
		FileSize:  totalUploaded,
	}, nil
}

func buildCaption(filename string, size int64, partNum, totalParts int, t *task.Task, uSettings *db.UserSettings, cfg *config.Config) string {
	var prefix, suffix, userCaption string
	if uSettings != nil {
		prefix = uSettings.Prefix
		suffix = uSettings.Suffix
		userCaption = uSettings.Caption
	}
	if prefix == "" {
		prefix = cfg.LeechPrefix
	}
	if suffix == "" {
		suffix = cfg.LeechSuffix
	}
	if userCaption == "" {
		userCaption = cfg.LeechCaption
	}

	var sb strings.Builder
	if prefix != "" {
		sb.WriteString("<b>" + prefix + "</b> ")
	}
	sb.WriteString("<code>" + filename + "</code>")
	if suffix != "" {
		sb.WriteString(" <b>" + suffix + "</b>")
	}

	sb.WriteString("\n\n┠ <b>Size:</b> " + utils.FormatBytes(size))
	if totalParts > 1 {
		sb.WriteString(fmt.Sprintf("\n┠ <b>Part:</b> %d / %d", partNum, totalParts))
	}
	if t.UserName != "" {
		sb.WriteString("\n┠ <b>By:</b> @" + t.UserName)
	}

	if userCaption != "" {
		sb.WriteString("\n\n" + userCaption)
	}

	return sb.String()
}
