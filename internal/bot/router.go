package bot

import (
	"strings"

	"github.com/GoLeech/GoLeech/internal/bot/handlers"
	"github.com/GoLeech/GoLeech/internal/config"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// RegisterRoutes registers all bot commands and callbacks
func RegisterRoutes(botClient, userClient *tg.Client) {
	cfg := config.Get()
	suffix := cfg.CmdSuffix

	cmd := func(name string) string {
		return "cmd:" + name + suffix
	}

	// 1. Core Task Commands
	botClient.On(cmd("start"), handlers.HandleStart)
	botClient.On(cmd("help"), handlers.HandleHelp)
	botClient.On(cmd("h"), handlers.HandleHelp)

	botClient.On(cmd("mirror"), func(m *tg.NewMessage) error {
		return handlers.HandleMirror(m, botClient, userClient)
	})
	botClient.On(cmd("m"), func(m *tg.NewMessage) error {
		return handlers.HandleMirror(m, botClient, userClient)
	})

	botClient.On(cmd("leech"), func(m *tg.NewMessage) error {
		return handlers.HandleLeech(m, botClient, userClient)
	})
	botClient.On(cmd("l"), func(m *tg.NewMessage) error {
		return handlers.HandleLeech(m, botClient, userClient)
	})

	// 2. Status & Monitoring
	botClient.On(cmd("status"), handlers.HandleStatus)
	botClient.On(cmd("s"), handlers.HandleStatus)
	botClient.On(cmd("statusall"), handlers.HandleStatus)

	// 3. Task Cancellation
	botClient.On(cmd("cancel"), handlers.HandleCancel)
	botClient.On(cmd("c"), handlers.HandleCancel)
	botClient.On(cmd("cancelall"), handlers.HandleCancelAll)

	// Support dynamic prefix cancellation like /c_1234abcd
	botClient.On("message", func(m *tg.NewMessage) error {
		if m.IsCommand() {
			c := m.GetCommand()
			if strings.HasPrefix(c, "c_") {
				return handlers.HandleCancel(m)
			}
		}
		return nil
	})

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
		return handlers.HandleCallback(cb, botClient)
	})

	// 7. Start periodic status updater
	handlers.StartStatusUpdater(botClient)
}
