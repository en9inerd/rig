package feedwatch

import (
	"strings"
	"testing"
	"time"
)

func TestLoadConfig_Valid(t *testing.T) {
	envs := map[string]string{
		"RIG_FEED_URL":      "https://example.com/feed.xml",
		"RIG_FEED_CHAT_ID":  "chat1",
		"RIG_FEED_INTERVAL": "5m",
	}
	getenv := func(key string) string { return envs[key] }

	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.URL != "https://example.com/feed.xml" {
		t.Errorf("URL = %q", cfg.URL)
	}
	if cfg.ChatID != "chat1" {
		t.Errorf("ChatID = %q", cfg.ChatID)
	}
	if cfg.Interval != 5*time.Minute {
		t.Errorf("Interval = %v, want 5m", cfg.Interval)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	envs := map[string]string{
		"RIG_FEED_URL":     "https://example.com/feed.xml",
		"RIG_FEED_CHAT_ID": "chat1",
	}
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
	envs := map[string]string{"RIG_FEED_ENABLED": "false"}
	getenv := func(key string) string { return envs[key] }

	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Error("expected nil config when disabled")
	}
}

func TestLoadConfig_MissingURL(t *testing.T) {
	envs := map[string]string{"RIG_FEED_CHAT_ID": "chat1"}
	getenv := func(key string) string { return envs[key] }

	_, err := LoadConfig(getenv)
	if err == nil {
		t.Fatal("expected error for missing URL")
	}
	if !strings.Contains(err.Error(), "RIG_FEED_URL") {
		t.Errorf("error should mention RIG_FEED_URL, got: %v", err)
	}
}

func TestLoadConfig_MissingChatID(t *testing.T) {
	envs := map[string]string{"RIG_FEED_URL": "https://example.com/feed.xml"}
	getenv := func(key string) string { return envs[key] }

	_, err := LoadConfig(getenv)
	if err == nil {
		t.Fatal("expected error for missing chat ID")
	}
	if !strings.Contains(err.Error(), "RIG_FEED_CHAT_ID") {
		t.Errorf("error should mention RIG_FEED_CHAT_ID, got: %v", err)
	}
}
