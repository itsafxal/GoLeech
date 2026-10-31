package handlers

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/GoLeech/GoLeech/internal/auth"
	"github.com/GoLeech/GoLeech/internal/bot/ui"
	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/search"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// HandleSearch performs live torrent and public source index searching
func HandleSearch(m *tg.NewMessage) error {
	senderID := m.SenderID()
	chatID := m.ChatID()

	if !auth.CanExecute(senderID, chatID) {
		_, err := ui.Reply(m, "⛔️ You are not authorized to use the search engine.")
		return err
	}

	cfg := config.Get()
	if cfg.DisableSearch {
		_, err := ui.Reply(m, "⚠️ Torrent and source search is currently disabled by administrator.")
		return err
	}

	query := strings.TrimSpace(m.Args())
	if query == "" {
		usage := `⚠️ <b>Search Usage:</b>
┃ ┠ <code>/search &lt;query&gt;</code>
┃ ┖ <b>Example:</b> <code>/search Ubuntu 24.04 ISO</code>`
		_, err := ui.Reply(m, usage)
		return err
	}

	searchWaitMsg, _ := ui.Reply(m, fmt.Sprintf("🔍 <i>Searching for <b><raw>%s</raw></b> across indexers...</i>", html.EscapeString(query)))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	provider := search.NewAPISearchProvider("")
	results, err := provider.Search(ctx, query)

	if err != nil || len(results) == 0 {
		failMsg := fmt.Sprintf("❌ No search results found for: <code><raw>%s</raw></code>", html.EscapeString(query))
		if searchWaitMsg != nil {
			_, _ = ui.EditMsg(searchWaitMsg, failMsg)
		} else {
			_, _ = ui.Reply(m, failMsg)
		}
		return nil
	}

	// Limit to top 5 results for clean Telegram UI display
	maxResults := 5
	if len(results) < maxResults {
		maxResults = len(results)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⌬ <b><u>Search Results for:</u></b> <i><raw>%s</raw></i>\n│\n", html.EscapeString(query)))

	for i := 0; i < maxResults; i++ {
		r := results[i]
		idx := i + 1
		sb.WriteString(fmt.Sprintf("<b>%d.</b> <b><raw>%s</raw></b>\n", idx, html.EscapeString(r.Name)))
		sb.WriteString(fmt.Sprintf("┟ <b>Size:</b> <code>%s</code> | <b>Seeds:</b> <code>%d</code> | <b>Leech:</b> <code>%d</code>\n", r.Size, r.Seeders, r.Leechers))
		sb.WriteString(fmt.Sprintf("┖ <b>Magnet:</b> <code><raw>%s</raw></code>\n\n", html.EscapeString(r.Magnet)))
	}

	sb.WriteString("<i>💡 Copy the magnet link above and send /leech or /mirror to start downloading!</i>")

	markup := ui.Button.Keyboard(
		ui.Button.Row(
			ui.Button.Data("❌ Close", "help:close"),
		),
	)

	if searchWaitMsg != nil {
		_, err = ui.EditMsg(searchWaitMsg, sb.String(), &tg.SendOptions{
			ReplyMarkup: markup,
		})
	} else {
		_, err = ui.Reply(m, sb.String(), &tg.SendOptions{
			ReplyMarkup: markup,
		})
	}

	return err
}

