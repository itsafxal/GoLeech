package bot

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GoLeech/GoLeech/internal/bot/handlers"
	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
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

		// Quick-cancellation handler for /c_<gid>
		if m.IsCommand() {
			c := m.GetCommand()
			if strings.HasPrefix(c, "c_") {
				return handlers.HandleCancel(m)
			}
		}

		// Private Chat: Auto-save user photo as custom leech thumbnail
		if m.ChatID() == m.SenderID() && m.Photo() != nil && !m.IsCommand() {
			thumbDir := filepath.Join("data", "thumbnails")
			_ = os.MkdirAll(thumbDir, 0755)
			targetFile := filepath.Join(thumbDir, fmt.Sprintf("%d.jpg", m.SenderID()))

			_, err := m.Download(&tg.DownloadOptions{FileName: targetFile})
			if err == nil && db.GlobalStore != nil {
				st, _ := db.GlobalStore.GetUserSettings(m.SenderID())
				if st == nil {
					st = &db.UserSettings{UserID: m.SenderID()}
				}
				st.ThumbPath = targetFile
				_ = db.GlobalStore.SaveUserSettings(st)

				successText := `🖼 <b><u>Custom Thumbnail Saved!</u></b>
│
┠ <b>Target:</b> Leech video & document thumbnail
┠ <b>Status:</b> ✅ Active
│
┖ <i>This image will be embedded into all your future leeched media files!</i>`
				_, _ = m.Reply(successText, &tg.SendOptions{
					ParseMode: "html",
					ReplyMarkup: tg.Button.Keyboard(
						tg.Button.Row(
							tg.Button.Data("⚙️ View Settings", fmt.Sprintf("usetting:refresh:%d", m.SenderID())),
							tg.Button.Data("🗑 Delete Thumb", fmt.Sprintf("usetting:del_thumb:%d", m.SenderID())),
						),
					),
				})
			}
		}

		return nil
	})

	// 1. Navigation & Help
	botClient.On(cmd("start"), handlers.HandleStart)
	botClient.On(cmd("help"), handlers.HandleHelp)
	botClient.On(cmd("h"), handlers.HandleHelp)

	// 2. Core Transfer Tasks
	botClient.On(cmd("mirror"), func(m *tg.NewMessage) error {
		return handlers.HandleMirror(m, botClient, GetUserClient())
	})
	botClient.On(cmd("m"), func(m *tg.NewMessage) error {
		return handlers.HandleMirror(m, botClient, GetUserClient())
	})

	botClient.On(cmd("leech"), func(m *tg.NewMessage) error {
		return handlers.HandleLeech(m, botClient, GetUserClient())
	})
	botClient.On(cmd("l"), func(m *tg.NewMessage) error {
		return handlers.HandleLeech(m, botClient, GetUserClient())
	})

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

	// 8. Interactive Callback Queries
	botClient.On("callback", func(cb *tg.CallbackQuery) error {
		log.Printf("🔘 Received callback query from %d: %q", cb.SenderID, string(cb.Data))
		return handlers.HandleCallback(cb, botClient)
	})

	// 9. Periodic Status Updater Service
	handlers.StartStatusUpdater(botClient)
}
