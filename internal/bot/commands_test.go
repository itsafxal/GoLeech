package bot

import (
	"testing"
)

func TestRegisterBotCommandsNilClient(t *testing.T) {
	err := RegisterBotCommands(nil)
	if err != nil {
		t.Errorf("expected nil error for nil client, got: %v", err)
	}
}
