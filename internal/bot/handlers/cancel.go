package handlers

import (
	"fmt"
	"strings"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/bot/ui"
	"github.com/GoLeech/GoLeech/internal/task"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func HandleCancel(m *tg.NewMessage) error {
	senderID := m.SenderID()
	args := strings.Fields(m.Args())

	var targetGID string
	if len(args) > 0 {
		targetGID = args[0]
	} else {
		// Check command name like /c_1234abcd
		cmd := m.GetCommand()
		if strings.HasPrefix(cmd, "c_") {
			targetGID = strings.TrimPrefix(cmd, "c_")
		}
	}

	if targetGID == "" {
		_, err := ui.Reply(m, "⚠️ <b>Usage:</b> <code>/cancel &lt;GID&gt;</code> or <code>/c_&lt;GID&gt;</code>")
		return err
	}

	isSudo := auth.IsSudo(senderID)
	success, msg := task.GlobalManager.CancelTask(targetGID, senderID, isSudo)
	if !success {
		_, err := ui.Reply(m, "❌ "+msg)
		return err
	}

	_, err := ui.Reply(m, "✅ "+msg)
	return err
}

func HandleCancelAll(m *tg.NewMessage) error {
	senderID := m.SenderID()
	filter := strings.ToLower(strings.TrimSpace(m.Args()))
	if filter == "" {
		filter = "all"
	}

	isSudo := auth.IsSudo(senderID)
	cancelledCount := task.GlobalManager.CancelAll(filter, senderID, isSudo)

	_, err := ui.Reply(m, fmt.Sprintf("🛑 Cancelled <b>%d</b> tasks (filter: <i>%s</i>).", cancelledCount, filter))
	return err
}
