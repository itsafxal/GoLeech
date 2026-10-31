package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/bot/ui"
	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/typography"
	"github.com/GoLeech/GoLeech/pkg/utils"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// RenderQueueTaskMessage formats the live interactive queue card for a task
func RenderQueueTaskMessage(snap task.TaskSnapshot, cloudURL string) (string, tg.ReplyMarkup) {
	var header string
	switch snap.Status {
	case task.StatusQueuedDL:
		header = "⏳ <b><u>Task Queued for Download</u></b>"
	case task.StatusQueuedUP:
		header = "⏳ <b><u>Task Queued for Upload</u></b>"
	case task.StatusDownloading:
		header = "📥 <b><u>Downloading Task</u></b>"
	case task.StatusProcessing:
		header = "⚙️ <b><u>Processing Media</u></b>"
	case task.StatusExtracting:
		header = "📂 <b><u>Extracting Archive</u></b>"
	case task.StatusArchiving:
		header = "📦 <b><u>Archiving Content</u></b>"
	case task.StatusSplitting:
		header = "✂️ <b><u>Splitting Files</u></b>"
	case task.StatusUploading:
		header = "📤 <b><u>Uploading Task</u></b>"
	case task.StatusCompleted:
		header = "✅ <b><u>Task Completed!</u></b>"
	case task.StatusFailed:
		header = "❌ <b><u>Task Failed</u></b>"
	case task.StatusCancelled:
		header = "🛑 <b><u>Task Cancelled</u></b>"
	default:
		header = "⚡️ <b><u>Task In Progress</u></b>"
	}

	var sb strings.Builder
	sb.WriteString(header + "\n│\n")
	sb.WriteString(fmt.Sprintf("┠ <b>Task:</b> <code><raw>%s</raw></code>\n", snap.Name))
	sb.WriteString(fmt.Sprintf("┠ <b>GID:</b> <code>%s</code>\n", snap.GID))
	sb.WriteString(fmt.Sprintf("┠ <b>Type:</b> <i>%s</i>\n", snap.Type))
	sb.WriteString(fmt.Sprintf("┠ <b>Engine:</b> <i>%s</i>\n", snap.Engine))

	elapsed := time.Since(snap.StartTime)

	var markup tg.ReplyMarkup

	switch snap.Status {
	case task.StatusQueuedDL, task.StatusQueuedUP:
		sb.WriteString(fmt.Sprintf("┠ <b>Status:</b> <b>%s</b>\n", snap.Status))
		sb.WriteString(fmt.Sprintf("┖ <b>Elapsed:</b> <i>%s</i>", utils.FormatDuration(elapsed)))

		markup = tg.Button.Keyboard(
			tg.Button.Row(
				ui.Button.Data("⚡️ Force Start", fmt.Sprintf("task:fs:%s", snap.GID)),
				ui.Button.Data("🚫 Cancel Task", fmt.Sprintf("cancel:%s", snap.GID)),
			),
		)

	case task.StatusCompleted:
		sb.WriteString(fmt.Sprintf("┠ <b>Size:</b> <i>%s</i>\n", utils.FormatBytes(snap.TotalSize)))
		sb.WriteString(fmt.Sprintf("┖ <b>Elapsed:</b> <i>%s</i>", utils.FormatDuration(elapsed)))
		if cloudURL != "" {
			sb.WriteString(fmt.Sprintf("\n\n🔗 <b>Cloud Link:</b> <raw>%s</raw>", cloudURL))
		}
		markup = nil

	case task.StatusFailed, task.StatusCancelled:
		sb.WriteString(fmt.Sprintf("┠ <b>Status:</b> <b>%s</b>\n", snap.Status))
		sb.WriteString(fmt.Sprintf("┖ <b>Elapsed:</b> <i>%s</i>", utils.FormatDuration(elapsed)))
		if snap.ErrorMsg != "" {
			sb.WriteString(fmt.Sprintf("\n\n⚠️ <b>Reason:</b> <code><raw>%s</raw></code>", snap.ErrorMsg))
		}
		markup = nil

	default: // Downloading, Processing, Uploading, etc.
		sb.WriteString(fmt.Sprintf("┠ <b>Status:</b> <b>%s</b>\n", snap.Status))
		pct := snap.ProgressPct
		sb.WriteString(fmt.Sprintf("┠ %s <i>%.1f%%</i>\n", utils.ProgressBar(pct), pct))
		sb.WriteString(fmt.Sprintf("┠ <b>Processed:</b> <i>%s of %s</i>\n", utils.FormatBytes(snap.ProcessedBytes), utils.FormatBytes(snap.TotalSize)))
		sb.WriteString(fmt.Sprintf("┠ <b>Speed:</b> <i>%s</i>\n", utils.FormatSpeed(snap.Speed)))
		sb.WriteString(fmt.Sprintf("┠ <b>ETA:</b> <i>%s</i>\n", utils.FormatSeconds(snap.ETA)))
		sb.WriteString(fmt.Sprintf("┖ <b>Elapsed:</b> <i>%s</i>", utils.FormatDuration(elapsed)))

		markup = tg.Button.Keyboard(
			tg.Button.Row(
				ui.Button.Data("🚫 Cancel Task", fmt.Sprintf("cancel:%s", snap.GID)),
			),
		)
	}

	formattedText := typography.ToSmallCapsHTML(sb.String())
	if markup != nil {
		markup = typography.FormatMarkup(markup)
	}

	return formattedText, markup
}
