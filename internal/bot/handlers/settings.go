package handlers

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/bot/ui"
	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/typography"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// HandleUserSettings opens the interactive user preferences panel
func HandleUserSettings(m *tg.NewMessage) error {
	senderID := m.SenderID()
	text, markup := GetUserSettingsFull(senderID, "main")

	_, err := ui.Reply(m, text, &tg.SendOptions{
		ReplyMarkup: markup,
	})
	return err
}

// RenderUserSettings generates the user settings card and inline keyboard
func RenderUserSettings(userID int64) (string, tg.ReplyMarkup) {
	return GetUserSettingsFull(userID, "main")
}

// HandleBotSettings opens the runtime control center for Sudo administrators
func HandleBotSettings(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := ui.Reply(m, "⛔️ Only Sudo administrators can access bot configuration.")
		return err
	}

	text, markup := RenderBotSettings()
	_, err := ui.Reply(m, text, &tg.SendOptions{
		ReplyMarkup: markup,
	})
	return err
}

// RenderBotSettings generates the interactive dashboard for bot admins
func RenderBotSettings() (string, tg.ReplyMarkup) {
	cfg := config.Get()

	leechStatus := "✅ Enabled"
	if cfg.DisableLeech {
		leechStatus = "❌ Disabled"
	}

	mirrorStatus := "✅ Enabled"
	if cfg.DisableMirror {
		mirrorStatus = "❌ Disabled"
	}

	torrentStatus := "✅ Enabled"
	if cfg.DisableTorrents {
		torrentStatus = "❌ Disabled"
	}

	rssStatus := "✅ Enabled"
	if cfg.DisableRSS {
		rssStatus = "❌ Disabled"
	}

	searchStatus := "✅ Enabled"
	if cfg.DisableSearch {
		searchStatus = "❌ Disabled"
	}

	text := fmt.Sprintf(`⌬ <b><u>GoLeech — System Control Dashboard</u></b>
│
┠ <b>Leech Engine:</b> <b>%s</b>
┠ <b>Mirror Engine:</b> <b>%s</b>
┠ <b>Torrent Client:</b> <b>%s</b>
┠ <b>RSS Monitor:</b> <b>%s</b>
┠ <b>Torrent Search:</b> <b>%s</b>
│
┠ <b>Concurrency Limits:</b>
┃ ┠ <b>Queue All:</b> <code>%d</code>
┃ ┠ <b>Queue DL:</b> <code>%d</code>
┃ ┖ <b>Queue UP:</b> <code>%d</code>
│
┠ <b>Status Settings:</b>
┃ ┠ <b>Page Limit:</b> <code>%d</code> tasks/page
┃ ┖ <b>Refresh Interval:</b> <code>%ds</code>
│
┠ <b>Network & Services:</b>
┃ ┠ <b>Web Port:</b> <code>%d</code>
┃ ┖ <b>Leech Log Chat:</b> <code>%d</code>
│
┖ <i>Tap toggles below to dynamically adjust live parameters:</i>`,
		leechStatus,
		mirrorStatus,
		torrentStatus,
		rssStatus,
		searchStatus,
		cfg.QueueAll,
		cfg.QueueDownload,
		cfg.QueueUpload,
		cfg.StatusLimit,
		cfg.StatusUpdateInterval,
		cfg.WebPort,
		cfg.LeechLogChat,
	)

	markup := tg.Button.Keyboard(
		tg.Button.Row(
			tg.Button.Data("Leech: "+leechStatus, "bsetting:toggle_leech"),
			tg.Button.Data("Mirror: "+mirrorStatus, "bsetting:toggle_mirror"),
		),
		tg.Button.Row(
			tg.Button.Data("Torrents: "+torrentStatus, "bsetting:toggle_torrents"),
			tg.Button.Data("RSS: "+rssStatus, "bsetting:toggle_rss"),
		),
		tg.Button.Row(
			tg.Button.Data("Search: "+searchStatus, "bsetting:toggle_search"),
			tg.Button.Data(fmt.Sprintf("Interval: %ds", cfg.StatusUpdateInterval), "bsetting:interval"),
		),
		tg.Button.Row(
			tg.Button.Data("Queue DL [-]", "bsetting:q_dl_dec"),
			tg.Button.Data(fmt.Sprintf("DL: %d", cfg.QueueDownload), "bsetting:refresh"),
			tg.Button.Data("Queue DL [+]", "bsetting:q_dl_inc"),
		),
		tg.Button.Row(
			tg.Button.Data("Queue UP [-]", "bsetting:q_up_dec"),
			tg.Button.Data(fmt.Sprintf("UP: %d", cfg.QueueUpload), "bsetting:refresh"),
			tg.Button.Data("Queue UP [+]", "bsetting:q_up_inc"),
		),
		tg.Button.Row(
			tg.Button.Data("♻️ Refresh", "bsetting:refresh"),
			tg.Button.Data("❌ Close", "bsetting:close"),
		),
	)

	return typography.ToSmallCapsHTML(text), typography.FormatMarkup(markup)
}

