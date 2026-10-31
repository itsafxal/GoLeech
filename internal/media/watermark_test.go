package media

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewWatermarkConfigFromMap(t *testing.T) {
	// 1. Defaults
	emptyMap := func(k string) string { return "" }
	defCfg := NewWatermarkConfigFromMap(emptyMap)
	if defCfg.Enabled {
		t.Errorf("expected Enabled to be false by default")
	}
	if defCfg.Text != "Join Our Channel" {
		t.Errorf("expected default text 'Join Our Channel', got %s", defCfg.Text)
	}
	if defCfg.Position != "bottom_center" {
		t.Errorf("expected default position 'bottom_center', got %s", defCfg.Position)
	}
	if defCfg.FontSize != 24 {
		t.Errorf("expected default font size 24, got %d", defCfg.FontSize)
	}
	if defCfg.DurationSec != 5 {
		t.Errorf("expected default duration 5, got %d", defCfg.DurationSec)
	}
	if defCfg.IntervalSec != 120 {
		t.Errorf("expected default interval 120, got %d", defCfg.IntervalSec)
	}
	if defCfg.InitialDelay != 30 {
		t.Errorf("expected default initial delay 30, got %d", defCfg.InitialDelay)
	}

	// 2. Custom values
	customMap := func(k string) string {
		switch k {
		case "WATERMARK_ENABLED":
			return "true"
		case "WATERMARK_TEXT":
			return "Subscribe to @MyChannel"
		case "WATERMARK_POS":
			return "bottom_right"
		case "WATERMARK_SIZE":
			return "32"
		case "WATERMARK_COLOR":
			return "yellow"
		case "WATERMARK_DURATION":
			return "10"
		case "WATERMARK_INTERVAL":
			return "300"
		case "WATERMARK_DELAY":
			return "60"
		default:
			return ""
		}
	}

	customCfg := NewWatermarkConfigFromMap(customMap)
	if !customCfg.Enabled {
		t.Errorf("expected custom Enabled true")
	}
	if customCfg.Text != "Subscribe to @MyChannel" {
		t.Errorf("expected custom text, got %s", customCfg.Text)
	}
	if customCfg.Position != "bottom_right" {
		t.Errorf("expected custom position bottom_right, got %s", customCfg.Position)
	}
	if customCfg.FontSize != 32 {
		t.Errorf("expected custom font size 32, got %d", customCfg.FontSize)
	}
	if customCfg.FontColor != "yellow" {
		t.Errorf("expected custom color yellow, got %s", customCfg.FontColor)
	}
	if customCfg.DurationSec != 10 {
		t.Errorf("expected custom duration 10, got %d", customCfg.DurationSec)
	}
	if customCfg.IntervalSec != 300 {
		t.Errorf("expected custom interval 300, got %d", customCfg.IntervalSec)
	}
	if customCfg.InitialDelay != 60 {
		t.Errorf("expected custom delay 60, got %d", customCfg.InitialDelay)
	}
}

func TestApplyVideoWatermarkDisabled(t *testing.T) {
	ctx := context.Background()
	// When disabled, should return input path without processing
	res, err := ApplyVideoWatermark(ctx, "sample.mp4", WatermarkConfig{Enabled: false}, nil)
	if err != nil {
		t.Errorf("unexpected error when disabled: %v", err)
	}
	if res != "sample.mp4" {
		t.Errorf("expected sample.mp4, got %s", res)
	}
}

func TestApplyVideoWatermarkNonVideo(t *testing.T) {
	ctx := context.Background()
	tmpFile := filepath.Join(t.TempDir(), "document.pdf")
	_ = os.WriteFile(tmpFile, []byte("dummy pdf content"), 0644)

	res, err := ApplyVideoWatermark(ctx, tmpFile, WatermarkConfig{
		Enabled: true,
		Text:    "Watermark",
	}, nil)
	if err != nil {
		t.Errorf("unexpected error on non-video file: %v", err)
	}
	if res != tmpFile {
		t.Errorf("expected non-video file to pass through untouched: %s vs %s", res, tmpFile)
	}
}
