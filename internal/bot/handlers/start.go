package handlers

import (
	"fmt"

	"github.com/GoLeech/GoLeech/internal/auth"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func HandleStart(m *tg.NewMessage) error {
	senderID := m.SenderID()
	chatID := m.ChatID()
	role := auth.GetUserRole(senderID, chatID)

	text := fmt.Sprintf(`⌬ <b><u>Welcome to GoLeech!</u></b>
│
┠ <b>Tagline:</b> <i>Mirror Anything. Leech Anywhere. Powered by Go.</i>
┠ <b>Your ID:</b> <code>%d</code>
┠ <b>Your Role:</b> <code>%s</code>
│
┖ Send /help to discover available commands and options.`, senderID, role)

	markup := tg.Button.Keyboard(
		tg.Button.Row(
			tg.Button.Data("📖 Help & Docs", "help:main"),
			tg.Button.Data("📊 System Stats", "stats:view"),
		),
		tg.Button.Row(
			tg.Button.URL("⭐️ GitHub Repository", "https://github.com/GoLeech/GoLeech"),
		),
	)

	_, err := m.Reply(text, &tg.SendOptions{
		ParseMode:   "html",
		ReplyMarkup: markup,
	})
	return err
}

func HandleHelp(m *tg.NewMessage) error {
	text := `⌬ <b><u>GoLeech Command Guide</u></b>
│
┠ <b>Task Commands:</b>
┃ ┠ <code>/mirror [link/flag]</code> - Mirror link to cloud storage
┃ ┠ <code>/leech [link/flag]</code> - Download and upload to Telegram
┃ ┠ <code>/status</code> (or <code>/s</code>) - Live task status & progress
┃ ┠ <code>/cancel [gid]</code> - Cancel a specific task
┃ ┖ <code>/cancelall [all/dl/up]</code> - Cancel tasks in bulk
│
┠ <b>Supported Flags:</b>
┃ ┠ <code>-n &lt;name&gt;</code> - Custom filename
┃ ┠ <code>-doc</code> - Leech media as document
┃ ┠ <code>-med</code> - Leech media as streamable video
┃ ┠ <code>-e</code> - Extract archive (.zip, .rar, .7z)
┃ ┠ <code>-z</code> - Compress files into zip before upload
┃ ┠ <code>-ss</code> - Generate screenshot thumbnails
┃ ┠ <code>-sv</code> - Generate 30s sample preview clip
┃ ┠ <code>-f</code> - Force start (bypass queue limit)
┃ ┠ <code>-sp &lt;size&gt;</code> - Split size override (e.g. 2GB)
┃ ┖ <code>-up &lt;dest&gt;</code> - Cloud destination path (Rclone/GDrive)
│
┠ <b>Settings & Admin:</b>
┃ ┠ <code>/usetting</code> - User settings (prefix, caption, thumb)
┃ ┠ <code>/bsetting</code> - Bot runtime settings (Sudo only)
┃ ┠ <code>/authorize</code> / <code>/unauthorize</code> - Manage chat access
┃ ┖ <code>/ping</code> / <code>/stats</code> - Speed and resource usage`

	_, err := m.Reply(text, &tg.SendOptions{ParseMode: "html"})
	return err
}
