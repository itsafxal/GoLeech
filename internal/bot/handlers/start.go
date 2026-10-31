package handlers

import (
	"fmt"
	"log"

	"github.com/GoLeech/GoLeech/internal/auth"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// HandleStart handles the /start command
func HandleStart(m *tg.NewMessage) error {
	senderID := m.SenderID()
	chatID := m.ChatID()
	role := auth.GetUserRole(senderID, chatID)

	text := fmt.Sprintf(`⌬ <b><u>GoLeech — High-Performance Cloud Transfer</u></b>
│
┠ <b>Tagline:</b> <i>Mirror Anything. Leech Anywhere. Powered by Go.</i>
┠ <b>User ID:</b> <code>%d</code>
┠ <b>Access Role:</b> <b>%s</b>
┠ <b>Core Engine:</b> <code>Gogram MTProto + Pure Go Multi-Engine</code>
│
┖ <i>Choose a category below to explore commands, features, and settings:</i>`, senderID, role)

	markup := tg.Button.Keyboard(
		tg.Button.Row(
			tg.Button.Data("📥 Downloads", "help:dl"),
			tg.Button.Data("📤 Uploads", "help:up"),
		),
		tg.Button.Row(
			tg.Button.Data("⚡️ Flags Reference", "help:flags"),
			tg.Button.Data("⚙️ User Settings", "help:settings"),
		),
		tg.Button.Row(
			tg.Button.Data("🛡 Admin & Sudo", "help:admin"),
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
	if err != nil {
		log.Printf("⚠️ Error replying to /start for %d: %v", senderID, err)
	}
	return err
}

// HandleHelp displays the interactive tabbed help menu
func HandleHelp(m *tg.NewMessage) error {
	text, markup := GetHelpContent("main")
	_, err := m.Reply(text, &tg.SendOptions{
		ParseMode:   "html",
		ReplyMarkup: markup,
	})
	if err != nil {
		log.Printf("⚠️ Error replying to /help for %d: %v", m.SenderID(), err)
	}
	return err
}

// GetHelpContent returns formatted text and interactive buttons for each tab
func GetHelpContent(tab string) (string, tg.ReplyMarkup) {
	var text string

	switch tab {
	case "dl":
		text = `⌬ <b><u>📥 Supported Download Sources</u></b>
│
┠ <b>🌐 Direct HTTP / HTTPS</b>
┃ ┠ Multi-threaded concurrent chunk downloads
┃ ┠ Automatic pause, resume, and redirect handling
┃ ┖ Direct link resolvers for MediaFire, Google Drive, Mega, etc.
│
┠ <b>🧲 BitTorrent & Magnet</b>
┃ ┠ High-speed peer-to-peer downloading
┃ ┠ Multi-tracker scraping, DHT, PEX, and PeX support
┃ ┖ Interactive selective file downloading (<code>/select</code> or <code>-s</code>)
│
┠ <b>✈️ Telegram Native Media</b>
┃ ┠ Reply to any Telegram file or video with <code>/leech</code> or <code>/mirror</code>
┃ ┖ Fast streaming MTProto chunk fetching (2GB/4GB support)
│
┠ <b>📹 YT-DLP Streaming Engine</b>
┃ ┠ Supports 1000+ video sites (YouTube, Twitter/X, Instagram, TikTok, etc.)
┃ ┖ Audio extraction (<code>-ad</code>), format selection, and playlist processing`

	case "up":
		text = `⌬ <b><u>📤 Supported Upload Targets</u></b>
│
┠ <b>✈️ Telegram Cloud (Leech)</b>
┃ ┠ <b>2GB Standard Uploads</b> via Telegram Bot API / MTProto
┃ ┠ <b>4GB Premium Uploads</b> via optional User Session
┃ ┠ Auto-splitting of oversized files with clean segment naming
┃ ┠ Custom thumbnail embedding (<code>-t</code> or <code>/usetting</code>)
┃ ┖ Streamable video conversion with FastStart (moov atom at head)
│
┠ <b>☁️ Cloud Storage (Mirror via Rclone)</b>
┃ ┠ Google Drive, OneDrive, Mega.nz, Dropbox, Nextcloud
┃ ┠ AWS S3, Cloudflare R2, Backblaze B2, Wasabi, MinIO
┃ ┠ WebDAV, FTP, SFTP, and 40+ Rclone-compatible remotes
┃ ┖ Direct cloud-to-cloud clone without local storage (<code>/clone</code>)`

	case "flags":
		text = `⌬ <b><u>⚡️ Command Flags Reference</u></b>
│
┠ <b>Output & Format:</b>
┃ ┠ <code>-n &lt;name&gt;</code> — Custom filename or target directory
┃ ┠ <code>-doc</code> — Send file strictly as an uncompressed document
┃ ┠ <code>-med</code> — Send file as a streamable Telegram video
┃ ┠ <code>-sp &lt;size&gt;</code> — Split size override (e.g. <code>-sp 1.5GB</code>)
│
┠ <b>Archive & Extraction:</b>
┃ ┠ <code>-e</code> — Auto-extract archives (.zip, .rar, .7z, .tar, .gz)
┃ ┖ <code>-z</code> — Compress downloaded files into a zip archive
│
┠ <b>Media Processing & FFmpeg:</b>
┃ ┠ <code>-ss</code> — Capture 9-frame photo grid screenshot thumbnail
┃ ┠ <code>-sv</code> — Generate 30-second sample preview video clip
┃ ┠ <code>-ca &lt;codec&gt;</code> — Convert audio stream (mp3, aac, opus, flac)
┃ ┠ <code>-cv &lt;codec&gt;</code> — Convert video stream (mp4, mkv, webm)
┃ ┠ <code>-meta &lt;title&gt;</code> — Inject stream metadata title tag
┃ ┠ <code>-ns &lt;find:rep&gt;</code> — Name swap regex replacement
┃ ┖ <code>-ff &lt;args&gt;</code> — Pass custom ffmpeg filter flags
│
┠ <b>Task Execution:</b>
┃ ┠ <code>-f</code> — Force start (bypass concurrency queue limits)
┃ ┠ <code>-s</code> — Select files interactively before downloading
┃ ┠ <code>-b</code> — Bulk processing from multi-line text input
┃ ┠ <code>-up &lt;dest&gt;</code> — Cloud destination path (e.g. <code>-up remote:folder</code>)
┃ ┖ <code>-ud &lt;chat_id&gt;</code> — Target leech dump channel ID`

	case "settings":
		text = `⌬ <b><u>⚙️ User Settings & Personalization</u></b>
│
┠ <b>Command:</b> <code>/usetting</code> (or alias <code>/us</code>)
│
┠ <b>Custom Thumbnails:</b>
┃ ┠ Send an image in private chat to set as your default thumbnail
┃ ┠ View your saved thumbnail: Click <b>View Thumb</b>
┃ ┖ Delete thumbnail: Click <b>Delete Thumb</b>
│
┠ <b>Leech Mode & Split Sizes:</b>
┃ ┠ Toggle between <b>Document 📄</b> and <b>Media 🎥</b> mode
┃ ┖ Toggle between <b>2GB Standard</b> and <b>4GB Premium</b> split limit
│
┠ <b>Caption Placeholders:</b>
┃ ┠ <code>{filename}</code> — Original file name
┃ ┠ <code>{size}</code> — Formatted file size (e.g. 1.45 GB)
┃ ┠ <code>{duration}</code> — Media duration (e.g. 01:45:20)
┃ ┖ <code>{speed}</code> — Transfer speed (e.g. 12.50 MB/s)`

	case "admin":
		text = `⌬ <b><u>🛡 Admin & Sudo Command Reference</u></b>
│
┠ <b>Chat & User Authorization:</b>
┃ ┠ <code>/authorize [chat_id]</code> (or <code>/a</code>) — Authorize group/user
┃ ┠ <code>/unauthorize [chat_id]</code> (or <code>/ua</code>) — Revoke chat access
┃ ┠ <code>/addsudo &lt;user_id&gt;</code> (or <code>/as</code>) — Grant Sudo admin
┃ ┖ <code>/rmsudo &lt;user_id&gt;</code> (or <code>/rs</code>) — Revoke Sudo admin
│
┠ <b>Security & Blacklist:</b>
┃ ┠ <code>/blacklist &lt;id&gt;</code> (or <code>/bl</code>) — Ban user or group
┃ ┠ <code>/rmblacklist &lt;id&gt;</code> (or <code>/rbl</code>) — Unban user or group
┃ ┖ <code>/users</code> — Show active user statistics & counts
│
┠ <b>System Maintenance:</b>
┃ ┠ <code>/bsetting</code> (or <code>/bs</code>) — Interactive bot runtime settings
┃ ┠ <code>/forcestart [gid]</code> (or <code>/fs</code>) — Force queued task to run
┃ ┠ <code>/broadcast</code> (or <code>/bc</code>) — Send announcement to all users
┃ ┠ <code>/log</code> — View recent logs or export log file
┃ ┠ <code>/restart</code> (or <code>/r</code>) — Gracefully restart bot instance
┃ ┖ <code>/ping</code> / <code>/stats</code> — Response latency & system memory`

	default: // "main"
		text = `⌬ <b><u>GoLeech — Comprehensive Guide</u></b>
│
┠ <b>Core Task Commands:</b>
┃ ┠ <code>/mirror [link/flag]</code> (or <code>/m</code>) — Mirror to cloud
┃ ┠ <code>/leech [link/flag]</code> (or <code>/l</code>) — Download and upload to Telegram
┃ ┠ <code>/clone &lt;src&gt; &lt;dst&gt;</code> (or <code>/cl</code>) — Cloud-to-cloud clone
┃ ┠ <code>/status</code> (or <code>/s</code>) — Live task progress dashboard
┃ ┠ <code>/cancel &lt;gid&gt;</code> (or <code>/c</code>) — Cancel specific task
┃ ┖ <code>/cancelall [all/dl/up]</code> — Cancel tasks in bulk
│
┠ <b>Interactive Features:</b>
┃ ┠ <code>/select &lt;gid&gt;</code> (or <code>/sel</code>) — Selective file picker
┃ ┠ <code>/search &lt;query&gt;</code> — Search torrents and sources
┃ ┠ <code>/usetting</code> (or <code>/us</code>) — User settings portal
┃ ┖ <code>/bsetting</code> (or <code>/bs</code>) — Bot settings dashboard
│
┖ <i>Tap on the tabs below to view detailed guides:</i>`
	}

	markup := tg.Button.Keyboard(
		tg.Button.Row(
			tg.Button.Data("📥 Downloads", "help:dl"),
			tg.Button.Data("📤 Uploads", "help:up"),
		),
		tg.Button.Row(
			tg.Button.Data("⚡️ Flags", "help:flags"),
			tg.Button.Data("⚙️ Settings", "help:settings"),
		),
		tg.Button.Row(
			tg.Button.Data("🛡 Admin", "help:admin"),
			tg.Button.Data("🔙 Overview", "help:main"),
		),
		tg.Button.Row(
			tg.Button.Data("❌ Close Menu", "help:close"),
		),
	)

	return text, markup
}
