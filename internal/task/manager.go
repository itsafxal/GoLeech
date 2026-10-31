package task

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
	"github.com/GoLeech/GoLeech/pkg/utils"
)

type Manager struct {
	tasks     map[string]*Task
	tasksMu   sync.RWMutex
	dlRunning int
	upRunning int
	queueMu   sync.Mutex
	dlQueueCh chan *Task
	upQueueCh chan *Task
}

var GlobalManager *Manager

func InitManager() *Manager {
	mgr := &Manager{
		tasks:     make(map[string]*Task),
		dlQueueCh: make(chan *Task, 1000),
		upQueueCh: make(chan *Task, 1000),
	}
	GlobalManager = mgr
	return mgr
}

func (m *Manager) AddTask(t *Task) {
	m.tasksMu.Lock()
	m.tasks[t.GID] = t
	m.tasksMu.Unlock()

	if db.GlobalStore != nil {
		_ = db.GlobalStore.SaveTask(&db.TaskRecord{
			GID:       t.GID,
			UserID:    t.UserID,
			Name:      t.Name,
			Type:      string(t.Type),
			Engine:    t.Engine,
			Status:    string(t.Status),
			Size:      t.TotalSize,
			CreatedAt: t.StartTime,
			UpdatedAt: time.Now(),
		})
	}
}

func (m *Manager) GetTask(gid string) *Task {
	if m == nil {
		return nil
	}
	m.tasksMu.RLock()
	defer m.tasksMu.RUnlock()

	// Direct match
	if t, exists := m.tasks[gid]; exists {
		return t
	}

	// Prefix match
	for id, t := range m.tasks {
		if strings.HasPrefix(id, gid) {
			return t
		}
	}
	return nil
}

func (m *Manager) GetAllTasks() []*Task {
	if m == nil {
		return nil
	}
	m.tasksMu.RLock()
	defer m.tasksMu.RUnlock()
	list := make([]*Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		list = append(list, t)
	}
	return list
}

func (m *Manager) GetTasksByStatus(st TaskStatus) []*Task {
	m.tasksMu.RLock()
	defer m.tasksMu.RUnlock()
	var list []*Task
	for _, t := range m.tasks {
		if t.Status == st {
			list = append(list, t)
		}
	}
	return list
}

func (m *Manager) RemoveTask(gid string) {
	m.tasksMu.Lock()
	t, exists := m.tasks[gid]
	if exists {
		delete(m.tasks, gid)
	}
	m.tasksMu.Unlock()

	if exists && t.LocalPath != "" {
		_ = os.RemoveAll(t.LocalPath)
	}

	if exists && db.GlobalStore != nil {
		_ = db.GlobalStore.SaveTask(&db.TaskRecord{
			GID:       t.GID,
			UserID:    t.UserID,
			Name:      t.Name,
			Type:      string(t.Type),
			Engine:    t.Engine,
			Status:    string(t.Status),
			Size:      t.TotalSize,
			CreatedAt: t.StartTime,
			UpdatedAt: time.Now(),
		})
	}
}

// AcquireDownloadSlot waits until download concurrency limits allow running
func (m *Manager) AcquireDownloadSlot(t *Task) error {
	cfg := config.Get()

	for {
		if t.IsCancelled() {
			return fmt.Errorf("task cancelled while queued")
		}

		m.queueMu.Lock()
		canRun := true
		if t.Options != nil && t.Options.ForceStart {
			canRun = true
		} else {
			if cfg.QueueDownload > 0 && m.dlRunning >= cfg.QueueDownload {
				canRun = false
			}
			if cfg.QueueAll > 0 && (m.dlRunning+m.upRunning) >= cfg.QueueAll {
				canRun = false
			}
		}

		if canRun {
			m.dlRunning++
			m.queueMu.Unlock()
			return nil
		}
		m.queueMu.Unlock()

		select {
		case <-t.Ctx.Done():
			return fmt.Errorf("task cancelled")
		case <-time.After(2 * time.Second):
		}
	}
}

func (m *Manager) ReleaseDownloadSlot() {
	m.queueMu.Lock()
	if m.dlRunning > 0 {
		m.dlRunning--
	}
	m.queueMu.Unlock()
}

// AcquireUploadSlot waits until upload concurrency limits allow running
func (m *Manager) AcquireUploadSlot(t *Task) error {
	cfg := config.Get()
	t.SetStatus(StatusQueuedUP)

	for {
		if t.IsCancelled() {
			return fmt.Errorf("task cancelled while queued for upload")
		}

		m.queueMu.Lock()
		canRun := true
		if t.Options != nil && t.Options.ForceStart {
			canRun = true
		} else {
			if cfg.QueueUpload > 0 && m.upRunning >= cfg.QueueUpload {
				canRun = false
			}
			if cfg.QueueAll > 0 && (m.dlRunning+m.upRunning) >= cfg.QueueAll {
				canRun = false
			}
		}

		if canRun {
			m.upRunning++
			m.queueMu.Unlock()
			return nil
		}
		m.queueMu.Unlock()

		select {
		case <-t.Ctx.Done():
			return fmt.Errorf("task cancelled")
		case <-time.After(2 * time.Second):
		}
	}
}

func (m *Manager) ReleaseUploadSlot() {
	m.queueMu.Lock()
	if m.upRunning > 0 {
		m.upRunning--
	}
	m.queueMu.Unlock()
}

// CancelTask cancels an active task by GID, checking permission
func (m *Manager) CancelTask(gid string, requestingUserID int64, isSudo bool) (bool, string) {
	t := m.GetTask(gid)
	if t == nil {
		return false, "Task not found."
	}

	if !isSudo && t.UserID != requestingUserID {
		return false, "You do not have permission to cancel this task."
	}

	t.Cancel("Cancelled by user request")
	m.RemoveTask(t.GID)
	return true, fmt.Sprintf("Task <code>%s</code> (<b>%s</b>) cancelled successfully.", t.GID, t.Name)
}

// ForceStartTask forces a queued task to bypass queue limits and run immediately
func (m *Manager) ForceStartTask(gid string, requestingUserID int64, isSudo bool) (bool, string) {
	t := m.GetTask(gid)
	if t == nil {
		return false, "Task not found."
	}

	if !isSudo && t.UserID != requestingUserID {
		return false, "You do not have permission to force-start this task."
	}

	if t.Options == nil {
		t.Options = &utils.TaskOptions{}
	}
	t.Options.ForceStart = true
	return true, fmt.Sprintf("Task <code>%s</code> (<b>%s</b>) has been promoted to run immediately!", t.GID, t.Name)
}

// CancelAll cancels all tasks matching filter ("all", "dl", "up")
func (m *Manager) CancelAll(filter string, requestingUserID int64, isSudo bool) int {
	all := m.GetAllTasks()
	count := 0
	for _, t := range all {
		if !isSudo && t.UserID != requestingUserID {
			continue
		}

		shouldCancel := false
		switch filter {
		case "dl":
			if t.Status == StatusDownloading || t.Status == StatusQueuedDL {
				shouldCancel = true
			}
		case "up":
			if t.Status == StatusUploading || t.Status == StatusQueuedUP {
				shouldCancel = true
			}
		default:
			shouldCancel = true
		}

		if shouldCancel {
			t.Cancel("Cancelled via /cancelall")
			m.RemoveTask(t.GID)
			count++
		}
	}
	return count
}
