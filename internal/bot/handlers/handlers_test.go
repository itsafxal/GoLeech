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

