package task

import (
	"testing"
	"time"

	"github.com/GoLeech/GoLeech/pkg/utils"
)

func TestTaskLifecycle(t *testing.T) {
	opts := &utils.TaskOptions{
		Name: "sample.mkv",
		Link: "https://example.com/sample.mkv",
	}

	task := NewTask(1234, "john_doe", -100999, 10, TypeLeech, opts)

	if task.GID == "" {
		t.Errorf("expected non-empty GID")
	}
	if task.Name != "sample.mkv" {
		t.Errorf("expected name 'sample.mkv', got '%s'", task.Name)
	}
	if task.Status != StatusQueuedDL {
		t.Errorf("expected initial status QueueDl, got %s", task.Status)
	}

	// Progress updates
	task.UpdateProgress(500, 1000, 100.0)
	if task.Progress() != 50.0 {
		t.Errorf("expected 50%% progress, got %f", task.Progress())
	}
	if task.ETA != 5 {
		t.Errorf("expected ETA 5s, got %d", task.ETA)
	}

	// Status transitions
	task.SetStatus(StatusDownloading)
	if task.Status != StatusDownloading {
		t.Errorf("expected status Download, got %s", task.Status)
	}

	// Cancellation
	if task.IsCancelled() {
		t.Errorf("expected task not cancelled initially")
	}
	task.Cancel("user requested")
	if !task.IsCancelled() {
		t.Errorf("expected task cancelled after Cancel()")
	}
	select {
	case <-task.Ctx.Done():
		// Success: context was properly cancelled
	case <-time.After(500 * time.Millisecond):
		t.Errorf("context was not cancelled properly")
	}
}
