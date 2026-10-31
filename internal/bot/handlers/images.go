package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/bot/ui"
	"github.com/GoLeech/GoLeech/internal/db"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// HandleAddImage handles /addimage and /ai
func HandleAddImage(m *tg.NewMessage, botClient *tg.Client) error {
	senderID := m.SenderID()
	if !auth.CanExecute(senderID, m.ChatID()) {
		_, err := ui.Reply(m, "⛔️ You are not authorized to use this command.")
		return err
	}

	replyMsg, _ := m.GetReplyMessage()
	var picURL string

	args := strings.Fields(m.Text())
	if len(args) > 1 {
		picURL = args[1]
	} else if replyMsg != nil && replyMsg.Text() != "" {
		picURL = strings.TrimSpace(replyMsg.Text())
	} else if replyMsg != nil && replyMsg.Photo() != nil {
		imgDir := filepath.Join("data", "images")
		_ = os.MkdirAll(imgDir, 0755)
		localPath := filepath.Join(imgDir, fmt.Sprintf("img_%d_%d.jpg", senderID, os.Getpid()))
		if _, err := replyMsg.Download(&tg.DownloadOptions{FileName: localPath}); err == nil {
			picURL = localPath
		}
	}

	if picURL == "" || (!strings.HasPrefix(picURL, "http://") && !strings.HasPrefix(picURL, "https://") && !strings.HasSuffix(picURL, ".jpg")) {
		usage := `⚜ <b><u>Add Image Usage</u></b>
│
┠ <b>Reply to Link:</b> <code>/addimage {link}</code>
┠ <b>Reply to Photo:</b> <code>/addimage</code>
┖ <b>Supported:</b> <i>Telegra.ph, DDL direct links, Telegram photos</i>`
		_, err := ui.Reply(m, usage)
		return err
	}

	if db.GlobalStore != nil {
		_ = db.GlobalStore.AddImage(picURL)
		images := db.GlobalStore.GetImages()
		resp := fmt.Sprintf(`⚜ <b><u>Image Added</u></b>
│
┖ <b>Total Images:</b> <code>%d</code>`, len(images))
		_, err := ui.Reply(m, resp)
		return err
	}

	_, err := ui.Reply(m, "Database not initialized.")
	return err
}

// HandleImages handles /images and /img gallery view
func HandleImages(m *tg.NewMessage, botClient *tg.Client) error {
	senderID := m.SenderID()
	if !auth.CanExecute(senderID, m.ChatID()) {
		_, err := ui.Reply(m, "⛔️ You are not authorized to use this command.")
		return err
	}

	if db.GlobalStore == nil {
		_, err := ui.Reply(m, "Database not initialized.")
		return err
	}

	images := db.GlobalStore.GetImages()
	if len(images) == 0 {
		_, err := ui.Reply(m, "<b>No Photo to Show!</b> Add one using <code>/addimage</code>.")
		return err
	}

	total := len(images)
	caption := fmt.Sprintf("⚜ <b><u>Image Gallery</u></b>\n│\n┖ 🌄 <b>No. : 1 / %d</b>", total)
	markup := ui.Button.Keyboard(
		ui.Button.Row(
			ui.Button.Data("«", fmt.Sprintf("images %d turn %d", senderID, total-1)),
			ui.Button.Data("»", fmt.Sprintf("images %d turn %d", senderID, 1%total)),
		),
		ui.Button.Row(
			ui.Button.Data("🗑 Remove Image", fmt.Sprintf("images %d remov 0", senderID)),
			ui.Button.Data("❌ Close", fmt.Sprintf("images %d close", senderID)),
		),
		ui.Button.Row(
			ui.Button.Data("💥 Remove All", fmt.Sprintf("images %d removall", senderID)),
		),
	)

	_, err := ui.Reply(m, caption, &tg.SendOptions{
		ReplyMarkup: markup,
	})
	return err
}

// HandleImageCallback processes inline interactions in the image gallery
func HandleImageCallback(cb *tg.CallbackQuery, botClient *tg.Client) error {
	data := strings.Fields(cb.DataString())
	if len(data) < 3 {
		return nil
	}

	var targetUID int64
	fmt.Sscanf(data[1], "%d", &targetUID)
	if targetUID != 0 && targetUID != cb.SenderID && !auth.IsSudo(cb.SenderID) {
		_, _ = ui.Answer(cb, "Not Authorized User!", &tg.CallbackOptions{Alert: true})
		return nil
	}

	action := data[2]
	if action == "close" {
		_, _ = cb.Answer("", &tg.CallbackOptions{})
		_, _ = cb.Delete()
		return nil
	}

	if db.GlobalStore == nil {
		return nil
	}

	images := db.GlobalStore.GetImages()

	if action == "removall" {
		_ = db.GlobalStore.ClearImages()
		_, _ = ui.Answer(cb, "All Images Successfully Deleted", &tg.CallbackOptions{Alert: true})
		_, _ = cb.Delete()
		return nil
	}

	if action == "remov" && len(data) > 3 {
		idx, _ := strconv.Atoi(data[3])
		_ = db.GlobalStore.RemoveImage(idx)
		_, _ = ui.Answer(cb, "Image Successfully Deleted", &tg.CallbackOptions{Alert: true})
		images = db.GlobalStore.GetImages()
		if len(images) == 0 {
			_, _ = cb.Delete()
			return nil
		}
	}

	if (action == "turn" || action == "remov") && len(images) > 0 {
		var idx int
		if len(data) > 3 {
			idx, _ = strconv.Atoi(data[3])
		}
		if idx < 0 {
			idx = len(images) - 1
		} else if idx >= len(images) {
			idx = 0
		}

		total := len(images)
		prev := (idx - 1 + total) % total
		next := (idx + 1) % total

		caption := fmt.Sprintf("⚜ <b><u>Image Gallery</u></b>\n│\n┖ 🌄 <b>No. : %d / %d</b>\n\n<code><raw>%s</raw></code>", idx+1, total, images[idx])
		markup := ui.Button.Keyboard(
			ui.Button.Row(
				ui.Button.Data("«", fmt.Sprintf("images %d turn %d", cb.SenderID, prev)),
				ui.Button.Data("»", fmt.Sprintf("images %d turn %d", cb.SenderID, next)),
			),
			ui.Button.Row(
				ui.Button.Data("🗑 Remove Image", fmt.Sprintf("images %d remov %d", cb.SenderID, idx)),
				ui.Button.Data("❌ Close", fmt.Sprintf("images %d close", cb.SenderID)),
			),
			ui.Button.Row(
				ui.Button.Data("💥 Remove All", fmt.Sprintf("images %d removall", cb.SenderID)),
			),
		)

		_, _ = ui.Edit(cb, caption, &tg.SendOptions{
			ReplyMarkup: markup,
		})
		_, _ = cb.Answer("", &tg.CallbackOptions{})
	}

	return nil
}
