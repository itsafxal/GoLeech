package bot

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/bot/handlers"
	"github.com/GoLeech/GoLeech/internal/bot/ui"
	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/internal/downloader"
	"github.com/GoLeech/GoLeech/pkg/typography"
	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	activeUserClient *tg.Client
	userClientMu     sync.RWMutex
)

// SetUserClient safely sets or updates the optional user client (for 4GB uploads)
func SetUserClient(client *tg.Client) {
	userClientMu.Lock()
	activeUserClient = client
	userClientMu.Unlock()
}

// GetUserClient safely returns the active user client
func GetUserClient() *tg.Client {
	userClientMu.RLock()
	defer userClientMu.RUnlock()
	return activeUserClient
}

// RegisterRoutes registers all bot commands, media listeners, and callbacks
func RegisterRoutes(botClient *tg.Client) {
	cfg := config.Get()
	suffix := cfg.CmdSuffix

	cmd := func(name string) string {
		return "cmd:" + name + suffix
	}

	// 0. Global incoming message logger & interactive private media handler
	botClient.On("message", func(m *tg.NewMessage) error {
		if m == nil {
			return nil
		}

		if m.Text() != "" {
			log.Printf("📩 Received message from %d (chat: %d): %q", m.SenderID(), m.ChatID(), m.Text())
		}

		// Strict Authorization Enforcement for all commands (fail-closed)
		if m.IsCommand() {
			c := m.GetCommand()
			cleanCmd := strings.TrimSuffix(c, suffix)
			if !auth.CanAccessBot(m.SenderID()) {
				if cleanCmd == "start" {
					return handlers.HandleStart(m)
				}
				unauthNotice := `⛔️ <b><u>Access Denied</u></b>
│
┖ You are not authorized to use this bot. Only authorized Admin and Sudo users can access its features. You can create your own repository and configure an independent instance.`
				repoMarkup := tg.Button.Keyboard(
					tg.Button.Row(
						tg.Button.URL("⭐️ GitHub Repository", "https://github.com/itsafxal/GoLeech"),
					),
				)
				_, _ = ui.Reply(m, unauthNotice, &tg.SendOptions{ReplyMarkup: repoMarkup})
				return nil
			}

			// Quick-cancellation handler for /c_<gid>
			if strings.HasPrefix(c, "c_") {
				return handlers.HandleCancel(m)
			}
		}

		// Block unauthorized non-command interaction (settings prompt or thumbnail upload)
		if !auth.CanAccessBot(m.SenderID()) {
			return nil
		}

		// Check for active interactive user settings prompt
		if m.SenderID() != 0 && (m.ChatID() == m.SenderID() || m.IsReply()) {
			if pending := handlers.GetPendingInput(m.SenderID()); pending != nil {
				return handlers.ProcessPendingUserInput(m, pending, botClient)
			}
		}

		// Private Chat: Auto-save user photo as custom leech thumbnail with permanent MongoDB persistence
		if m.ChatID() == m.SenderID() && m.Photo() != nil && !m.IsCommand() {
			thumbDir := filepath.Join("data", "thumbnails")
			_ = os.MkdirAll(thumbDir, 0755)
			targetFile := filepath.Join(thumbDir, fmt.Sprintf("%d.jpg", m.SenderID()))

			_, err := m.Download(&tg.DownloadOptions{FileName: targetFile})
			if err == nil && db.GlobalStore != nil {
				thumbBytes, _ := os.ReadFile(targetFile)
				st, _ := db.GlobalStore.GetUserSettings(m.SenderID())
				if st == nil {
					st = &db.UserSettings{UserID: m.SenderID()}
				}
				st.ThumbPath = targetFile
				st.ThumbData = thumbBytes
				if m.Photo() != nil {
					st.ThumbFileID = fmt.Sprintf("%d", m.Photo().ID)
				}
				_ = db.GlobalStore.SaveUserSettings(st)

				successText := `🖼 <b><u>Custom Thumbnail Saved & Persisted!</u></b>
│
┠ <b>Target:</b> Leech video & document thumbnail
┠ <b>Status:</b> ✅ Active & Synced to Database
│
┖ <i>This thumbnail is permanently stored and will be embedded into all your future leeched media!</i>`
				_, _ = ui.Reply(m, successText, &tg.SendOptions{
					ReplyMarkup: ui.Button.Keyboard(
						ui.Button.Row(
							ui.Button.Data("⚙️ View Settings", fmt.Sprintf("userset %d thumb", m.SenderID())),
							ui.Button.Data("🗑 Delete Thumb", fmt.Sprintf("userset %d remove THUMBNAIL", m.SenderID())),
						),
					),
				})
			}
		}

		// Private Chat: Auto-Leech for direct HTTP/HTTPS URLs, Torrents, or Telegram Media
		if m.ChatID() == m.SenderID() && !m.IsCommand() {
			src := downloader.ClassifySource(m, nil, "")
			if src.Type != downloader.SourceTypeNone {
				return handlers.HandleLeech(m, botClient, GetUserClient())
			}
		}

		return nil
	})

	// 1. Navigation & Help
	botClient.On(cmd("start"), handlers.HandleStart)
	botClient.On(cmd("help"), handlers.HandleHelp)
	botClient.On(cmd("h"), handlers.HandleHelp)

	// 2. Core Transfer Tasks
	mirrorHandler := func(m *tg.NewMessage) error {
		return handlers.HandleMirror(m, botClient, GetUserClient())
	}
	leechHandler := func(m *tg.NewMessage) error {
		return handlers.HandleLeech(m, botClient, GetUserClient())
	}

	botClient.On(cmd("mirror"), mirrorHandler)
	botClient.On(cmd("m"), mirrorHandler)
	botClient.On(cmd("qbmirror"), mirrorHandler)
	botClient.On(cmd("qm"), mirrorHandler)
	botClient.On(cmd("jdmirror"), mirrorHandler)
	botClient.On(cmd("jm"), mirrorHandler)
	botClient.On(cmd("ytdl"), mirrorHandler)
	botClient.On(cmd("y"), mirrorHandler)
	botClient.On(cmd("uphoster"), mirrorHandler)
	botClient.On(cmd("up"), mirrorHandler)
	botClient.On(cmd("nzbmirror"), mirrorHandler)
	botClient.On(cmd("nm"), mirrorHandler)
	botClient.On(cmd("seedrlink"), mirrorHandler)
	botClient.On(cmd("slink"), mirrorHandler)
	botClient.On(cmd("srlink"), mirrorHandler)

	botClient.On(cmd("leech"), leechHandler)
	botClient.On(cmd("l"), leechHandler)
	botClient.On(cmd("qbleech"), leechHandler)
	botClient.On(cmd("ql"), leechHandler)
	botClient.On(cmd("jdleech"), leechHandler)
	botClient.On(cmd("jl"), leechHandler)
	botClient.On(cmd("ytdlleech"), leechHandler)
	botClient.On(cmd("yl"), leechHandler)
	botClient.On(cmd("nzbleech"), leechHandler)
	botClient.On(cmd("nl"), leechHandler)

	botClient.On(cmd("clone"), func(m *tg.NewMessage) error {
		return handlers.HandleClone(m, botClient)
	})
	botClient.On(cmd("cl"), func(m *tg.NewMessage) error {
		return handlers.HandleClone(m, botClient)
	})

	// 3. Queue & Task Management
	botClient.On(cmd("forcestart"), handlers.HandleForceStart)
	botClient.On(cmd("fs"), handlers.HandleForceStart)
	botClient.On(cmd("select"), handlers.HandleSelect)
	botClient.On(cmd("sel"), handlers.HandleSelect)

	// 4. Status & Cancellation
	botClient.On(cmd("status"), handlers.HandleStatus)
	botClient.On(cmd("s"), handlers.HandleStatus)
	botClient.On(cmd("statusall"), handlers.HandleStatus)
	botClient.On(cmd("cancel"), handlers.HandleCancel)
	botClient.On(cmd("c"), handlers.HandleCancel)
	botClient.On(cmd("cancelall"), handlers.HandleCancelAll)

	// 5. Search Engine
	botClient.On(cmd("search"), handlers.HandleSearch)

	// 6. Settings Panels & Customization
	botClient.On(cmd("usetting"), handlers.HandleUserSettings)
	botClient.On(cmd("us"), handlers.HandleUserSettings)
	botClient.On(cmd("bsetting"), handlers.HandleBotSettings)
	botClient.On(cmd("bs"), handlers.HandleBotSettings)
	botClient.On(cmd("addimage"), func(m *tg.NewMessage) error {
		return handlers.HandleAddImage(m, botClient)
	})
	botClient.On(cmd("ai"), func(m *tg.NewMessage) error {
		return handlers.HandleAddImage(m, botClient)
	})
	botClient.On(cmd("images"), func(m *tg.NewMessage) error {
		return handlers.HandleImages(m, botClient)
	})
	botClient.On(cmd("img"), func(m *tg.NewMessage) error {
		return handlers.HandleImages(m, botClient)
	})
	botClient.On(cmd("memory"), handlers.HandleMemory)
	botClient.On(cmd("mem"), handlers.HandleMemory)
	botClient.On(cmd("plugins"), handlers.HandlePlugins)
	botClient.On(cmd("setthumb"), handlers.HandleSetThumb)
	botClient.On(cmd("thumb"), handlers.HandleSetThumb)
	botClient.On(cmd("delthumb"), handlers.HandleDelThumb)
	botClient.On(cmd("rmthumb"), handlers.HandleDelThumb)
	botClient.On(cmd("setprefix"), handlers.HandleSetPrefix)
	botClient.On(cmd("setsuffix"), handlers.HandleSetSuffix)
	botClient.On(cmd("setcaption"), handlers.HandleSetCaption)
	botClient.On(cmd("setdump"), handlers.HandleSetDump)

	// 7. Administration, Permissions & Maintenance
	botClient.On(cmd("ping"), handlers.HandlePing)
	botClient.On(cmd("stats"), handlers.HandleStats)
	botClient.On(cmd("users"), handlers.HandleUsers)
	botClient.On(cmd("authorize"), handlers.HandleAuthorize)
	botClient.On(cmd("a"), handlers.HandleAuthorize)
	botClient.On(cmd("unauthorize"), handlers.HandleUnauthorize)
	botClient.On(cmd("ua"), handlers.HandleUnauthorize)
	botClient.On(cmd("addsudo"), handlers.HandleAddSudo)
	botClient.On(cmd("as"), handlers.HandleAddSudo)
	botClient.On(cmd("rmsudo"), handlers.HandleRemoveSudo)
	botClient.On(cmd("rs"), handlers.HandleRemoveSudo)
	botClient.On(cmd("blacklist"), handlers.HandleBlacklist)
	botClient.On(cmd("bl"), handlers.HandleBlacklist)
	botClient.On(cmd("rmblacklist"), handlers.HandleRemoveBlacklist)
	botClient.On(cmd("rbl"), handlers.HandleRemoveBlacklist)

	botClient.On(cmd("broadcast"), func(m *tg.NewMessage) error {
		return handlers.HandleBroadcast(m, botClient)
	})
	botClient.On(cmd("bc"), func(m *tg.NewMessage) error {
		return handlers.HandleBroadcast(m, botClient)
	})

	botClient.On(cmd("log"), func(m *tg.NewMessage) error {
		return handlers.HandleLog(m, botClient)
	})
	botClient.On(cmd("restart"), handlers.HandleRestart)
	botClient.On(cmd("r"), handlers.HandleRestart)

	// 8. Interactive Callback Queries (Strict Authorization Guard)
	botClient.On("callback", func(cb *tg.CallbackQuery) error {
		log.Printf("🔘 Received callback query from %d: %q", cb.SenderID, string(cb.Data))
		if !auth.CanAccessBot(cb.SenderID) {
			_, _ = cb.Answer(typography.ToSmallCaps("Access Denied: Only Admin and Sudo users can interact with this bot."), &tg.CallbackOptions{Alert: true})
			return nil
		}
		return handlers.HandleCallback(cb, botClient)
	})

	// 9. Periodic Status Updater Service
	handlers.StartStatusUpdater(botClient)
}
