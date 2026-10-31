package ui

import (
	"strings"
	"testing"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func TestUIButtons(t *testing.T) {
	btn := Button.Data("⚙️ General Settings", "userset 12345 general")
	if btn.Text != "⚙️ ɢᴇɴᴇʀᴀʟ ꜱᴇᴛᴛɪɴɢꜱ" {
		t.Errorf("expected small caps button text, got %q", btn.Text)
	}
	if string(btn.Data) != "userset 12345 general" {
		t.Errorf("callback data was modified: %s", string(btn.Data))
	}

	keyboard := Button.Keyboard(
		Button.Row(btn),
		Button.Row(Button.URL("Website", "https://google.com")),
	)

	if len(keyboard.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(keyboard.Rows))
	}

	urlBtn := keyboard.Rows[1].Buttons[0].(*tg.KeyboardButtonURL)
	if urlBtn.Text != "ᴡᴇʙꜱɪᴛᴇ" {
		t.Errorf("expected small caps URL text, got %q", urlBtn.Text)
	}
}

func TestUIRawPreservation(t *testing.T) {
	userFilename := "Movie.2024.1080p.WEBRip.mkv"
	userCaption := "Uploaded by User123"
	botMsg := "<b>Leech Complete!</b>\nFilename: " + Raw(userFilename) + "\nCaption: " + Raw(userCaption)

	formatted := ToSmallCapsHTML(botMsg)
	if !strings.Contains(formatted, userFilename) {
		t.Errorf("expected raw user filename to be preserved, got: %s", formatted)
	}
	if !strings.Contains(formatted, userCaption) {
		t.Errorf("expected raw user caption to be preserved, got: %s", formatted)
	}
	if strings.Contains(formatted, "<raw>") || strings.Contains(formatted, "</raw>") {
		t.Errorf("expected <raw> tags to be stripped, got: %s", formatted)
	}
	if !strings.Contains(formatted, "ʟᴇᴇᴄʜ ᴄᴏᴍᴘʟᴇᴛᴇ!") {
		t.Errorf("expected bot text to be in small caps, got: %s", formatted)
	}
}
