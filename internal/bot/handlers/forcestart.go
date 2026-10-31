package handlers

import (
	"strings"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/bot/ui"
	"github.com/GoLeech/GoLeech/internal/task"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// HandleForceStart promotes a queued task to run immediately
func HandleForceStart(m *tg.NewMessage) error {
	senderID := m.SenderID()
	chatID := m.ChatID()

	if !auth.CanExecute(senderID, chatID) {
		_, err := ui.Reply(m, "⛔️ You are not authorized to execute tasks.")
		return err
	}

	args := strings.Fields(m.Args())
	var targetGID string

	if len(args) > 0 {
		targetGID = args[0]
	} else if m.IsReply() {
		// Try to extract GID from replied message
		replyMsg, err := m.GetReplyMessage()
		if err == nil && replyMsg != nil {
			text := replyMsg.Text()
			// Status messages contain "/c_<gid>" or GID patterns
			if idx := strings.Index(text, "/c_"); idx != -1 {
				rem := text[idx+3:]
				fields := strings.Fields(rem)
				if len(fields) > 0 {
					targetGID = strings.Trim(fields[0], "</i></b> \n\r\t")
				}
			}
		}
	}

	if targetGID == "" {
		usage := `⚠️ <b>ForceStart Usage:</b>
┃ ┠ <code>/forcestart &lt;gid&gt;</code> (or <code>/fs &lt;gid&gt;</code>)
┃ ┖ Or reply to a task status message with: <code>/forcestart</code>`
		_, err := ui.Reply(m, usage)
		return err
	}

	isSudo := auth.IsSudo(senderID)
	success, msg := task.GlobalManager.ForceStartTask(targetGID, senderID, isSudo)

	if success {
		_, err := ui.Reply(m, "⚡️ "+msg)
		return err
	}

	_, err := ui.Reply(m, "❌ "+msg)
	return err
}
