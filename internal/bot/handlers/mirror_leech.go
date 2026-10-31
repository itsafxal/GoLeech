package handlers

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
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
	if !opts.AsDocument && !opts.AsMedia {
		if db.GlobalStore != nil {
			if st, _ := db.GlobalStore.GetUserSettings(senderID); st != nil {
				opts.AsDocument = st.AsDoc
			} else {
				opts.AsDocument = cfg.LeechAsDocument
			}
		} else {
			opts.AsDocument = cfg.LeechAsDocument
		}
	}

	if db.GlobalStore != nil {
		if st, _ := db.GlobalStore.GetUserSettings(senderID); st != nil {
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

	// Send initial confirmation with interactive queue card
	initialText, initialMarkup := RenderQueueTaskMessage(t.GetSnapshot(), "")
	notifyMsg, err := ui.Reply(m, initialText, &tg.SendOptions{
		ReplyMarkup: initialMarkup,
	})
	if err == nil && notifyMsg != nil {
		t.SetQueueMessage(m.ChatID(), notifyMsg.ID)
		t.MessageID = notifyMsg.ID
	}

	// Launch async task pipeline in dedicated goroutine
	go runTaskPipeline(t, m, notifyMsg, replyMsg, botClient, userClient)

	return nil
}

func runTaskPipeline(t *task.Task, origMsg, notifyMsg, replyMsg *tg.NewMessage, botClient, userClient *tg.Client) {
	var lastEditedText string
	updateQueueCard := func(force bool) {
		if notifyMsg == nil {
			return
		}
		snap := t.GetSnapshot()
		text, markup := RenderQueueTaskMessage(snap, "")
		if !force && text == lastEditedText {
			return
		}
		_, err := ui.EditMsg(notifyMsg, text, &tg.SendOptions{
			ReplyMarkup: markup,
		})
		if err == nil {
			lastEditedText = text
		}
	}

	// Start live progress updater ticker (updates every 2.5 seconds during active transfer)
	stopTicker := make(chan struct{})
	var tickerWg sync.WaitGroup
	tickerWg.Add(1)

	go func() {
		defer tickerWg.Done()
		ticker := time.NewTicker(2500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopTicker:
				return
			case <-ticker.C:
				st := t.GetStatus()
				if st == task.StatusCompleted || st == task.StatusFailed || st == task.StatusCancelled {
					return
				}
				updateQueueCard(false)
			}
		}
	}()

	finalize := func(cloudURL string) {
		close(stopTicker)
		tickerWg.Wait()

		snap := t.GetSnapshot()
		finalText, finalMarkup := RenderQueueTaskMessage(snap, cloudURL)
		if notifyMsg != nil {
			_, err := ui.EditMsg(notifyMsg, finalText, &tg.SendOptions{
				ReplyMarkup: finalMarkup,
			})
			if err != nil {
				_, _ = ui.Reply(origMsg, finalText, &tg.SendOptions{
					ReplyMarkup: finalMarkup,
				})
			}
		} else {
			_, _ = ui.Reply(origMsg, finalText, &tg.SendOptions{
				ReplyMarkup: finalMarkup,
			})
		}
	}

	defer func() {
		task.GlobalManager.RemoveTask(t.GID)
		if r := recover(); r != nil {
			t.SetError(fmt.Errorf("panic in task pipeline: %v", r))
			finalize("")
		}
	}()

	// 1. Acquire Download Slot
	if err := task.GlobalManager.AcquireDownloadSlot(t); err != nil {
		t.SetError(err)
		finalize("")
		return
	}
	defer task.GlobalManager.ReleaseDownloadSlot()

	t.SetStatus(task.StatusDownloading)
	engineName := downloader.DetectEngine(t.Options.Link)
	t.SetEngine(engineName)
	updateQueueCard(true)

	var downloadedPath string
	var dlErr error

	progressCb := func(cur, tot int64, spd float64) {
		t.UpdateProgress(cur, tot, spd)
	}

	// 2. Execute Download based on engine
	switch {
	case t.Options.Link == "tg://media" && replyMsg != nil && replyMsg.IsMedia():
		t.SetEngine("Telegram")
		tgDL := dlTG.NewDownloader(botClient, userClient)
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
		finalize("")
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
		updateQueueCard(true)
		extractDir := filepath.Join(filepath.Dir(downloadedPath), "extracted")
		if err := media.ExtractArchive(t.Ctx, downloadedPath, extractDir); err == nil {
			workPath = extractDir
		}
	}

	// Zip archive if requested (-z)
	if t.Options.Zip {
		t.SetStatus(task.StatusArchiving)
		updateQueueCard(true)
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
		updateQueueCard(true)
		if taggedPath, err := media.ApplyMetadata(t.Ctx, workPath, globalMeta, audioMeta, videoMeta, subMeta); err == nil {
			workPath = taggedPath
		}
	}

	// 4. Acquire Upload Slot
	if err := task.GlobalManager.AcquireUploadSlot(t); err != nil {
		t.SetError(err)
		finalize("")
		return
	}
	defer task.GlobalManager.ReleaseUploadSlot()

	t.SetStatus(task.StatusUploading)
	updateQueueCard(true)

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
		finalize("")
		return
	}

	t.SetStatus(task.StatusCompleted)
	cloudURL := ""
	if uploadRes != nil && uploadRes.URL != "" {
		cloudURL = uploadRes.URL
	}
	finalize(cloudURL)
}
