package handlers

import (
	"fmt"
	"html"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/internal/plugin"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/utils"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// HandlePing measures round-trip latency
func HandlePing(m *tg.NewMessage) error {
	start := time.Now()
	reply, err := m.Reply("🏓 <i>Pinging Telegram MTProto servers...</i>", &tg.SendOptions{ParseMode: "html"})
	if err != nil {
		log.Printf("⚠️ Error replying to /ping: %v", err)
		return err
	}
	latency := time.Since(start).Milliseconds()
	_, err = reply.Edit(fmt.Sprintf("🏓 <b>Pong!</b> Response Latency: <code>%d ms</code>", latency), &tg.SendOptions{ParseMode: "html"})
	if err != nil {
		log.Printf("⚠️ Error editing /ping reply: %v", err)
	}
	return err
}

// GetStatsMessage builds the comprehensive system statistics text
func GetStatsMessage() string {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	tasks := task.GlobalManager.GetAllTasks()
	activeCount := len(tasks)
	uptime := utils.FormatDuration(time.Since(botStartTime))

	return fmt.Sprintf(`⌬ <b><u>GoLeech — System & Architecture Statistics</u></b>
│
┠ <b>Memory Allocated:</b> <code>%d MB</code>
┠ <b>Total System Memory:</b> <code>%d MB</code>
┠ <b>Active Goroutines:</b> <code>%d</code>
┠ <b>CPU Logical Cores:</b> <code>%d</code>
┠ <b>Active Running Tasks:</b> <code>%d</code>
┠ <b>Instance Uptime:</b> <code>%s</code>
┠ <b>Go Runtime Version:</b> <code>%s</code>
┖ <b>Framework:</b> <code>Gogram v1.7.71 (Native MTProto)</code>`,
		mem.Alloc/1024/1024,
		mem.Sys/1024/1024,
		runtime.NumGoroutine(),
		runtime.NumCPU(),
		activeCount,
		uptime,
		runtime.Version(),
	)
}

// HandleStats responds with detailed system memory and task statistics
func HandleStats(m *tg.NewMessage) error {
	text := GetStatsMessage()
	_, err := m.Reply(text, &tg.SendOptions{
		ParseMode: "html",
		ReplyMarkup: tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("♻️ Refresh Stats", "stats:view"),
			),
		),
	})
	return err
}

