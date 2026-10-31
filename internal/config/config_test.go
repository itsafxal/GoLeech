package config

import (
	"os"
	"testing"
)

func TestConfigDefaultsAndValidation(t *testing.T) {
	cfg := &Config{}
	missing := cfg.Validate()
	if len(missing) != 4 {
		t.Errorf("expected 4 missing required variables, got: %d (%v)", len(missing), missing)
	}

	cfg.BotToken = "123:ABC"
	cfg.TelegramAPI = 12345
	cfg.TelegramHash = "hash123"
	cfg.OwnerID = 99999
	if len(cfg.Validate()) != 0 {
		t.Errorf("expected 0 missing variables once required fields are set")
	}
}

func TestConfigAliasesAndParsing(t *testing.T) {
	os.Setenv("API_ID", "778899")
	os.Setenv("API_HASH", "mock_hash_xyz")
	os.Setenv("SESSION_STRING", "mock_session_string")
	os.Setenv("SUDO_USERS", "111, 222, 333")
	os.Setenv("MAX_CONCURRENT_DOWNLOADS", "8")
	defer func() {
		os.Unsetenv("API_ID")
		os.Unsetenv("API_HASH")
		os.Unsetenv("SESSION_STRING")
		os.Unsetenv("SUDO_USERS")
		os.Unsetenv("MAX_CONCURRENT_DOWNLOADS")
	}()

	cfg := LoadConfig()
	if cfg.TelegramAPI != 778899 {
		t.Errorf("expected API_ID alias 778899, got: %d", cfg.TelegramAPI)
	}
	if cfg.TelegramHash != "mock_hash_xyz" {
		t.Errorf("expected API_HASH alias 'mock_hash_xyz', got: %s", cfg.TelegramHash)
	}
	if cfg.UserSessionString != "mock_session_string" {
		t.Errorf("expected SESSION_STRING alias, got: %s", cfg.UserSessionString)
	}
	if len(cfg.SudoUsers) != 3 || cfg.SudoUsers[0] != 111 || cfg.SudoUsers[2] != 333 {
		t.Errorf("expected comma-separated SUDO_USERS [111 222 333], got: %v", cfg.SudoUsers)
	}
	if cfg.QueueDownload != 8 {
		t.Errorf("expected MAX_CONCURRENT_DOWNLOADS 8, got: %d", cfg.QueueDownload)
	}
}

func TestParseIntList(t *testing.T) {
	res := parseIntList("100, 200,300 400")
	if len(res) != 4 || res[0] != 100 || res[3] != 400 {
		t.Errorf("unexpected parseIntList output: %v", res)
	}
}
