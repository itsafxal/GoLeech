package typography

import (
	"strings"
	"testing"

	tg "github.com/amarnathcjd/gogram/telegram"
)

func TestToSmallCaps(t *testing.T) {
	input := "Cyber TG"
	expected := "ᴄʏʙᴇʀ ᴛɢ"
	actual := ToSmallCaps(input)
	if actual != expected {
		t.Errorf("expected %q, got %q", expected, actual)
	}

	alphabet := "abcdefghijklmnopqrstuvwxyz"
	scAlphabet := ToSmallCaps(alphabet)
	if scAlphabet != "ᴀʙᴄᴅᴇғɢʜɪᴊᴋʟᴍɴᴏᴘǫʀꜱᴛᴜᴠᴡxʏᴢ" {
		t.Errorf("alphabet small caps mismatch: %s", scAlphabet)
	}

	// Numbers and emojis preserved
	mixed := "Mode: 🎥 Media [Split: 2 GB]"
	scMixed := ToSmallCaps(mixed)
	if !strings.Contains(scMixed, "🎥") || !strings.Contains(scMixed, "2") {
		t.Errorf("emojis or numbers altered in %q", scMixed)
	}
}

func TestToSmallCapsHTML(t *testing.T) {
	input := "<b><u>Personal Leech &amp; Mirror Settings</u></b>\nFilename: <raw>Ubuntu.24.04.iso</raw>\nLink: https://t.me/GoLeech"
	actual := ToSmallCapsHTML(input)

	// HTML tags preserved
	if !strings.HasPrefix(actual, "<b><u>") || !strings.Contains(actual, "</u></b>") {
		t.Errorf("HTML tags damaged: %s", actual)
	}

	// Entity preserved
	if !strings.Contains(actual, "&amp;") {
		t.Errorf("&amp; entity damaged: %s", actual)
	}

	// User raw content untouched
	if !strings.Contains(actual, "Ubuntu.24.04.iso") {
		t.Errorf("user raw content was altered: %s", actual)
	}

	// URL preserved
	if !strings.Contains(actual, "https://t.me/GoLeech") {
		t.Errorf("URL was altered: %s", actual)
	}

	// Bot text converted
	if !strings.Contains(actual, "ᴘᴇʀꜱᴏɴᴀʟ ʟᴇᴇᴄʜ") {
		t.Errorf("bot text not converted to small caps: %s", actual)
	}
}

func TestFormatMarkup(t *testing.T) {
	markup := tg.Button.Keyboard(
		tg.Button.Row(
			tg.Button.Data("⚙️ General Settings", "userset 12345 general"),
			tg.Button.Data("Close", "userset 12345 close"),
		),
		tg.Button.Row(
			tg.Button.URL("Website", "https://google.com"),
		),
	)

	formatted := FormatMarkup(markup)
	inline, ok := formatted.(*tg.ReplyInlineMarkup)
	if !ok {
		t.Fatal("expected *tg.ReplyInlineMarkup")
	}

	btn1 := inline.Rows[0].Buttons[0].(*tg.KeyboardButtonCallback)
	if btn1.Text != "⚙️ ɢᴇɴᴇʀᴀʟ ꜱᴇᴛᴛɪɴɢꜱ" {
		t.Errorf("expected small caps button label, got %q", btn1.Text)
	}
	// Callback data must remain ASCII
	if string(btn1.Data) != "userset 12345 general" {
		t.Errorf("callback data was modified: %s", string(btn1.Data))
	}

	btnURL := inline.Rows[1].Buttons[0].(*tg.KeyboardButtonURL)
	if btnURL.Text != "ᴡᴇʙꜱɪᴛᴇ" {
		t.Errorf("expected small caps URL button, got %q", btnURL.Text)
	}
	if btnURL.URL != "https://google.com" {
		t.Errorf("button URL was altered: %s", btnURL.URL)
	}
}
