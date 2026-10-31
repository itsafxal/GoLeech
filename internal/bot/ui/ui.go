package ui

import (
	"fmt"

	"github.com/GoLeech/GoLeech/pkg/typography"
	tg "github.com/amarnathcjd/gogram/telegram"
)

// UI provides a centralized interface for sending formatted messages and constructing buttons
var Button = &buttonBuilder{}

type buttonBuilder struct{}

// Data creates an inline callback button with Small Caps typography on the label
func (b *buttonBuilder) Data(text, data string) *tg.KeyboardButtonCallback {
	return tg.Button.Data(typography.ToSmallCaps(text), data)
}

// URL creates an inline URL button with Small Caps typography on the label
func (b *buttonBuilder) URL(text, url string) *tg.KeyboardButtonURL {
	return tg.Button.URL(typography.ToSmallCaps(text), url)
}

// Row wraps buttons into a KeyboardButtonRow
func (b *buttonBuilder) Row(buttons ...tg.KeyboardButton) *tg.KeyboardButtonRow {
	return tg.Button.Row(buttons...)
}

// Keyboard constructs an inline reply markup from rows, ensuring all button labels are in Small Caps
func (b *buttonBuilder) Keyboard(rows ...*tg.KeyboardButtonRow) *tg.ReplyInlineMarkup {
	markup := tg.Button.Keyboard(rows...)
	return typography.FormatMarkup(markup).(*tg.ReplyInlineMarkup)
}

// Raw marks user-provided dynamic content (e.g. filenames, user captions) so it is NOT transformed
func Raw(text string) string {
	return fmt.Sprintf("<raw>%s</raw>", text)
}

// ToSmallCaps converts plain text into Small Caps
func ToSmallCaps(text string) string {
	return typography.ToSmallCaps(text)
}

// ToSmallCapsHTML converts HTML text into Small Caps while respecting tags, entities, and raw blocks
func ToSmallCapsHTML(text string) string {
	return typography.ToSmallCapsHTML(text)
}

// FormatMarkup formats all buttons in a markup into Small Caps
func FormatMarkup(markup tg.ReplyMarkup) tg.ReplyMarkup {
	return typography.FormatMarkup(markup)
}

// prepareOptions formats any attached keyboard markup in the SendOptions and ensures HTML parse mode
func prepareOptions(opts []*tg.SendOptions) []*tg.SendOptions {
	if len(opts) == 0 {
		return []*tg.SendOptions{{ParseMode: "html"}}
	}
	for _, opt := range opts {
		if opt != nil {
			if opt.ParseMode == "" {
				opt.ParseMode = "html"
			}
			if opt.ReplyMarkup != nil {
				opt.ReplyMarkup = typography.FormatMarkup(opt.ReplyMarkup)
			}
		}
	}
	return opts
}

// Reply sends a reply message with Small Caps typography applied to all bot text and buttons
func Reply(m *tg.NewMessage, text string, opts ...*tg.SendOptions) (*tg.NewMessage, error) {
	if m == nil {
		return nil, nil
	}
	formattedText := typography.ToSmallCapsHTML(text)
	opts = prepareOptions(opts)
	return m.Reply(formattedText, opts...)
}

// Edit edits an existing message with Small Caps typography applied
func Edit(cb *tg.CallbackQuery, text string, opts ...*tg.SendOptions) (*tg.NewMessage, error) {
	if cb == nil {
		return nil, nil
	}
	formattedText := typography.ToSmallCapsHTML(text)
	opts = prepareOptions(opts)
	return cb.Edit(formattedText, opts...)
}

// EditMsg edits a *tg.NewMessage directly with Small Caps typography applied
func EditMsg(m *tg.NewMessage, text string, opts ...*tg.SendOptions) (*tg.NewMessage, error) {
	if m == nil {
		return nil, nil
	}
	formattedText := typography.ToSmallCapsHTML(text)
	opts = prepareOptions(opts)
	return m.Edit(formattedText, opts...)
}

// Answer responds to a callback query popup/alert with Small Caps typography
func Answer(cb *tg.CallbackQuery, text string, opts ...*tg.CallbackOptions) (bool, error) {
	if cb == nil {
		return false, nil
	}
	formattedText := typography.ToSmallCaps(text)
	return cb.Answer(formattedText, opts...)
}

// SendMessage sends a new message to a peer with Small Caps typography applied
func SendMessage(client *tg.Client, peerID any, text string, opts ...*tg.SendOptions) (*tg.NewMessage, error) {
	if client == nil {
		return nil, nil
	}
	formattedText := typography.ToSmallCapsHTML(text)
	opts = prepareOptions(opts)
	return client.SendMessage(peerID, formattedText, opts...)
}
