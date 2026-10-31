package handlers

import (
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/internal/task"
	tg "github.com/amarnathcjd/gogram/telegram"
)

func HandlePing(m *tg.NewMessage) error {
	start := time.Now()
	reply, err := m.Reply("🏓 Pong!", &tg.SendOptions{})
	if err != nil {
		log.Printf("⚠️ Error replying to /ping: %v", err)
		return err
	}
	latency := time.Since(start).Milliseconds()
	_, err = reply.Edit(fmt.Sprintf("🏓 <b>Pong!</b> Latency: <code>%d ms</code>", latency), &tg.SendOptions{ParseMode: "html"})
	if err != nil {
		log.Printf("⚠️ Error editing /ping reply: %v", err)
	}
	return err
}

func HandleStats(m *tg.NewMessage) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	tasks := task.GlobalManager.GetAllTasks()
	activeCount := len(tasks)

	text := fmt.Sprintf(`⌬ <b><u>GoLeech System Statistics</u></b>
│
┠ <b>Memory Allocated:</b> <code>%d MB</code>
┠ <b>Total System Memory:</b> <code>%d MB</code>
┠ <b>Active Goroutines:</b> <code>%d</code>
┠ <b>CPU Cores:</b> <code>%d</code>
┠ <b>Active Tasks:</b> <code>%d</code>
┖ <b>Go Version:</b> <code>%s</code>`,
		mem.Alloc/1024/1024,
		mem.Sys/1024/1024,
		runtime.NumGoroutine(),
		runtime.NumCPU(),
		activeCount,
		runtime.Version(),
	)

	_, err := m.Reply(text, &tg.SendOptions{ParseMode: "html"})
	return err
}

func HandleAuthorize(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo users can authorize chats.", &tg.SendOptions{})
		return err
	}

	targetID := m.ChatID()
	args := strings.Fields(m.Args())
	if len(args) > 0 {
		if id, err := strconv.ParseInt(args[0], 10, 64); err == nil {
			targetID = id
		}
	}

	if db.GlobalStore != nil {
		if err := db.GlobalStore.AuthorizeChat(targetID); err != nil {
			_, err = m.Reply("❌ Failed to authorize: "+err.Error(), &tg.SendOptions{})
			return err
		}
	}

	_, err := m.Reply(fmt.Sprintf("✅ Chat/User <code>%d</code> authorized successfully.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}

func HandleUnauthorize(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo users can unauthorize chats.", &tg.SendOptions{})
		return err
	}

	targetID := m.ChatID()
	args := strings.Fields(m.Args())
	if len(args) > 0 {
		if id, err := strconv.ParseInt(args[0], 10, 64); err == nil {
			targetID = id
		}
	}

	if db.GlobalStore != nil {
		if err := db.GlobalStore.UnauthorizeChat(targetID); err != nil {
			_, err = m.Reply("❌ Failed to unauthorize: "+err.Error(), &tg.SendOptions{})
			return err
		}
	}

	_, err := m.Reply(fmt.Sprintf("🚫 Chat/User <code>%d</code> unauthorized.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}

func HandleAddSudo(m *tg.NewMessage) error {
	if !auth.IsOwner(m.SenderID()) {
		_, err := m.Reply("⛔️ Only the bot Owner can add Sudo users.", &tg.SendOptions{})
		return err
	}

	args := strings.Fields(m.Args())
	if len(args) == 0 {
		_, err := m.Reply("⚠️ <b>Usage:</b> <code>/addsudo &lt;user_id&gt;</code>", &tg.SendOptions{ParseMode: "html"})
		return err
	}

	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		_, err = m.Reply("❌ Invalid User ID.", &tg.SendOptions{})
		return err
	}

	if db.GlobalStore != nil {
		if err := db.GlobalStore.AddSudo(targetID, ""); err != nil {
			_, err = m.Reply("❌ Failed to add sudo: "+err.Error(), &tg.SendOptions{})
			return err
		}
	}

	_, err = m.Reply(fmt.Sprintf("👑 User <code>%d</code> added as Sudo.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}

func HandleRemoveSudo(m *tg.NewMessage) error {
	if !auth.IsOwner(m.SenderID()) {
		_, err := m.Reply("⛔️ Only the bot Owner can remove Sudo users.", &tg.SendOptions{})
		return err
	}

	args := strings.Fields(m.Args())
	if len(args) == 0 {
		_, err := m.Reply("⚠️ <b>Usage:</b> <code>/rmsudo &lt;user_id&gt;</code>", &tg.SendOptions{ParseMode: "html"})
		return err
	}

	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		_, err = m.Reply("❌ Invalid User ID.", &tg.SendOptions{})
		return err
	}

	if db.GlobalStore != nil {
		if err := db.GlobalStore.RemoveSudo(targetID); err != nil {
			_, err = m.Reply("❌ Failed to remove sudo: "+err.Error(), &tg.SendOptions{})
			return err
		}
	}

	_, err = m.Reply(fmt.Sprintf("🔻 Sudo privileges removed from <code>%d</code>.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}
