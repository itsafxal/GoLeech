package handlers

import (
	"strings"
	"testing"

	"github.com/GoLeech/GoLeech/internal/task"
	"github.com/GoLeech/GoLeech/pkg/typography"
	"github.com/GoLeech/GoLeech/pkg/utils"
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
	expected := typography.ToSmallCapsHTML("System Control Dashboard")
	if !strings.Contains(text, expected) {
		t.Errorf("expected '%s' in text, got: %s", expected, text)
	}
	if markup == nil {
		t.Errorf("expected non-nil markup")
	}
}

func TestGetStatsMessage(t *testing.T) {
	stats := GetStatsMessage()
	expectedHeader := typography.ToSmallCapsHTML("System & Architecture Statistics")
	if !strings.Contains(stats, expectedHeader) {
		t.Errorf("expected header '%s' in stats message, got: %s", expectedHeader, stats)
	}
	expectedFramework := typography.ToSmallCapsHTML("Gogram")
	if !strings.Contains(stats, expectedFramework) {
		t.Errorf("expected '%s' in stats message, got: %s", expectedFramework, stats)
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

func TestRenderQueueTaskMessage(t *testing.T) {
	// 1. Queued task
	snapQueued := task.TaskSnapshot{
		GID:       "test1234",
		Name:      "ubuntu-22.04.iso",
		Type:      task.TypeLeech,
		Engine:    "HTTP",
		Status:    task.StatusQueuedDL,
		TotalSize: 1073741824,
	}
	textQ, markupQ := RenderQueueTaskMessage(snapQueued, "")
	if textQ == "" || !strings.Contains(textQ, "ubuntu-22.04.iso") {
		t.Errorf("expected task name in queued message: %s", textQ)
	}
	if markupQ == nil {
		t.Errorf("expected markup with Cancel and Force Start buttons for queued task")
	}

	// 2. Downloading task
	snapDl := task.TaskSnapshot{
		GID:            "test1234",
		Name:           "ubuntu-22.04.iso",
		Type:           task.TypeLeech,
		Engine:         "HTTP",
		Status:         task.StatusDownloading,
		TotalSize:      1073741824,
		ProcessedBytes: 536870912,
		Speed:          10485760,
		ETA:            50,
		ProgressPct:    50.0,
	}
	textDl, markupDl := RenderQueueTaskMessage(snapDl, "")
	if textDl == "" || !strings.Contains(textDl, "50.0%") {
		t.Errorf("expected 50.0%% in download message: %s", textDl)
	}
	if markupDl == nil {
		t.Errorf("expected cancel button for downloading task")
	}

	// 3. Completed task with Cloud URL
	snapDone := task.TaskSnapshot{
		GID:       "test1234",
		Name:      "ubuntu-22.04.iso",
		Type:      task.TypeMirror,
		Engine:    "Pixeldrain",
		Status:    task.StatusCompleted,
		TotalSize: 1073741824,
	}
	textDone, _ := RenderQueueTaskMessage(snapDone, "https://pixeldrain.com/u/abc12345")
	if !strings.Contains(textDone, "https://pixeldrain.com/u/abc12345") {
		t.Errorf("expected cloud link in completed message: %s", textDone)
	}
}

func TestThumbnailStatusIndicator(t *testing.T) {
	expected := typography.ToSmallCaps("Custom Thumbnail")
	textLeech, _ := GetUserSettingsFull(11223344, "leech")
	if !strings.Contains(textLeech, expected) {
		t.Errorf("expected '%s' indicator in leech settings: %s", expected, textLeech)
	}

	textThumb, _ := GetUserSettingsFull(11223344, "thumb")
	if !strings.Contains(textThumb, expected) {
		t.Errorf("expected '%s' indicator in thumb settings: %s", expected, textThumb)
	}
}

func TestNavStack(t *testing.T) {
	uid := int64(999888)
	NavManager.Clear(uid)

	// Scenario A: Leech Settings -> Thumbnail Settings -> Back returns to Leech Settings
	NavManager.Push(uid, "leech")
	NavManager.Push(uid, "thumb")

	last := NavManager.Pop(uid)
	if last != "thumb" {
		t.Errorf("expected popped page 'thumb', got: %s", last)
	}
	prev := NavManager.Pop(uid)
	if prev != "leech" {
		t.Errorf("expected previous page 'leech', got: %s", prev)
	}

	// Scenario B: Mirror Settings -> Metadata Settings -> Back returns to Mirror Settings
	NavManager.Push(uid, "mirror")
	NavManager.Push(uid, "rclone")
	prevMirror := NavManager.Pop(uid)
	if prevMirror != "rclone" {
		t.Errorf("expected 'rclone', got: %s", prevMirror)
	}
	backMirror := NavManager.Pop(uid)
	if backMirror != "mirror" {
		t.Errorf("expected 'mirror', got: %s", backMirror)
	}

	// Hierarchical fallback
	if NavManager.GetParent("thumb") != "leech" {
		t.Errorf("expected parent of 'thumb' to be 'leech'")
	}
	if NavManager.GetParent("rclone") != "mirror" {
		t.Errorf("expected parent of 'rclone' to be 'mirror'")
	}
}

func TestBlockProgressBarAndSpinner(t *testing.T) {
	bar := utils.BlockProgressBar(60.0)
	if bar != "██████░░░░" {
		t.Errorf("expected '██████░░░░', got '%s'", bar)
	}

	bar100 := utils.BlockProgressBar(100.0)
	if bar100 != "██████████" {
		t.Errorf("expected '██████████', got '%s'", bar100)
	}

	f0 := utils.GetSpinnerFrame(0)
	f1 := utils.GetSpinnerFrame(1)
	f2 := utils.GetSpinnerFrame(2)
	f3 := utils.GetSpinnerFrame(3)

	if f0 != "◐" || f1 != "◓" || f2 != "◑" || f3 != "◒" {
		t.Errorf("unexpected spinner frames: %s, %s, %s, %s", f0, f1, f2, f3)
	}
}

