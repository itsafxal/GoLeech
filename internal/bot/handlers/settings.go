package handlers

import (
	"fmt"
	"strings"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func HandleUserSettings(m *tg.NewMessage) error {
	senderID := m.SenderID()
	var st *db.UserSettings
	if db.GlobalStore != nil {
		st, _ = db.GlobalStore.GetUserSettings(senderID)
	}
	if st == nil {
		st = &db.UserSettings{UserID: senderID}
	}

	asDocStr := "False (Media)"
	if st.AsDoc {
		asDocStr = "True (Document)"
	}

	splitSizeStr := "Default (2GB)"
	if st.SplitSize > 0 {
		splitSizeStr = utils.FormatBytes(st.SplitSize)
	}

	text := fmt.Sprintf(`⌬ <b><u>User Settings:</u></b> <code>#ID%d</code>
│
┠ <b>Leech as Doc:</b> <i>%s</i>
┠ <b>Leech Prefix:</b> <code>%s</code>
┠ <b>Leech Suffix:</b> <code>%s</code>
┠ <b>Leech Caption:</b> <code>%s</code>
┠ <b>Custom Split:</b> <i>%s</i>
┖ <b>Dump Chat:</b> <code>%d</code>`,
		senderID,
		asDocStr,
		st.Prefix,
		st.Suffix,
		st.Caption,
		splitSizeStr,
		st.DumpChat,
	)

	markup := tg.Button.Keyboard(
		tg.Button.Row(
			tg.Button.Data("Toggle Doc/Media", "usetting:toggle_doc"),
		),
		tg.Button.Row(
			tg.Button.Data("Close", "usetting:close"),
		),
	)

	_, err := m.Reply(text, &tg.SendOptions{
		ParseMode:   "html",
		ReplyMarkup: markup,
	})
	return err
}

func HandleBotSettings(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo users can view bot settings.", &tg.SendOptions{})
		return err
	}

	cfg := config.Get()
	text := fmt.Sprintf(`⌬ <b><u>GoLeech Bot Configuration</u></b>
│
┠ <b>Default Upload:</b> <code>%s</code>
┠ <b>Queue All:</b> <code>%d</code>
┠ <b>Queue DL:</b> <code>%d</code>
┠ <b>Queue UP:</b> <code>%d</code>
┠ <b>Status Limit:</b> <code>%d</code>
┠ <b>Status Interval:</b> <code>%ds</code>
┠ <b>Web Port:</b> <code>%d</code>
┠ <b>Leech Log Chat:</b> <code>%d</code>
┠ <b>Direct Limit:</b> <i>%s</i>
┖ <b>Torrent Limit:</b> <i>%s</i>`,
		cfg.DefaultUpload,
		cfg.QueueAll,
		cfg.QueueDownload,
		cfg.QueueUpload,
		cfg.StatusLimit,
		cfg.StatusUpdateInterval,
		cfg.WebPort,
		cfg.LeechLogChat,
		utils.FormatBytes(cfg.DirectLimit),
		utils.FormatBytes(cfg.TorrentLimit),
	)

	_, err := m.Reply(text, &tg.SendOptions{ParseMode: "html"})
	return err
}

// HandleCallback processes inline keyboard callback queries
func HandleCallback(cb *tg.CallbackQuery, botClient *tg.Client) error {
	data := cb.DataString()
	senderID := cb.SenderID

	if strings.HasPrefix(data, "cancel:") {
		gid := strings.TrimPrefix(data, "cancel:")
		isSudo := auth.IsSudo(senderID)
		success, msg := task.GlobalManager.CancelTask(gid, senderID, isSudo)
		if success {
			_, _ = cb.Answer("Task cancelled successfully!", &tg.CallbackOptions{Alert: false})
			_, _ = cb.Edit("🛑 " + msg)
		} else {
			_, _ = cb.Answer(msg, &tg.CallbackOptions{Alert: true})
		}
		return nil
	}

	if strings.HasPrefix(data, "status:") {
		parts := strings.Split(data, ":")
		if len(parts) >= 2 {
			action := parts[1]
			switch action {
			case "refresh":
				text, markup := GenerateStatusMessage(1, "ALL")
				_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
				_, _ = cb.Answer("Refreshed", &tg.CallbackOptions{})
			case "page":
				if len(parts) >= 3 {
					var p int
					fmt.Sscanf(parts[2], "%d", &p)
					text, markup := GenerateStatusMessage(p, "ALL")
					_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
				}
				_, _ = cb.Answer("", &tg.CallbackOptions{})
			case "filter":
				if len(parts) >= 3 {
					flt := parts[2]
					text, markup := GenerateStatusMessage(1, flt)
					_, _ = cb.Edit(text, &tg.SendOptions{ParseMode: "html", ReplyMarkup: markup})
				}
				_, _ = cb.Answer("", &tg.CallbackOptions{})
			}
		}
		return nil
	}

	if data == "usetting:toggle_doc" {
		if db.GlobalStore != nil {
			st, _ := db.GlobalStore.GetUserSettings(senderID)
			if st == nil {
				st = &db.UserSettings{UserID: senderID}
			}
			st.AsDoc = !st.AsDoc
			_ = db.GlobalStore.SaveUserSettings(st)
			_, _ = cb.Answer("Leech document mode toggled!", &tg.CallbackOptions{})
		}
		return nil
	}

	if data == "usetting:close" {
		_, _ = cb.Delete()
		return nil
	}

	_, _ = cb.Answer("Action handled", &tg.CallbackOptions{})
	return nil
}
