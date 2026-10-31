package handlers

import (
	"fmt"
	"html"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/typography"
	"github.com/GoLeech/GoLeech/pkg/utils"
	tg "github.com/amarnathcjd/gogram/telegram"
)

var (
	botStartTime = time.Now()
	activeStatus sync.Map // map[int64]*StatusSession
)

type StatusSession struct {
	ChatID    int64
	MessageID int32
	Page      int
	Filter    string
	LastText  string
	mu        sync.Mutex
}

// StartStatusUpdater runs a background worker refreshing active status messages
func StartStatusUpdater(client *tg.Client) {
	ticker := time.NewTicker(time.Duration(config.Get().StatusUpdateInterval) * time.Second)
	go func() {
		for range ticker.C {
			activeStatus.Range(func(key, val interface{}) bool {
				sess := val.(*StatusSession)
				go updateSessionMessage(client, sess)
				return true
			})
		}
	}()
}

// RegisterStatusSession tracks a status message for periodic editing
func RegisterStatusSession(chatID int64, msgID int32) {
	activeStatus.Store(chatID, &StatusSession{
		ChatID:    chatID,
		MessageID: msgID,
		Page:      1,
		Filter:    "ALL",
	})
}

func updateSessionMessage(client *tg.Client, sess *StatusSession) {
	sess.mu.Lock()
	defer sess.mu.Unlock()

	text, markup := GenerateStatusMessage(sess.Page, sess.Filter)
	if text == "" {
		text = "No active tasks at this moment."
	}

	if text == sess.LastText {
		return // skip edit if unchanged
	}

	_, err := client.EditMessage(sess.ChatID, sess.MessageID, text, &tg.SendOptions{
		ParseMode:   "html",
		ReplyMarkup: markup,
	})
	if err == nil {
		sess.LastText = text
	}
}

// GenerateStatusMessage builds the formatted status HTML string and inline keyboard
func GenerateStatusMessage(page int, filter string) (string, tg.ReplyMarkup) {
	cfg := config.Get()
	tasks := task.GlobalManager.GetAllTasks()

	// Apply filter
	var filtered []*task.Task
	for _, t := range tasks {
		if filter == "ALL" {
			filtered = append(filtered, t)
		} else if filter == "DL" && (t.Status == task.StatusDownloading || t.Status == task.StatusQueuedDL) {
			filtered = append(filtered, t)
		} else if filter == "UP" && (t.Status == task.StatusUploading || t.Status == task.StatusQueuedUP) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) == 0 {
		emptyMsg := typography.ToSmallCapsHTML("<b>No Active Tasks!</b>\n\n" + getBotStatsFooter())
		return emptyMsg, typography.FormatMarkup(getControlButtons(1, 1, filter))
	}

	limit := cfg.StatusLimit
	if limit <= 0 {
		limit = 10
	}

	totalPages := (len(filtered) + limit - 1) / limit
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * limit
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	var sb strings.Builder
	for i, t := range filtered[start:end] {
		idx := start + i + 1
		sb.WriteString(fmt.Sprintf("<b>%d.</b> <b><i><raw>%s</raw></i></b>\n", idx, html.EscapeString(t.Name)))
		if t.UserName != "" {
			sb.WriteString(fmt.Sprintf("<b>Task By:</b> @%s (<code>#ID%d</code>)\n", t.UserName, t.UserID))
		}

		pct := t.Progress()
		sb.WriteString(fmt.Sprintf("┟ %s <i>%.1f%%</i>\n", utils.ProgressBar(pct), pct))
		sb.WriteString(fmt.Sprintf("┠ <b>Processed</b> → <i>%s of %s</i>\n", utils.FormatBytes(t.ProcessedBytes), utils.FormatBytes(t.TotalSize)))
		sb.WriteString(fmt.Sprintf("┠ <b>Status</b> → <b>%s</b>\n", t.Status))
		sb.WriteString(fmt.Sprintf("┠ <b>Speed</b> → <i>%s</i>\n", utils.FormatSpeed(t.Speed)))
		sb.WriteString(fmt.Sprintf("┠ <b>ETA</b> → <i>%s</i>\n", utils.FormatSeconds(t.ETA)))
		sb.WriteString(fmt.Sprintf("┠ <b>Engine</b> → <i>%s</i>\n", t.Engine))
		sb.WriteString(fmt.Sprintf("<b>┖ Stop</b> → <i>/c_%s</i>\n\n", t.GID))
	}

	sb.WriteString("⌬ <b><u>Bot Stats</u></b>\n")
	sb.WriteString(getBotStatsFooter())

	return typography.ToSmallCapsHTML(sb.String()), typography.FormatMarkup(getControlButtons(page, totalPages, filter))
}

func getBotStatsFooter() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ramMB := m.Alloc / 1024 / 1024
	uptime := utils.FormatDuration(time.Since(botStartTime))

	return fmt.Sprintf("┟ <b>RAM</b> → %d MB | <b>Goroutines</b> → %d\n┖ <b>Uptime</b> → %s", ramMB, runtime.NumGoroutine(), uptime)
}

func getControlButtons(page, totalPages int, filter string) tg.ReplyMarkup {
	var rows []*tg.KeyboardButtonRow

	// Pagination row if multiple pages
	if totalPages > 1 {
		prevPage := page - 1
		if prevPage < 1 {
			prevPage = totalPages
		}
		nextPage := page + 1
		if nextPage > totalPages {
			nextPage = 1
		}
		rows = append(rows, tg.Button.Row(
			tg.Button.Data("<<", fmt.Sprintf("status:page:%d", prevPage)),
			tg.Button.Data(fmt.Sprintf("%d/%d", page, totalPages), "status:refresh"),
			tg.Button.Data(">>", fmt.Sprintf("status:page:%d", nextPage)),
		))
	}

	// Filter & Refresh row
	rows = append(rows, tg.Button.Row(
		tg.Button.Data("ALL", "status:filter:ALL"),
		tg.Button.Data("DL", "status:filter:DL"),
		tg.Button.Data("UP", "status:filter:UP"),
		tg.Button.Data("♻️ Refresh", "status:refresh"),
	))

	return tg.Button.Keyboard(rows...)
}
