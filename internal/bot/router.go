package bot

import (
	"log"
	"strings"
	"sync"

	"github.com/GoLeech/GoLeech/internal/bot/handlers"
	"github.com/GoLeech/GoLeech/internal/config"
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

// RegisterRoutes registers all bot commands and callbacks
func RegisterRoutes(botClient *tg.Client) {
	cfg := config.Get()
	suffix := cfg.CmdSuffix

	cmd := func(name string) string {
		return "cmd:" + name + suffix
	}

	// 0. Global incoming message logger for debugging and transparency
	botClient.On("message", func(m *tg.NewMessage) error {
		if m != nil && m.Text() != "" {
			log.Printf("📩 Received message from %d (chat: %d): %q", m.SenderID(), m.ChatID(), m.Text())
		}
		if m != nil && m.IsCommand() {
			c := m.GetCommand()
			if strings.HasPrefix(c, "c_") {
				return handlers.HandleCancel(m)
			}
		}
		return nil
	})

	// 1. Core Task Commands
	botClient.On(cmd("start"), handlers.HandleStart)
	botClient.On(cmd("help"), handlers.HandleHelp)
	botClient.On(cmd("h"), handlers.HandleHelp)

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

	// 2. Status & Monitoring
	botClient.On(cmd("status"), handlers.HandleStatus)
	botClient.On(cmd("s"), handlers.HandleStatus)
	botClient.On(cmd("statusall"), handlers.HandleStatus)

	// 3. Task Cancellation
	botClient.On(cmd("cancel"), handlers.HandleCancel)
	botClient.On(cmd("c"), handlers.HandleCancel)
	botClient.On(cmd("cancelall"), handlers.HandleCancelAll)

	// 4. Admin & Auth
	botClient.On(cmd("ping"), handlers.HandlePing)
	botClient.On(cmd("stats"), handlers.HandleStats)
	botClient.On(cmd("authorize"), handlers.HandleAuthorize)
	botClient.On(cmd("a"), handlers.HandleAuthorize)
	botClient.On(cmd("unauthorize"), handlers.HandleUnauthorize)
	botClient.On(cmd("ua"), handlers.HandleUnauthorize)
	botClient.On(cmd("addsudo"), handlers.HandleAddSudo)
	botClient.On(cmd("as"), handlers.HandleAddSudo)
	botClient.On(cmd("rmsudo"), handlers.HandleRemoveSudo)
	botClient.On(cmd("rs"), handlers.HandleRemoveSudo)

	// 5. Settings
	botClient.On(cmd("usetting"), handlers.HandleUserSettings)
	botClient.On(cmd("us"), handlers.HandleUserSettings)
	botClient.On(cmd("bsetting"), handlers.HandleBotSettings)
	botClient.On(cmd("bs"), handlers.HandleBotSettings)

	// 6. Callback Queries
	botClient.On("callback", func(cb *tg.CallbackQuery) error {
		log.Printf("🔘 Received callback query from %d: %q", cb.SenderID, string(cb.Data))
		return handlers.HandleCallback(cb, botClient)
	})

	// 7. Start periodic status updater
	handlers.StartStatusUpdater(botClient)
}
