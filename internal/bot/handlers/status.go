package handlers

import (
	tg "github.com/amarnathcjd/gogram/telegram"
)

func HandleStatus(m *tg.NewMessage) error {
	text, markup := GenerateStatusMessage(1, "ALL")
	sentMsg, err := m.Reply(text, &tg.SendOptions{
		ParseMode:   "html",
		ReplyMarkup: markup,
	})
	if err == nil && sentMsg != nil {
		RegisterStatusSession(m.ChatID(), sentMsg.ID)
	}
	return err
}
