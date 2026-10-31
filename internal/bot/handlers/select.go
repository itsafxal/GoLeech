package handlers

import (
	"fmt"
	"strings"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/task"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// HandleSelect enables interactive file selection on active torrent tasks
func HandleSelect(m *tg.NewMessage) error {
	senderID := m.SenderID()
	chatID := m.ChatID()

	if !auth.CanExecute(senderID, chatID) {
		_, err := m.Reply("⛔️ You are not authorized to execute tasks.", &tg.SendOptions{})
		return err
	}

	args := strings.Fields(m.Args())
	var targetGID string

	if len(args) > 0 {
		targetGID = args[0]
	} else if m.IsReply() {
		replyMsg, err := m.GetReplyMessage()
		if err == nil && replyMsg != nil {
			text := replyMsg.Text()
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
		usage := `⚠️ <b>File Select Usage:</b>
┃ ┠ <code>/select &lt;gid&gt;</code> (or <code>/sel &lt;gid&gt;</code>)
┃ ┖ Or reply to an active torrent task with: <code>/select</code>
┃
<i>💡 Tip: You can also pass <code>-s</code> directly when starting a task: <code>/leech &lt;magnet&gt; -s</code></i>`
		_, err := m.Reply(usage, &tg.SendOptions{ParseMode: "html"})
		return err
	}

	t := task.GlobalManager.GetTask(targetGID)
	if t == nil {
		_, err := m.Reply(fmt.Sprintf("❌ Task <code>%s</code> not found or already completed.", targetGID), &tg.SendOptions{ParseMode: "html"})
		return err
	}

	if !auth.IsSudo(senderID) && t.UserID != senderID {
		_, err := m.Reply("⛔️ You do not have permission to manage this task.", &tg.SendOptions{})
		return err
	}

	if t.Options != nil {
		t.Options.SelectFiles = true
	}

	msg := fmt.Sprintf(`📁 <b><u>Selective Downloading Activated</u></b>
│
┠ <b>Task GID:</b> <code>%s</code>
┠ <b>Target:</b> <b>%s</b>
│
┖ <i>File selection filter is now active for this task. Unwanted files will be bypassed!</i>`, t.GID, t.Name)

	_, err := m.Reply(msg, &tg.SendOptions{ParseMode: "html"})
	return err
}

