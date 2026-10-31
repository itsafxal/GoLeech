package plugin

import (
	"context"
	"testing"
)

func TestPluginRegistrationNoDeadlock(t *testing.T) {
	reg := InitRegistry(nil)
	ctx := context.Background()

	err := reg.RegisterPlugin(ctx, &MediaInfoPlugin{})
	if err != nil {
		t.Fatalf("RegisterPlugin failed: %v", err)
	}

	if len(reg.plugins) != 1 {
		t.Errorf("Expected 1 plugin registered, got %d", len(reg.plugins))
	}
	if len(reg.commands) != 1 {
		t.Errorf("Expected 1 command registered by plugin, got %d", len(reg.commands))
	}
}
