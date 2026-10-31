package task

import (
	"testing"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/pkg/utils"
)

func TestQueueAndManager(t *testing.T) {
	cfg := config.Get()
	cfg.QueueDownload = 2
	cfg.QueueAll = 3

	mgr := InitManager()

	t1 := NewTask(100, "user1", 1, 1, TypeMirror, &utils.TaskOptions{Name: "T1"})
	t2 := NewTask(100, "user1", 1, 2, TypeMirror, &utils.TaskOptions{Name: "T2"})
	t3 := NewTask(200, "user2", 1, 3, TypeMirror, &utils.TaskOptions{Name: "T3"})

	mgr.AddTask(t1)
	mgr.AddTask(t2)
	mgr.AddTask(t3)

	if len(mgr.GetAllTasks()) != 3 {
		t.Fatalf("expected 3 tasks in manager, got %d", len(mgr.GetAllTasks()))
	}

	// Slot acquisition
	if err := mgr.AcquireDownloadSlot(t1); err != nil {
		t.Fatalf("t1 acquire failed: %v", err)
	}
	if err := mgr.AcquireDownloadSlot(t2); err != nil {
		t.Fatalf("t2 acquire failed: %v", err)
	}

	// Force start should bypass limits
	tForce := NewTask(100, "user1", 1, 4, TypeMirror, &utils.TaskOptions{Name: "TForce", ForceStart: true})
	mgr.AddTask(tForce)
	if err := mgr.AcquireDownloadSlot(tForce); err != nil {
		t.Fatalf("tForce acquire failed: %v", err)
	}
	mgr.ReleaseDownloadSlot()

	// Cancellation permission
	success, _ := mgr.CancelTask(t3.GID, 100, false) // user 100 trying to cancel user 200 task without sudo
	if success {
		t.Errorf("expected cancellation failure due to permission check")
	}

	successSudo, _ := mgr.CancelTask(t3.GID, 100, true) // user 100 with sudo
	if !successSudo {
		t.Errorf("expected cancellation success with sudo")
	}

	// CancelAll
	cancelledCount := mgr.CancelAll("all", 100, true)
	if cancelledCount < 2 {
		t.Errorf("expected at least 2 tasks cancelled, got %d", cancelledCount)
	}
}