// HandleAuthorize authorizes a group chat or user ID
func HandleAuthorize(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo administrators can authorize chats.", &tg.SendOptions{})
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

	_, err := m.Reply(fmt.Sprintf("✅ <b>Authorized:</b> Chat/User <code>%d</code> has been granted access.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}

// HandleUnauthorize revokes authorization from a group chat or user ID
func HandleUnauthorize(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo administrators can unauthorize chats.", &tg.SendOptions{})
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

	_, err := m.Reply(fmt.Sprintf("🚫 <b>Unauthorized:</b> Access revoked for Chat/User <code>%d</code>.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}

// HandleAddSudo grants Sudo admin privileges to a user
func HandleAddSudo(m *tg.NewMessage) error {
	if !auth.IsOwner(m.SenderID()) {
		_, err := m.Reply("⛔️ Only the bot Owner can appoint Sudo users.", &tg.SendOptions{})
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

	_, err = m.Reply(fmt.Sprintf("👑 <b>Sudo Granted:</b> User <code>%d</code> is now a Sudo Administrator.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}

// HandleRemoveSudo revokes Sudo privileges from a user
func HandleRemoveSudo(m *tg.NewMessage) error {
	if !auth.IsOwner(m.SenderID()) {
		_, err := m.Reply("⛔️ Only the bot Owner can revoke Sudo users.", &tg.SendOptions{})
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

	_, err = m.Reply(fmt.Sprintf("🔻 <b>Sudo Revoked:</b> User <code>%d</code> is no longer a Sudo Administrator.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}

// HandleBlacklist bans a user or chat ID from using any bot feature
func HandleBlacklist(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo administrators can blacklist users.", &tg.SendOptions{})
		return err
	}

	args := strings.Fields(m.Args())
	if len(args) == 0 {
		_, err := m.Reply("⚠️ <b>Usage:</b> <code>/blacklist &lt;id&gt; [username]</code>", &tg.SendOptions{ParseMode: "html"})
		return err
	}

	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		_, err = m.Reply("❌ Invalid User ID.", &tg.SendOptions{})
		return err
	}

	username := ""
	if len(args) > 1 {
		username = args[1]
	}

	if db.GlobalStore != nil {
		if err := db.GlobalStore.Blacklist(targetID, username); err != nil {
			_, err = m.Reply("❌ Failed to blacklist: "+err.Error(), &tg.SendOptions{})
			return err
		}
	}

	_, err = m.Reply(fmt.Sprintf("⛔️ <b>Blacklisted:</b> ID <code>%d</code> has been banned from all bot operations.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}

// HandleRemoveBlacklist unbans a user or chat ID
func HandleRemoveBlacklist(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo administrators can manage the blacklist.", &tg.SendOptions{})
		return err
	}

	args := strings.Fields(m.Args())
	if len(args) == 0 {
		_, err := m.Reply("⚠️ <b>Usage:</b> <code>/rmblacklist &lt;id&gt;</code>", &tg.SendOptions{ParseMode: "html"})
		return err
	}

	targetID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		_, err = m.Reply("❌ Invalid User ID.", &tg.SendOptions{})
		return err
	}

	if db.GlobalStore != nil {
		if err := db.GlobalStore.RemoveBlacklist(targetID); err != nil {
			_, err = m.Reply("❌ Failed to remove from blacklist: "+err.Error(), &tg.SendOptions{})
			return err
		}
	}

	_, err = m.Reply(fmt.Sprintf("✅ <b>Unbanned:</b> ID <code>%d</code> has been removed from the blacklist.", targetID), &tg.SendOptions{ParseMode: "html"})
	return err
}

// HandleUsers displays registered user metrics and database statistics
func HandleUsers(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo administrators can view user metrics.", &tg.SendOptions{})
		return err
	}

	if db.GlobalStore == nil {
		_, err := m.Reply("Database connection unavailable.", &tg.SendOptions{})
		return err
	}

	total, sudo, authorized, blacklisted, err := db.GlobalStore.GetUserStats()
	if err != nil {
		_, err = m.Reply("❌ Failed to retrieve user statistics: "+err.Error(), &tg.SendOptions{})
		return err
	}

	text := fmt.Sprintf(`⌬ <b><u>GoLeech — User Registry & Analytics</u></b>
│
┠ <b>Total Known Users:</b> <code>%d</code>
┠ <b>Sudo Administrators:</b> <code>%d</code>
┠ <b>Authorized Chats / Users:</b> <code>%d</code>
┠ <b>Blacklisted Accounts:</b> <code>%d</code>
│
┖ <i>Database Mode: SQLite WAL Mode High Concurrency</i>`,
		total, sudo, authorized, blacklisted)

	_, err = m.Reply(text, &tg.SendOptions{ParseMode: "html"})
	return err
}

// HandleBroadcast sends announcements to all registered private chat users
func HandleBroadcast(m *tg.NewMessage, botClient *tg.Client) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo administrators can initiate broadcasts.", &tg.SendOptions{})
		return err
	}

	broadcastText := m.Args()
	hasReply := m.IsReply()

	if broadcastText == "" && !hasReply {
		msg := `⚠️ <b>Broadcast Usage:</b>
┃ ┠ Send: <code>/broadcast &lt;message&gt;</code>
┃ ┖ Or reply to any message/media with: <code>/broadcast</code>`
		_, err := m.Reply(msg, &tg.SendOptions{ParseMode: "html"})
		return err
	}

	if db.GlobalStore == nil {
		_, err := m.Reply("Database unavailable.", &tg.SendOptions{})
		return err
	}

	users, err := db.GlobalStore.GetAllUsers()
	if err != nil || len(users) == 0 {
		_, err = m.Reply("No registered users found to broadcast to.", &tg.SendOptions{})
		return err
	}

	progressMsg, _ := m.Reply(fmt.Sprintf("🚀 <b>Broadcasting to %d users...</b> Please wait.", len(users)), &tg.SendOptions{ParseMode: "html"})

	go func() {
		start := time.Now()
		sentCount := 0
		failCount := 0

		for _, u := range users {
			if u.IsBlacklisted {
				continue
			}

			var sendErr error
			if hasReply {
				replyMsg, rErr := m.GetReplyMessage()
				if rErr == nil && replyMsg != nil {
					_, sendErr = replyMsg.ForwardTo(u.ID)
				}
			} else {
				_, sendErr = botClient.SendMessage(u.ID, broadcastText, &tg.SendOptions{ParseMode: "html"})
			}

			if sendErr != nil {
				failCount++
			} else {
				sentCount++
			}

			// Rate limiting delay (30 messages per second safety limit)
			time.Sleep(35 * time.Millisecond)
		}

		duration := time.Since(start).Round(time.Millisecond)
		summary := fmt.Sprintf(`⌬ <b><u>Broadcast Transmission Complete</u></b>
│
┠ <b>Total Target Users:</b> <code>%d</code>
┠ <b>Successfully Delivered:</b> <code>%d</code>
┠ <b>Failed / Blocked:</b> <code>%d</code>
┠ <b>Duration:</b> <code>%v</code>
│
┖ <i>Broadcast ID:</i> <code>bc_%d</code>`, len(users), sentCount, failCount, duration, time.Now().Unix())

		if progressMsg != nil {
			_, _ = progressMsg.Edit(summary, &tg.SendOptions{ParseMode: "html"})
		}
	}()

	return nil
}

// HandleLog displays recent logs in an expandable block or provides file upload
func HandleLog(m *tg.NewMessage, botClient *tg.Client) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo administrators can view logs.", &tg.SendOptions{})
		return err
	}

	logFile := "goleech.log"
	data, err := os.ReadFile(logFile)
	if err != nil || len(data) == 0 {
		// Fallback to active runtime status
		text := fmt.Sprintf(`⌬ <b><u>GoLeech — System Runtime Log</u></b>
│
┠ <b>Log File:</b> <i>goleech.log</i> (stdout capture active)
┠ <b>Uptime:</b> <code>%s</code>
┠ <b>Active Goroutines:</b> <code>%d</code>
│
<blockquote expandable>[INFO] Database connected in SQLite WAL Mode
[INFO] Task & Queue Manager running
[INFO] MTProto update stream active
[INFO] All systems operational</blockquote>`, utils.FormatDuration(time.Since(botStartTime)), runtime.NumGoroutine())

		_, err = m.Reply(text, &tg.SendOptions{ParseMode: "html"})
		return err
	}

	lines := strings.Split(string(data), "\n")
	startLine := 0
	if len(lines) > 35 {
		startLine = len(lines) - 35
	}
	recent := strings.Join(lines[startLine:], "\n")

	text := fmt.Sprintf(`⌬ <b><u>GoLeech — System Log (Last %d lines)</u></b>
│
<blockquote expandable>%s</blockquote>`, len(lines)-startLine, html.EscapeString(recent))

	_, err = m.Reply(text, &tg.SendOptions{
		ParseMode: "html",
		ReplyMarkup: tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("❌ Close", "help:close"),
			),
		),
	})
	return err
}

// HandleRestart offers confirmation before gracefully restarting the bot instance
func HandleRestart(m *tg.NewMessage) error {
	if !auth.IsSudo(m.SenderID()) {
		_, err := m.Reply("⛔️ Only Sudo administrators can restart the bot.", &tg.SendOptions{})
		return err
	}

	text := `⚠️ <b><u>Confirm Bot Restart</u></b>
│
┠ Are you sure you want to restart the <b>GoLeech</b> process?
┠ Active transfers will be safely stopped and saved.
┖ The supervisor (Docker / systemd) will immediately launch a clean instance.`

	markup := tg.Button.Keyboard(
		tg.Button.Row(
			tg.Button.Data("✅ Confirm Restart", "bot:restart_confirm"),
			tg.Button.Data("❌ Cancel", "help:close"),
		),
	)

	_, err := m.Reply(text, &tg.SendOptions{
		ParseMode:   "html",
		ReplyMarkup: markup,
	})
	return err
}

// HandleMemory displays detailed Go runtime memory and performance metrics
func HandleMemory(m *tg.NewMessage) error {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	text := fmt.Sprintf(`🧠 <b><u>GoLeech — Runtime Memory & Heap Stats</u></b>
│
┠ <b>Allocated Memory:</b> <code>%s</code>
┠ <b>Total Allocated:</b> <code>%s</code>
┠ <b>System OS Memory:</b> <code>%s</code>
┠ <b>Heap In-Use:</b> <code>%s</code>
┠ <b>Heap Idle:</b> <code>%s</code>
┠ <b>Heap Objects:</b> <code>%d</code>
┠ <b>Completed GC Cycles:</b> <code>%d</code>
┠ <b>Active Goroutines:</b> <code>%d</code>
┖ <b>Logical CPUs:</b> <code>%d</code>`,
		utils.FormatBytes(int64(mem.Alloc)),
		utils.FormatBytes(int64(mem.TotalAlloc)),
		utils.FormatBytes(int64(mem.Sys)),
		utils.FormatBytes(int64(mem.HeapAlloc)),
		utils.FormatBytes(int64(mem.HeapIdle)),
		mem.HeapObjects,
		mem.NumGC,
		runtime.NumGoroutine(),
		runtime.NumCPU(),
	)

	_, err := m.Reply(text, &tg.SendOptions{
		ParseMode: "html",
		ReplyMarkup: tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("❌ Close", "help:close"),
			),
		),
	})
	return err
}

// HandlePlugins displays all loaded system extensions
func HandlePlugins(m *tg.NewMessage) error {
	var plugins []plugin.Plugin
	if plugin.GlobalRegistry != nil {
		plugins = plugin.GlobalRegistry.ListPlugins()
	}

	if len(plugins) == 0 {
		_, err := m.Reply("🧩 <b>Loaded Plugins:</b> <i>No external plugins currently loaded.</i>", &tg.SendOptions{ParseMode: "html"})
		return err
	}

	var sb strings.Builder
	sb.WriteString("🧩 <b><u>Active GoLeech Plugins</u></b>\n│\n")
	for i, p := range plugins {
		sb.WriteString(fmt.Sprintf("┠ <b>%d. %s</b> (v%s)\n┃ ┖ <i>%s</i>\n", i+1, html.EscapeString(p.Name()), html.EscapeString(p.Version()), html.EscapeString(p.Description())))
	}
	sb.WriteString("┖ <i>All plugins run concurrently in native Go goroutines.</i>")

	_, err := m.Reply(sb.String(), &tg.SendOptions{
		ParseMode: "html",
		ReplyMarkup: tg.Button.Keyboard(
			tg.Button.Row(
				tg.Button.Data("❌ Close", "help:close"),
			),
		),
	})
	return err
}
