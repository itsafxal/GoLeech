package bot

import (
	"log"

	"github.com/GoLeech/GoLeech/pkg/typography"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// RegisterBotCommands registers all supported commands with Telegram MTProto so they appear in native command menu
func RegisterBotCommands(client *tg.Client) error {
	if client == nil {
		return nil
	}

	cmdList := []struct {
		Command string
		Desc    string
	}{
		{"start", "Start bot & get help menu"},
		{"help", "Guide & usage instructions"},
		{"leech", "Leech links/files to Telegram"},
		{"mirror", "Mirror links/files to Cloud"},
		{"clone", "Clone files between cloud storage"},
		{"status", "View active tasks and queue"},
		{"cancel", "Cancel an active or queued task"},
		{"cancelall", "Cancel all your active tasks"},
		{"forcestart", "Force-start queued task immediately"},
		{"select", "Select torrent files to download"},
		{"search", "Search files on torrent providers"},
		{"usetting", "Personal leech and mirror settings"},
		{"bsetting", "Bot administrator global settings"},
		{"images", "Manage custom poster gallery"},
		{"addimage", "Add photo to poster gallery"},
		{"stats", "Bot & server resource statistics"},
		{"ping", "Test bot response latency"},
		{"restart", "Restart bot service"},
		{"log", "Fetch bot runtime logs"},
		{"users", "List authorized bot users"},
		{"broadcast", "Broadcast message to bot users"},
	}

	var tgCommands []*tg.BotCommand
	for _, c := range cmdList {
		tgCommands = append(tgCommands, &tg.BotCommand{
			Command:     c.Command,
			Description: typography.ToSmallCaps(c.Desc),
		})
	}

	_, err := client.BotsSetBotCommands(&tg.BotCommandScopeDefault{}, "", tgCommands)
	if err != nil {
		log.Printf("⚠️ Failed to register bot commands with Telegram: %v", err)
		return err
	}

	log.Printf("🤖 Registered %d bot commands with Telegram native command menu", len(tgCommands))
	return nil
}
