package auth

import (
	"testing"

	"github.com/GoLeech/GoLeech/internal/config"
)

func TestAuthRoles(t *testing.T) {
	cfg := config.Get()
	cfg.OwnerID = 1001
	cfg.SudoUsers = []int64{2002, 3003}
	cfg.AuthorizedChats = []int64{-1004004}

	if !IsOwner(1001) {
		t.Errorf("expected 1001 to be owner")
	}
	if IsOwner(2002) {
		t.Errorf("did not expect 2002 to be owner")
	}

	if !IsSudo(1001) || !IsSudo(2002) || !IsSudo(3003) {
		t.Errorf("expected 1001, 2002, 3003 to be sudo")
	}
	if IsSudo(9999) {
		t.Errorf("did not expect 9999 to be sudo")
	}

	if !IsAuthorized(9999, -1004004) {
		t.Errorf("expected chat -1004004 to be authorized")
	}
	if IsAuthorized(9999, -1005555) {
		t.Errorf("did not expect chat -1005555 to be authorized")
	}

	if GetUserRole(1001, 0) != RoleOwner {
		t.Errorf("expected RoleOwner for 1001")
	}
	if GetUserRole(2002, 0) != RoleSudo {
		t.Errorf("expected RoleSudo for 2002")
	}
	if GetUserRole(9999, -1004004) != RoleAuthorized {
		t.Errorf("expected RoleAuthorized for 9999 in chat -1004004")
	}
	if GetUserRole(9999, 0) != RoleGuest {
		t.Errorf("expected RoleGuest for 9999")
	}

	if !CanAccessBot(1001) || !CanAccessBot(2002) || !CanAccessBot(3003) {
		t.Errorf("expected 1001, 2002, 3003 to have bot access")
	}
	if CanAccessBot(9999) {
		t.Errorf("did not expect unauthorized user 9999 to have bot access")
	}
}