// HandleCallback processes all incoming inline keyboard callback queries
func HandleCallback(cb *tg.CallbackQuery, botClient *tg.Client) error {
	data := cb.DataString()
	senderID := cb.SenderID

	// 0. WZML-X Native User Settings & Gallery Callbacks
	if strings.HasPrefix(data, "userset ") {
		return HandleUsersetCallback(cb, botClient)
	}
	if strings.HasPrefix(data, "images ") {
		return HandleImageCallback(cb, botClient)
	}

	// 1. Task Cancellation & Force Start
	if strings.HasPrefix(data, "task:fs:") {
		gid := strings.TrimPrefix(data, "task:fs:")
		isSudo := auth.IsSudo(senderID)
		success, msg := task.GlobalManager.ForceStartTask(gid, senderID, isSudo)
		if success {
			_, _ = ui.Answer(cb, "⚡️ Task promoted to run immediately!", &tg.CallbackOptions{Alert: false})
		} else {
			_, _ = ui.Answer(cb, msg, &tg.CallbackOptions{Alert: true})
		}
		return nil
	}

	if strings.HasPrefix(data, "cancel:") {
		gid := strings.TrimPrefix(data, "cancel:")
		isSudo := auth.IsSudo(senderID)
		success, msg := task.GlobalManager.CancelTask(gid, senderID, isSudo)
		if success {
			_, _ = ui.Answer(cb, "Task cancelled successfully!", &tg.CallbackOptions{Alert: false})
			_, _ = ui.Edit(cb, "🛑 "+msg)
		} else {
			_, _ = ui.Answer(cb, msg, &tg.CallbackOptions{Alert: true})
		}
		return nil
	}

	// 2. Status Dashboard Navigation
	if strings.HasPrefix(data, "status:") {
		parts := strings.Split(data, ":")
		if len(parts) >= 2 {
			action := parts[1]
			switch action {
			case "refresh":
				text, markup := GenerateStatusMessage(1, "ALL")
				_, _ = ui.Edit(cb, text, &tg.SendOptions{ReplyMarkup: markup})
				_, _ = ui.Answer(cb, "Refreshed", &tg.CallbackOptions{})
			case "page":
				if len(parts) >= 3 {
					var p int
					fmt.Sscanf(parts[2], "%d", &p)
					text, markup := GenerateStatusMessage(p, "ALL")
					_, _ = ui.Edit(cb, text, &tg.SendOptions{ReplyMarkup: markup})
				}
				_, _ = cb.Answer("", &tg.CallbackOptions{})
			case "filter":
				if len(parts) >= 3 {
					flt := parts[2]
					text, markup := GenerateStatusMessage(1, flt)
					_, _ = ui.Edit(cb, text, &tg.SendOptions{ReplyMarkup: markup})
				}
				_, _ = cb.Answer("", &tg.CallbackOptions{})
			}
		}
		return nil
	}

	// 3. Tabbed Help Menu Navigation
	if strings.HasPrefix(data, "help:") {
		tab := strings.TrimPrefix(data, "help:")
		if tab == "close" {
			_, _ = cb.Delete()
			return nil
		}
		text, markup := GetHelpContent(tab)
		_, _ = ui.Edit(cb, text, &tg.SendOptions{
			ReplyMarkup: markup,
		})
		_, _ = cb.Answer("", &tg.CallbackOptions{})
		return nil
	}

	// 4. System Stats Callback
	if data == "stats:view" {
		text := GetStatsMessage()
		_, _ = ui.Edit(cb, text, &tg.SendOptions{
			ReplyMarkup: ui.Button.Keyboard(
				ui.Button.Row(
					ui.Button.Data("🔙 Back to Help", "help:main"),
					ui.Button.Data("❌ Close", "help:close"),
				),
			),
		})
		_, _ = cb.Answer("", &tg.CallbackOptions{})
		return nil
	}

	// 5. User Settings Callbacks
	if strings.HasPrefix(data, "usetting:") {
		parts := strings.Split(data, ":")
		action := parts[1]

		targetUID := senderID
		if len(parts) >= 3 {
			var uid int64
			if _, err := fmt.Sscanf(parts[2], "%d", &uid); err == nil && uid != 0 {
				targetUID = uid
			}
		}

		// Security: Users can only change their own settings unless Sudo
		if targetUID != senderID && !auth.IsSudo(senderID) {
			_, _ = ui.Answer(cb, "⛔️ You can only modify your own settings.", &tg.CallbackOptions{Alert: true})
			return nil
		}

		if action == "close" {
			_, _ = cb.Delete()
			return nil
		}

		if db.GlobalStore != nil {
			st, _ := db.GlobalStore.GetUserSettings(targetUID)
			if st == nil {
				st = &db.UserSettings{UserID: targetUID}
			}

			switch action {
			case "toggle_doc":
				st.AsDoc = !st.AsDoc
				_ = db.GlobalStore.SaveUserSettings(st)
				_, _ = ui.Answer(cb, "Leech format toggled!", &tg.CallbackOptions{})

			case "toggle_split":
				if st.SplitSize > 2147483648 {
					st.SplitSize = 2097152000 // 2GB
				} else {
					st.SplitSize = 4194304000 // 4GB
				}
				_ = db.GlobalStore.SaveUserSettings(st)
				_, _ = ui.Answer(cb, "Split size updated!", &tg.CallbackOptions{})

			case "del_thumb":
				if st.ThumbPath != "" {
					_ = os.Remove(st.ThumbPath)
					st.ThumbPath = ""
					_ = db.GlobalStore.SaveUserSettings(st)
					_, _ = ui.Answer(cb, "Thumbnail deleted!", &tg.CallbackOptions{})
				} else {
					_, _ = ui.Answer(cb, "No custom thumbnail configured.", &tg.CallbackOptions{})
				}

			case "view_thumb":
				if st.ThumbPath != "" {
					if _, err := os.Stat(st.ThumbPath); err == nil {
						_, _ = ui.Answer(cb, "Sending thumbnail...", &tg.CallbackOptions{})
						_, _ = cb.RespondMedia(st.ThumbPath, &tg.MediaOptions{
							Caption:   typography.ToSmallCapsHTML("🖼 <i>Your active custom leech thumbnail</i>"),
							ParseMode: "html",
						})
						return nil
					}
				}
				_, _ = ui.Answer(cb, "No custom thumbnail found. Send a photo in private chat to set one!", &tg.CallbackOptions{Alert: true})
				return nil

			case "info_prefix":
				_, _ = ui.Answer(cb, "To set prefix, send:\n/setprefix <text>\nTo clear: /setprefix", &tg.CallbackOptions{Alert: true})
				return nil
			case "info_suffix":
				_, _ = ui.Answer(cb, "To set suffix, send:\n/setsuffix <text>\nTo clear: /setsuffix", &tg.CallbackOptions{Alert: true})
				return nil
			case "info_caption":
				_, _ = ui.Answer(cb, "To set caption template, send:\n/setcaption <text>\nPlaceholders: {filename}, {size}, {duration}, {speed}\nTo clear: /setcaption", &tg.CallbackOptions{Alert: true})
				return nil
			case "info_dump":
				_, _ = ui.Answer(cb, "To set dump channel, send:\n/setdump <chat_id>\nTo clear: /setdump", &tg.CallbackOptions{Alert: true})
				return nil

			case "refresh":
				_, _ = ui.Answer(cb, "Refreshed", &tg.CallbackOptions{})
			}
		}

		text, markup := RenderUserSettings(targetUID)
		_, _ = ui.Edit(cb, text, &tg.SendOptions{
			ReplyMarkup: markup,
		})
		return nil
	}

	// 6. Bot Settings Callbacks (Sudo Only)
	if strings.HasPrefix(data, "bsetting:") {
		if !auth.IsSudo(senderID) {
			_, _ = ui.Answer(cb, "⛔️ Admin access required.", &tg.CallbackOptions{Alert: true})
			return nil
		}

		action := strings.TrimPrefix(data, "bsetting:")
		cfg := config.Get()

		switch action {
		case "close":
			_, _ = cb.Delete()
			return nil
		case "toggle_leech":
			cfg.DisableLeech = !cfg.DisableLeech
			_, _ = ui.Answer(cb, "Leech state toggled!", &tg.CallbackOptions{})
		case "toggle_mirror":
			cfg.DisableMirror = !cfg.DisableMirror
			_, _ = ui.Answer(cb, "Mirror state toggled!", &tg.CallbackOptions{})
		case "toggle_torrents":
			cfg.DisableTorrents = !cfg.DisableTorrents
			_, _ = ui.Answer(cb, "Torrent engine toggled!", &tg.CallbackOptions{})
		case "toggle_rss":
			cfg.DisableRSS = !cfg.DisableRSS
			_, _ = ui.Answer(cb, "RSS engine toggled!", &tg.CallbackOptions{})
		case "toggle_search":
			cfg.DisableSearch = !cfg.DisableSearch
			_, _ = ui.Answer(cb, "Search engine toggled!", &tg.CallbackOptions{})
		case "interval":
			if cfg.StatusUpdateInterval == 5 {
				cfg.StatusUpdateInterval = 10
			} else if cfg.StatusUpdateInterval == 10 {
				cfg.StatusUpdateInterval = 15
			} else {
				cfg.StatusUpdateInterval = 5
			}
			_, _ = ui.Answer(cb, fmt.Sprintf("Interval set to %ds", cfg.StatusUpdateInterval), &tg.CallbackOptions{})
		case "q_dl_inc":
			cfg.QueueDownload++
			_, _ = ui.Answer(cb, fmt.Sprintf("Queue DL: %d", cfg.QueueDownload), &tg.CallbackOptions{})
		case "q_dl_dec":
			if cfg.QueueDownload > 1 {
				cfg.QueueDownload--
			}
			_, _ = ui.Answer(cb, fmt.Sprintf("Queue DL: %d", cfg.QueueDownload), &tg.CallbackOptions{})
		case "q_up_inc":
			cfg.QueueUpload++
			_, _ = ui.Answer(cb, fmt.Sprintf("Queue UP: %d", cfg.QueueUpload), &tg.CallbackOptions{})
		case "q_up_dec":
			if cfg.QueueUpload > 1 {
				cfg.QueueUpload--
			}
			_, _ = ui.Answer(cb, fmt.Sprintf("Queue UP: %d", cfg.QueueUpload), &tg.CallbackOptions{})
		case "refresh":
			_, _ = ui.Answer(cb, "Refreshed", &tg.CallbackOptions{})
		}

		text, markup := RenderBotSettings()
		_, _ = ui.Edit(cb, text, &tg.SendOptions{
			ReplyMarkup: markup,
		})
		return nil
	}

	// 7. Process Restart Callback
	if data == "bot:restart_confirm" {
		if !auth.IsSudo(senderID) {
			_, _ = ui.Answer(cb, "⛔️ Sudo privileges required.", &tg.CallbackOptions{Alert: true})
			return nil
		}
		_, _ = ui.Answer(cb, "Restarting bot...", &tg.CallbackOptions{})
		_, _ = ui.Edit(cb, "🔄 <b>GoLeech is restarting gracefully...</b>")
		go func() {
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}()
		return nil
	}

	_, _ = ui.Answer(cb, "Action handled", &tg.CallbackOptions{})
	return nil
}

