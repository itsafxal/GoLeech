package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

	// Split size check adhering to active client limits (2GB Standard Bot / 4GB Premium User)
	maxClientLimit := int64(2097152000)
	if u.userClient != nil {
		maxClientLimit = 4194304000
	}

	splitLimit := maxClientLimit
	if cfg.LeechSplitSize > 0 && cfg.LeechSplitSize <= maxClientLimit {
		splitLimit = cfg.LeechSplitSize
	}
	if uSettings != nil && uSettings.SplitSize > 0 && uSettings.SplitSize <= maxClientLimit {
		splitLimit = uSettings.SplitSize
	}
	if t.Options.SplitSize > 0 && t.Options.SplitSize <= maxClientLimit {
		splitLimit = t.Options.SplitSize
	}
	if splitLimit <= 0 || splitLimit > maxClientLimit {
		splitLimit = maxClientLimit
	}

	equalSplits := false
	if uSettings != nil && uSettings.Get("EQUAL_SPLITS") == "true" {
		equalSplits = true
	}

	// 1. Apply Prefix and Suffix to file before splitting/uploading
	prefix := ""
	suffix := ""
	if uSettings != nil {
		prefix = uSettings.Prefix
		suffix = uSettings.Suffix
	}
	if prefix == "" && cfg != nil {
		prefix = cfg.LeechPrefix
	}
	if suffix == "" && cfg != nil {
		suffix = cfg.LeechSuffix
	}

	if prefix != "" || suffix != "" {
		if fi, err := os.Stat(filePath); err == nil && !fi.IsDir() {
			dir := filepath.Dir(filePath)
			origName := filepath.Base(filePath)
			newName := utils.ApplyPrefixSuffix(origName, prefix, suffix)
			if newName != origName {
				newPath := filepath.Join(dir, newName)
				if err := os.Rename(filePath, newPath); err == nil {
					filePath = newPath
					t.SetName(newName)
				}
			}
		}
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

		isDoc := true
		if t.Options.AsMedia {
			isDoc = false
		} else if t.Options.AsDocument {
			isDoc = true
		} else if uSettings != nil {
			isDoc = uSettings.AsDoc
		}

		partName := filepath.Base(partPath)

		// Probe video metadata if file is a video container (for attributes and dynamic caption variables)
		ext := strings.ToLower(filepath.Ext(partPath))
		isVideoExt := (ext == ".mp4" || ext == ".mkv" || ext == ".avi" || ext == ".mov" || ext == ".webm" || ext == ".flv" || ext == ".wmv")
		var meta *media.VideoMetadata
		if isVideoExt {
			meta, _ = media.GetVideoMetadata(ctx, partPath)
		}

		caption := buildCaption(partName, info.Size(), idx+1, len(parts), t, uSettings, cfg, meta)

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

		isVideo := !isDoc && isVideoExt

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

		// Use user client if available and premium file size (> 2GB), else standard bot client
		clientToUse := u.client
		if u.userClient != nil && info.Size() > 2097152000 {
			clientToUse = u.userClient
		}

		sentMsg, err := clientToUse.SendMedia(targetChatID, partPath, mediaOpts)
		if err != nil && clientToUse == u.userClient {
			// Fallback: If user client cannot send directly to targetChatID (e.g. peer restriction in private chats),
			// upload to dump channel or log chat and forward via the bot client
			dumpTarget := cfg.LeechLogChat
			if t.Options.DumpChat != "" {
				if dID, dErr := strconv.ParseInt(t.Options.DumpChat, 10, 64); dErr == nil {
					dumpTarget = dID
				}
			}
			if dumpTarget != 0 {
				dumpMsg, dErr := u.userClient.SendMedia(dumpTarget, partPath, mediaOpts)
				if dErr == nil && dumpMsg != nil {
					fwdMsgs, fwdErr := u.client.Forward(targetChatID, dumpTarget, []int32{dumpMsg.ID})
					if fwdErr == nil && len(fwdMsgs) > 0 {
						sentMsg = &fwdMsgs[0]
						err = nil
					}
				}
			}
		}

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

func buildCaption(filename string, size int64, partNum, totalParts int, t *task.Task, uSettings *db.UserSettings, cfg *config.Config, meta *media.VideoMetadata) string {
	userCaption := ""
	if uSettings != nil && uSettings.Caption != "" {
		userCaption = uSettings.Caption
	} else if cfg != nil && cfg.LeechCaption != "" {
		userCaption = cfg.LeechCaption
	}

	var width, height int
	var durationSec int64
	if meta != nil {
		width = meta.Width
		height = meta.Height
		durationSec = int64(meta.Duration)
	}

	return utils.FormatCaption(userCaption, filename, size, partNum, totalParts, t.UserName, fmt.Sprintf("%d", t.UserID), width, height, durationSec)
}
