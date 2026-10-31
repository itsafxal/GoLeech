package task

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/GoLeech/GoLeech/pkg/utils"
)

type TaskStatus string

const (
	StatusQueuedDL    TaskStatus = "QueueDl"
	StatusQueuedUP    TaskStatus = "QueueUp"
	StatusDownloading TaskStatus = "Download"
	StatusUploading   TaskStatus = "Upload"
	StatusProcessing  TaskStatus = "Process"
	StatusArchiving   TaskStatus = "Archive"
	StatusExtracting  TaskStatus = "Extract"
	StatusSplitting   TaskStatus = "Split"
	StatusPaused      TaskStatus = "Pause"
	StatusCompleted   TaskStatus = "Complete"
	StatusFailed      TaskStatus = "Failed"
	StatusCancelled   TaskStatus = "Cancelled"
)

type TaskType string

const (
	TypeMirror TaskType = "Mirror"
	TypeLeech  TaskType = "Leech"
	TypeClone  TaskType = "Clone"
)

type Task struct {
	GID            string             `json:"gid"`
	UserID         int64              `json:"user_id"`
	UserName       string             `json:"user_name"`
	ChatID         int64              `json:"chat_id"`
	MessageID      int32              `json:"message_id"`
	Name           string             `json:"name"`
	Type           TaskType           `json:"type"`
	Status         TaskStatus         `json:"status"`
	Engine         string             `json:"engine"`
	TotalSize      int64              `json:"total_size"`
	ProcessedBytes int64              `json:"processed_bytes"`
	Speed          float64            `json:"speed"`
	ETA            int64              `json:"eta"`
	StartTime      time.Time          `json:"start_time"`
	Options        *utils.TaskOptions `json:"options"`
	LocalPath      string             `json:"local_path"`
	ErrorMsg       string             `json:"error_msg"`

	Ctx        context.Context    `json:"-"`
	CancelFunc context.CancelFunc `json:"-"`
	mu         sync.RWMutex
}

func GenerateGID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func NewTask(userID int64, userName string, chatID int64, msgID int32, taskType TaskType, opts *utils.TaskOptions) *Task {
	ctx, cancel := context.WithCancel(context.Background())
	name := opts.Name
	if name == "" && opts.Link != "" {
		name = utils.CleanFilename(opts.Link)
	}
	if name == "" {
		name = "Task-" + GenerateGID()
	}

	return &Task{
		GID:        GenerateGID(),
		UserID:     userID,
		UserName:   userName,
		ChatID:     chatID,
		MessageID:  msgID,
		Name:       name,
		Type:       taskType,
		Status:     StatusQueuedDL,
		Engine:     "HTTP",
		StartTime:  time.Now(),
		Options:    opts,
		Ctx:        ctx,
		CancelFunc: cancel,
	}
}

func (t *Task) UpdateProgress(processed, total int64, speed float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ProcessedBytes = processed
	if total > 0 {
		t.TotalSize = total
	}
	t.Speed = speed

	if speed > 0 && t.TotalSize > t.ProcessedBytes {
		remaining := t.TotalSize - t.ProcessedBytes
		t.ETA = int64(float64(remaining) / speed)
	} else {
		t.ETA = 0
	}
}

func (t *Task) Progress() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.TotalSize <= 0 {
		return 0
	}
	pct := (float64(t.ProcessedBytes) / float64(t.TotalSize)) * 100.0
	if pct > 100 {
		return 100
	}
	return pct
}

func (t *Task) SetStatus(st TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = st
}

func (t *Task) SetEngine(eng string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Engine = eng
}

func (t *Task) SetName(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Name = utils.CleanFilename(name)
}

func (t *Task) SetError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err != nil {
		t.ErrorMsg = err.Error()
		t.Status = StatusFailed
	}
}

func (t *Task) Cancel(reason string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = StatusCancelled
	t.ErrorMsg = reason
	if t.CancelFunc != nil {
		t.CancelFunc()
	}
}

func (t *Task) IsCancelled() bool {
	select {
	case <-t.Ctx.Done():
		return true
	default:
		t.mu.RLock()
		defer t.mu.RUnlock()
		return t.Status == StatusCancelled
	}
}
