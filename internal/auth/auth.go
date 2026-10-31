package auth

import (
	"github.com/GoLeech/GoLeech/internal/config"
	"github.com/GoLeech/GoLeech/internal/db"
)

type Role string

const (
	RoleOwner       Role = "owner"
	RoleSudo        Role = "sudo"
	RoleAuthorized  Role = "authorized"
	RoleBlacklisted Role = "blacklisted"
	RoleGuest       Role = "guest"
)

// IsBlacklisted checks if user or chat is blacklisted
func IsBlacklisted(id int64) bool {
	if db.GlobalStore != nil && db.GlobalStore.IsBlacklisted(id) {
		return true
	}
	return false
}

// IsOwner checks if user ID matches configured owner
func IsOwner(userID int64) bool {
	cfg := config.Get()
	return cfg.OwnerID != 0 && userID == cfg.OwnerID
}

// IsSudo checks if user is owner or listed as sudo in config or db
func IsSudo(userID int64) bool {
	if IsBlacklisted(userID) {
		return false
	}
	if IsOwner(userID) {
		return true
	}
	cfg := config.Get()
	for _, id := range cfg.SudoUsers {
		if id == userID {
			return true
		}
	}
	if db.GlobalStore != nil && db.GlobalStore.IsSudo(userID) {
		return true
	}
	return false
}

// IsAuthorized checks if user or chat has permission to execute tasks
func IsAuthorized(userID, chatID int64) bool {
	if IsBlacklisted(userID) || IsBlacklisted(chatID) {
		return false
	}
	if IsSudo(userID) {
		return true
	}
	cfg := config.Get()
	for _, id := range cfg.AuthorizedChats {
		if id == chatID || id == userID {
			return true
		}
	}
	if db.GlobalStore != nil && db.GlobalStore.IsAuthorized(userID, chatID) {
		return true
	}
	return false
}

// CanExecute checks if user or chat is allowed to execute tasks
func CanExecute(userID, chatID int64) bool {
	return CanAccessBot(userID)
}

// CanAccessBot checks if a user is an authorized Admin or Sudo user (strict fail-closed authorization)
func CanAccessBot(userID int64) bool {
	if userID == 0 || IsBlacklisted(userID) {
		return false
	}
	return IsSudo(userID)
}

// GetUserRole returns the highest role of a user
func GetUserRole(userID, chatID int64) Role {
	if IsBlacklisted(userID) || IsBlacklisted(chatID) {
		return RoleBlacklisted
	}
	if IsOwner(userID) {
		return RoleOwner
	}
	if IsSudo(userID) {
		return RoleSudo
	}
	if IsAuthorized(userID, chatID) {
		return RoleAuthorized
	}
	return RoleGuest
}