// HandleSetThumb sets custom leech thumbnail from replied photo or document
func HandleSetThumb(m *tg.NewMessage) error {
	senderID := m.SenderID()

	targetMsg := m
	if m.IsReply() {
		if r, err := m.GetReplyMessage(); err == nil && r != nil {
			targetMsg = r
		}
	}

	if targetMsg.Photo() == nil && !targetMsg.IsMedia() {
		_, err := ui.Reply(m, "⚠️ <b>Please reply to a photo or image</b> with <code>/setthumb</code>, or send a photo directly in private chat.")
		return err
	}

	thumbDir := filepath.Join("data", "thumbnails")
	_ = os.MkdirAll(thumbDir, 0755)
	targetFile := filepath.Join(thumbDir, fmt.Sprintf("%d.jpg", senderID))

	_, err := targetMsg.Download(&tg.DownloadOptions{FileName: targetFile})
	if err != nil {
		_, err = ui.Reply(m, "❌ Failed to download and save thumbnail: "+err.Error())
		return err
	}

	if db.GlobalStore != nil {
		st, _ := db.GlobalStore.GetUserSettings(senderID)
		if st == nil {
			st = &db.UserSettings{UserID: senderID}
		}
		st.ThumbPath = targetFile
		_ = db.GlobalStore.SaveUserSettings(st)
	}

	msg := `🖼 <b><u>Custom Leech Thumbnail Saved!</u></b>
│
┠ <b>File:</b> <code>Saved to persistent profile</code>
┠ <b>Status:</b> ✅ Active
│
┖ <i>This thumbnail will be embedded into all your future leeched media files!</i>`

	_, err = ui.Reply(m, msg, &tg.SendOptions{
		ReplyMarkup: ui.Button.Keyboard(
			ui.Button.Row(
				ui.Button.Data("⚙️ View Settings", fmt.Sprintf("usetting:refresh:%d", senderID)),
				ui.Button.Data("🗑 Delete Thumb", fmt.Sprintf("usetting:del_thumb:%d", senderID)),
			),
		),
	})
	return err
}

