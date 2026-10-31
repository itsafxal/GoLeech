package handlers

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/bot/ui"
	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/internal/downloader"
	dlHTTP "github.com/GoLeech/GoLeech/internal/downloader/http"
	dlTG "github.com/GoLeech/GoLeech/internal/downloader/telegram"
	dlTor "github.com/GoLeech/GoLeech/internal/downloader/torrent"
	dlYT "github.com/GoLeech/GoLeech/internal/downloader/ytdlp"
	"github.com/GoLeech/GoLeech/internal/media"
	"github.com/GoLeech/GoLeech/internal/providers"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func HandleMirror(m *tg.NewMessage, botClient, userClient *tg.Client) error {
	return processTask(m, task.TypeMirror, botClient, userClient)
}

func HandleLeech(m *tg.NewMessage, botClient, userClient *tg.Client) error {
	return processTask(m, task.TypeLeech, botClient, userClient)
}

func processTask(m *tg.NewMessage, tType task.TaskType, botClient, userClient *tg.Client) error {
	senderID := m.SenderID()
	chatID := m.ChatID()

	if !auth.IsAuthorized(senderID, chatID) {
		_, err := ui.Reply(m, "⚠️ <b>Unauthorized:</b> You do not have permission to execute this command.")
		return err
	}

	cfg := config.Get()
	if tType == task.TypeLeech && cfg.DisableLeech {
		_, err := ui.Reply(m, "⚠️ Leech functionality is currently disabled.")
		return err
	}
	if tType == task.TypeMirror && cfg.DisableMirror {
		_, err := ui.Reply(m, "⚠️ Mirror functionality is currently disabled.")
		return err
	}

	opts := utils.ParseTaskOptions(m.Args())

	// Apply user personal defaults if flags not explicitly passed
	if db.GlobalStore != nil {
		if st, _ := db.GlobalStore.GetUserSettings(senderID); st != nil {
			if !opts.AsDocument && !opts.AsMedia && st.AsDoc {
				opts.AsDocument = true
			}
			if opts.SplitSize == 0 && st.SplitSize > 0 {
				opts.SplitSize = st.SplitSize
			}
			if opts.CustomThumb == "" && st.ThumbPath != "" {
				opts.CustomThumb = st.ThumbPath
			}
			if opts.DumpChat == "" && st.DumpChat != 0 {
				opts.DumpChat = fmt.Sprintf("%d", st.DumpChat)
			}
			if opts.UploadDest == "" && st.UploadDest != "" {
				opts.UploadDest = st.UploadDest
			}
		}
	}

	// Check if replied message contains media or links
	replyMsg, _ := m.GetReplyMessage()
	if replyMsg != nil {
		if replyMsg.IsMedia() {
			opts.Link = "tg://media"
		} else if opts.Link == "" && replyMsg.Text() != "" {
			opts.Link = strings.TrimSpace(replyMsg.Text())
		}
	}

	if opts.Link == "" {
		_, err := ui.Reply(m, fmt.Sprintf("⚠️ <b>Usage:</b> <code>/%s [link/flags]</code> or reply to a link/file.", strings.ToLower(string(tType))))
		return err
	}

	// Create and register new task
	userName := ""
	if sender, err := m.GetSender(); err == nil && sender != nil {
		userName = sender.Username
	}

	t := task.NewTask(senderID, userName, chatID, m.ID, tType, opts)
	task.GlobalManager.AddTask(t)

	// Send initial confirmation
	notifyMsg, err := ui.Reply(m, fmt.Sprintf("⏳ <b>Task Added to Queue:</b>\n┠ <b>GID:</b> <code>%s</code>\n┠ <b>Type:</b> <i>%s</i>\n┖ <b>Name:</b> <code><raw>%s</raw></code>", t.GID, t.Type, t.Name), &tg.SendOptions{
		ReplyMarkup: ui.Button.Keyboard(
			ui.Button.Row(
				ui.Button.Data("🚫 Cancel Task", fmt.Sprintf("cancel:%s", t.GID)),
			),
		),
	})
	if err == nil && notifyMsg != nil {
		t.MessageID = notifyMsg.ID
	}

	// Launch async task pipeline in dedicated goroutine
	go runTaskPipeline(t, m, replyMsg, botClient, userClient)

	return nil
}

