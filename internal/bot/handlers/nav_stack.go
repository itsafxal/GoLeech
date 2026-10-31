package handlers

import (
	"sync"
)

type navStackManager struct {
	mu     sync.RWMutex
	stacks map[int64][]string
}

// NavManager tracks context-aware navigation per user across settings panels
var NavManager = &navStackManager{
	stacks: make(map[int64][]string),
}

// Push records the current page onto the user's navigation history stack
func (n *navStackManager) Push(userID int64, page string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if page == "" || page == "close" {
		return
	}
	stack := n.stacks[userID]
	if len(stack) > 0 && stack[len(stack)-1] == page {
		return
	}
	n.stacks[userID] = append(stack, page)
}

// Pop returns the previous screen from the user's navigation history stack
func (n *navStackManager) Pop(userID int64) string {
	n.mu.Lock()
	defer n.mu.Unlock()
	stack := n.stacks[userID]
	if len(stack) == 0 {
		return ""
	}
	prev := stack[len(stack)-1]
	n.stacks[userID] = stack[:len(stack)-1]
	return prev
}

// Clear resets the user's navigation history stack
func (n *navStackManager) Clear(userID int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.stacks, userID)
}

// GetParent returns the natural hierarchical parent category for any settings sub-page
func (n *navStackManager) GetParent(page string) string {
	switch page {
	case "thumb":
		return "leech"
	case "rclone", "gdrive", "mega":
		return "mirror"
	case "gofile", "buzzheavier", "pixeldrain", "devuploads", "vikingfile":
		return "uphoster"
	case "ffset":
		return "main"
	case "leech", "mirror", "general", "uphoster", "clone", "ytdlp", "advanced", "seedr":
		return "main"
	default:
		return "main"
	}
}

