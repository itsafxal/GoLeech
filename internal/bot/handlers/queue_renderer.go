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

// RenderQueueTaskMessage formats the live animated interactive queue card for a task
func RenderQueueTaskMessage(snap task.TaskSnapshot, cloudURL string) (string, tg.ReplyMarkup) {
	elapsed := time.Since(snap.StartTime)
	tick := int(elapsed.Seconds() / 2.0)
	spinner := utils.GetSpinnerFrame(tick)

	var header string
	switch snap.Status {
	case task.StatusQueuedDL:
		header = fmt.Sprintf("%s <b><u>Initializing & Queued for Download...</u></b>", spinner)
	case task.StatusQueuedUP:
		header = fmt.Sprintf("%s <b><u>Queued for Upload...</u></b>", spinner)
	case task.StatusDownloading:
		header = fmt.Sprintf("%s <b><u>Downloading Task...</u></b>", spinner)
	case task.StatusProcessing:
		header = fmt.Sprintf("%s <b><u>Processing Media...</u></b>", spinner)
	case task.StatusExtracting:
		header = fmt.Sprintf("%s <b><u>Extracting Archive...</u></b>", spinner)
	case task.StatusArchiving:
		header = fmt.Sprintf("%s <b><u>Archiving Content...</u></b>", spinner)
	case task.StatusSplitting:
		header = fmt.Sprintf("%s <b><u>Splitting Files...</u></b>", spinner)
	case task.StatusUploading:
		header = fmt.Sprintf("%s <b><u>Uploading to Telegram...</u></b>", spinner)
	case task.StatusCompleted:
		header = "✓ <b><u>Completed Successfully</u></b>"
	case task.StatusFailed:
		header = "✗ <b><u>Task Failed</u></b>"
	case task.StatusCancelled:
		header = "🛑 <b><u>Task Cancelled</u></b>"
	default:
		header = fmt.Sprintf("%s <b><u>Transmitting...</u></b>", spinner)
	}

	inMode := "Direct URL"
	if snap.Options != nil {
		if strings.HasPrefix(snap.Options.Link, "tg://") {
			inMode = "Telegram Media"
		} else if strings.HasPrefix(strings.ToLower(snap.Options.Link), "magnet:?") || strings.HasSuffix(strings.ToLower(snap.Options.Link), ".torrent") {
			inMode = "Torrent"
		} else if strings.EqualFold(snap.Engine, "yt-dlp") {
			inMode = "Stream URL"
		}
	}
	outMode := string(snap.Type)
	if outMode == "" {
		outMode = "Leech"
	}

	var sb strings.Builder
	sb.WriteString(header + "\n│\n")
	sb.WriteString(fmt.Sprintf("┠ <b>Task:</b> <code><raw>%s</raw></code>\n", snap.Name))
	sb.WriteString(fmt.Sprintf("┠ <b>GID:</b> <code>%s</code>\n", snap.GID))
	sb.WriteString(fmt.Sprintf("┠ <b>Engine:</b> <i>%s</i>\n", snap.Engine))
	sb.WriteString(fmt.Sprintf("┠ <b>In Mode:</b> <i>%s</i>\n", inMode))
	sb.WriteString(fmt.Sprintf("┠ <b>Out Mode:</b> <i>%s</i>\n", outMode))

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
		sb.WriteString(fmt.Sprintf("┠ <code>%s</code> <i>%.1f%%</i>\n", utils.BlockProgressBar(pct), pct))
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