func runTaskPipeline(t *task.Task, origMsg, replyMsg *tg.NewMessage, botClient, userClient *tg.Client) {
	defer func() {
		task.GlobalManager.RemoveTask(t.GID)
		if r := recover(); r != nil {
			t.SetError(fmt.Errorf("panic in task pipeline: %v", r))
		}
	}()

	// 1. Acquire Download Slot
	if err := task.GlobalManager.AcquireDownloadSlot(t); err != nil {
		t.SetError(err)
		_, _ = ui.Reply(origMsg, fmt.Sprintf("❌ Task <code>%s</code> failed to start: <raw>%v</raw>", t.GID, err))
		return
	}
	defer task.GlobalManager.ReleaseDownloadSlot()

	t.SetStatus(task.StatusDownloading)
	engineName := downloader.DetectEngine(t.Options.Link)
	t.SetEngine(engineName)

	var downloadedPath string
	var dlErr error

	progressCb := func(cur, tot int64, spd float64) {
		t.UpdateProgress(cur, tot, spd)
	}

	// 2. Execute Download based on engine
	switch {
	case t.Options.Link == "tg://media" && replyMsg != nil && replyMsg.IsMedia():
		t.SetEngine("Telegram")
		tgDL := dlTG.NewDownloader(botClient)
		downloadedPath, dlErr = tgDL.DownloadMedia(t.Ctx, replyMsg, t, progressCb)

	case engineName == "Torrent":
		t.SetEngine("Torrent")
		torDL := dlTor.NewAria2Downloader("", "")
		downloadedPath, dlErr = torDL.Download(t.Ctx, t, progressCb)

	case engineName == "yt-dlp":
		t.SetEngine("yt-dlp")
		ytDL := dlYT.NewDownloader()
		downloadedPath, dlErr = ytDL.Download(t.Ctx, t, progressCb)

	default:
		t.SetEngine("HTTP")
		httpDL := dlHTTP.NewDownloader()
		downloadedPath, dlErr = httpDL.Download(t.Ctx, t, progressCb)
	}

	if dlErr != nil {
		t.SetError(dlErr)
		_, _ = ui.Reply(origMsg, fmt.Sprintf("❌ Download failed for <code><raw>%s</raw></code>:\n<code><raw>%v</raw></code>", t.Name, dlErr))
		return
	}

	// 3. Post-Download Media Processing
	workPath := downloadedPath

	var uSettings *db.UserSettings
	if db.GlobalStore != nil {
		uSettings, _ = db.GlobalStore.GetUserSettings(t.UserID)
	}

	// Apply Name Swap (-ns or user setting)
	nameSwapRule := t.Options.NameSwap
	if nameSwapRule == "" && uSettings != nil {
		nameSwapRule = uSettings.Get("NAME_SWAP")
	}
	if nameSwapRule != "" {
		newName := media.ApplyNameSwap(t.Name, nameSwapRule)
		if newName != t.Name {
			t.SetName(newName)
		}
	}

	// Extract archive if requested (-e)
	if t.Options.Extract {
		t.SetStatus(task.StatusExtracting)
		extractDir := filepath.Join(filepath.Dir(downloadedPath), "extracted")
		if err := media.ExtractArchive(t.Ctx, downloadedPath, extractDir); err == nil {
			workPath = extractDir
		}
	}

	// Zip archive if requested (-z)
	if t.Options.Zip {
		t.SetStatus(task.StatusArchiving)
		zipPath := filepath.Join(filepath.Dir(downloadedPath), t.Name+".zip")
		if err := media.CreateZip(t.Ctx, downloadedPath, zipPath); err == nil {
			workPath = zipPath
		}
	}

	// Metadata stream tagging (-meta or user setting)
	globalMeta := t.Options.MetadataTitle
	if globalMeta == "" && uSettings != nil {
		globalMeta = uSettings.Get("METADATA")
	}
	audioMeta := ""
	videoMeta := ""
	subMeta := ""
	if uSettings != nil {
		audioMeta = uSettings.Get("AUDIO_METADATA")
		videoMeta = uSettings.Get("VIDEO_METADATA")
		subMeta = uSettings.Get("SUBTITLE_METADATA")
	}

	if globalMeta != "" || audioMeta != "" || videoMeta != "" || subMeta != "" {
		t.SetStatus(task.StatusProcessing)
		if taggedPath, err := media.ApplyMetadata(t.Ctx, workPath, globalMeta, audioMeta, videoMeta, subMeta); err == nil {
			workPath = taggedPath
		}
	}

	// 4. Acquire Upload Slot
	if err := task.GlobalManager.AcquireUploadSlot(t); err != nil {
		t.SetError(err)
		return
	}
	defer task.GlobalManager.ReleaseUploadSlot()

	t.SetStatus(task.StatusUploading)

	// 5. Upload based on task type
	cfg := config.Get()
	var uploadRes *providers.UploadResult
	var upErr error

	if t.Type == task.TypeLeech {
		t.SetEngine("Telegram")
		tgUploader := providers.NewTelegramUploader(botClient, userClient)
		uploadRes, upErr = tgUploader.Upload(t.Ctx, t, workPath, progressCb)
	} else {
		// Mirror task
		dest := cfg.DefaultUpload
		if t.Options.UploadDest != "" {
			dest = t.Options.UploadDest
		}

		if strings.HasPrefix(dest, "pixeldrain") {
			t.SetEngine("Pixeldrain")
			uphoster := providers.NewUphosterUploader()
			uploadRes, upErr = uphoster.UploadToPixeldrain(t.Ctx, workPath, progressCb)
		} else if strings.HasPrefix(dest, "gofile") {
			t.SetEngine("Gofile")
			uphoster := providers.NewUphosterUploader()
			uploadRes, upErr = uphoster.UploadToGofile(t.Ctx, workPath, progressCb)
		} else {
			// Rclone Cloud upload
			t.SetEngine("Rclone")
			rclone := providers.NewRcloneUploader()
			uploadRes, upErr = rclone.Upload(t.Ctx, t, workPath, progressCb)
		}
	}

	if upErr != nil {
		t.SetError(upErr)
		_, _ = ui.Reply(origMsg, fmt.Sprintf("❌ Upload failed for <code><raw>%s</raw></code>:\n<code><raw>%v</raw></code>", t.Name, upErr))
		return
	}

	t.SetStatus(task.StatusCompleted)

	// Send success completion message
	elapsed := time.Since(t.StartTime)
	completeText := fmt.Sprintf(`✅ <b>Task Completed!</b>
│
┠ <b>Name:</b> <code><raw>%s</raw></code>
┠ <b>Size:</b> <i>%s</i>
┠ <b>Type:</b> <i>%s</i>
┠ <b>Engine:</b> <i>%s</i>
┖ <b>Elapsed:</b> <i>%s</i>`, t.Name, utils.FormatBytes(t.TotalSize), t.Type, t.Engine, utils.FormatDuration(elapsed))

	if uploadRes != nil && uploadRes.URL != "" {
		completeText += fmt.Sprintf("\n\n🔗 <b>Cloud Link:</b> <raw>%s</raw>", uploadRes.URL)
	}

	_, _ = ui.Reply(origMsg, completeText)
}
