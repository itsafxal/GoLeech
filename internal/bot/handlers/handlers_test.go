package handlers

import (
	"strings"
	"testing"
)

func TestGetHelpContent(t *testing.T) {
	tabs := []string{"main", "dl", "up", "flags", "settings", "admin"}

	for _, tab := range tabs {
		text, markup := GetHelpContent(tab)
		if text == "" {
			t.Errorf("expected non-empty text for tab %s", tab)
		}
		if markup == nil {
			t.Errorf("expected non-nil markup for tab %s", tab)
		}
	}
}

func TestRenderUserSettings(t *testing.T) {
	text, markup := RenderUserSettings(12345678)
	if !strings.Contains(text, "12345678") {
		t.Errorf("expected user ID 12345678 in text, got: %s", text)
	}
	if markup == nil {
		t.Errorf("expected non-nil markup")
	}
}

func TestRenderBotSettings(t *testing.T) {
	text, markup := RenderBotSettings()
	if !strings.Contains(text, "System Control Dashboard") {
		t.Errorf("expected 'System Control Dashboard' in text, got: %s", text)
	}
	if markup == nil {
		t.Errorf("expected non-nil markup")
	}
}

func TestGetStatsMessage(t *testing.T) {
	stats := GetStatsMessage()
	if !strings.Contains(stats, "System & Architecture Statistics") {
		t.Errorf("expected header in stats message, got: %s", stats)
	}
	if !strings.Contains(stats, "Gogram") {
		t.Errorf("expected Gogram in stats message, got: %s", stats)
	}
}

func TestUserSettingsPages(t *testing.T) {
	pages := []string{
		"main", "general", "leech", "thumb", "ffset",
		"mirror", "rclone", "gdrive", "uphoster", "gofile",
		"buzzheavier", "pixeldrain", "devuploads", "vikingfile",
		"advanced", "clone", "ytdlp", "mega", "seedr",
	}

	for _, page := range pages {
		text, markup := GetUserSettingsFull(99887766, page)
		if text == "" {
			t.Errorf("expected non-empty text for user settings page %s", page)
		}
		if markup == nil {
			t.Errorf("expected non-nil markup for user settings page %s", page)
		}
	}
}

func TestGetOptionMenu(t *testing.T) {
	options := []string{"LEECH_PREFIX", "METADATA", "THUMBNAIL", "NAME_SWAP", "RCLONE_CONFIG"}
	for _, opt := range options {
		text, markup := GetOptionMenu(opt, 99887766)
		if text == "" {
			t.Errorf("expected non-empty text for option %s", opt)
		}
		if markup == nil {
			t.Errorf("expected non-nil markup for option %s", opt)
		}
	}
}

func TestPendingUserInput(t *testing.T) {
	uid := int64(445566)
	SetPendingInput(uid, "LEECH_PREFIX", "leech")

	pending := GetPendingInput(uid)
	if pending == nil || pending.Option != "LEECH_PREFIX" || pending.Category != "leech" {
		t.Errorf("unexpected pending state: %+v", pending)
	}

	ClearPendingInput(uid)
	if GetPendingInput(uid) != nil {
		t.Errorf("expected pending state to be cleared")
	}
}