// HandleDelThumb deletes the active custom leech thumbnail
func HandleDelThumb(m *tg.NewMessage) error {
	senderID := m.SenderID()
	if db.GlobalStore != nil {
		st, _ := db.GlobalStore.GetUserSettings(senderID)
		if st != nil && st.ThumbPath != "" {
			_ = os.Remove(st.ThumbPath)
			st.ThumbPath = ""
			_ = db.GlobalStore.SaveUserSettings(st)
			_, err := ui.Reply(m, "🗑 <b>Custom thumbnail removed successfully.</b>")
			return err
		}
	}
	_, err := ui.Reply(m, "⚠️ You do not have an active custom thumbnail.")
	return err
}

// HandleSetPrefix sets or clears custom leech filename prefix
func HandleSetPrefix(m *tg.NewMessage) error {
	senderID := m.SenderID()
	prefix := strings.TrimSpace(m.Args())

	if db.GlobalStore != nil {
		st, _ := db.GlobalStore.GetUserSettings(senderID)
		if st == nil {
			st = &db.UserSettings{UserID: senderID}
		}
		st.Prefix = prefix
		_ = db.GlobalStore.SaveUserSettings(st)
	}

	if prefix == "" {
		_, err := ui.Reply(m, "🗑 <b>Custom leech prefix cleared.</b>")
		return err
	}

	_, err := ui.Reply(m, fmt.Sprintf("🏷 <b>Leech Prefix updated:</b> <code><raw>%s</raw></code>", html.EscapeString(prefix)))
	return err
}

