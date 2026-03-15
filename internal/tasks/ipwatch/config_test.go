package ipwatch

import (
	"strings"
	"testing"
	"time"
)

func TestLoadConfig_Valid(t *testing.T) {
	envs := map[string]string{
		"RIG_IP_CHAT_ID":  "chat1",
		"RIG_IP_INTERVAL": "30m",
	}
	getenv := func(key string) string { return envs[key] }

	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.ChatID != "chat1" {
		t.Errorf("ChatID = %q", cfg.ChatID)
	}
	if cfg.Interval != 30*time.Minute {
		t.Errorf("Interval = %v, want 30m", cfg.Interval)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	envs := map[string]string{"RIG_IP_CHAT_ID": "chat1"}
	getenv := func(key string) string { return envs[key] }

	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Interval != 15*time.Minute {
		t.Errorf("Interval = %v, want 15m (default)", cfg.Interval)
	}
}

func TestLoadConfig_Disabled(t *testing.T) {
	envs := map[string]string{"RIG_IP_ENABLED": "false"}
	getenv := func(key string) string { return envs[key] }

	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Error("expected nil config when disabled")
	}
}

func TestLoadConfig_MissingChatID(t *testing.T) {
	getenv := func(string) string { return "" }

	_, err := LoadConfig(getenv)
	if err == nil {
		t.Fatal("expected error for missing chat ID")
	}
	if !strings.Contains(err.Error(), "RIG_IP_CHAT_ID") {
		t.Errorf("error should mention RIG_IP_CHAT_ID, got: %v", err)
	}
}