// HandleSetSuffix sets or clears custom leech filename suffix
func HandleSetSuffix(m *tg.NewMessage) error {
	senderID := m.SenderID()
	suffix := strings.TrimSpace(m.Args())

	if db.GlobalStore != nil {
		st, _ := db.GlobalStore.GetUserSettings(senderID)
		if st == nil {
			st = &db.UserSettings{UserID: senderID}
		}
		st.Suffix = suffix
		_ = db.GlobalStore.SaveUserSettings(st)
	}

	if suffix == "" {
		_, err := ui.Reply(m, "🗑 <b>Custom leech suffix cleared.</b>")
		return err
	}

	_, err := ui.Reply(m, fmt.Sprintf("🏷 <b>Leech Suffix updated:</b> <code><raw>%s</raw></code>", html.EscapeString(suffix)))
	return err
}

// HandleSetCaption sets or clears custom leech caption template
func HandleSetCaption(m *tg.NewMessage) error {
	senderID := m.SenderID()
	caption := strings.TrimSpace(m.Args())

	if db.GlobalStore != nil {
		st, _ := db.GlobalStore.GetUserSettings(senderID)
		if st == nil {
			st = &db.UserSettings{UserID: senderID}
		}
		st.Caption = caption
		_ = db.GlobalStore.SaveUserSettings(st)
	}

	if caption == "" {
		_, err := ui.Reply(m, "🗑 <b>Custom caption reset to default.</b>")
		return err
	}

	msg := fmt.Sprintf(`📝 <b><u>Custom Leech Caption Saved!</u></b>
│
<blockquote expandable><raw>%s</raw></blockquote>
│
┖ <i>Available placeholders: {filename}, {size}, {duration}, {speed}</i>`, html.EscapeString(caption))

	_, err := ui.Reply(m, msg)
	return err
}

// HandleSetDump sets or clears the target leech dump channel
func HandleSetDump(m *tg.NewMessage) error {
	senderID := m.SenderID()
	arg := strings.TrimSpace(m.Args())

	if db.GlobalStore != nil {
		st, _ := db.GlobalStore.GetUserSettings(senderID)
		if st == nil {
			st = &db.UserSettings{UserID: senderID}
		}

		if arg == "" {
			st.DumpChat = 0
			_ = db.GlobalStore.SaveUserSettings(st)
			_, err := ui.Reply(m, "🗑 <b>Leech dump channel cleared.</b>")
			return err
		}

		id, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			_, err = ui.Reply(m, "❌ Invalid chat ID format. Example: <code>/setdump -1001234567890</code>")
			return err
		}

		st.DumpChat = id
		_ = db.GlobalStore.SaveUserSettings(st)
		_, err = ui.Reply(m, fmt.Sprintf("📺 <b>Leech dump channel set to:</b> <code>%d</code>", id))
		return err
	}

	_, err := ui.Reply(m, "Database unavailable.")
	return err
}
