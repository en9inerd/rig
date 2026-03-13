package config

import (
	"strings"
	"testing"
	"time"
)

func validEnv() map[string]string {
	return map[string]string{
		"RIG_TELEGRAM_BOT_TOKEN": "test-token",
		"RIG_VISITOR_AUTH_TOKEN": "test-auth",
		"RIG_VISITOR_CHAT_ID":    "chat1",
		"RIG_FEED_URL":           "https://example.com/feed.xml",
		"RIG_FEED_CHAT_ID":       "chat2",
		"RIG_IP_CHAT_ID":         "chat3",
	}
}

func disabledEnv() map[string]string {
	return map[string]string{
		"RIG_VISITOR_ENABLED": "false",
		"RIG_FEED_ENABLED":    "false",
		"RIG_IP_ENABLED":      "false",
	}
}

func TestParseConfig_Defaults(t *testing.T) {
	envs := validEnv()
	getenv := func(key string) string { return envs[key] }

	cfg, err := ParseConfig([]string{"rig"}, getenv)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want \":8080\"", cfg.HTTPAddr)
	}
	if cfg.StorePath != "/data/rig.json" {
		t.Errorf("StorePath = %q, want \"/data/rig.json\"", cfg.StorePath)
	}
	if !cfg.Visitor.Enabled {
		t.Error("Visitor.Enabled = false, want true")
	}
	if cfg.Visitor.Tag != "EngiNerd" {
		t.Errorf("Visitor.Tag = %q, want \"EngiNerd\"", cfg.Visitor.Tag)
	}
	if !cfg.Feed.Enabled {
		t.Error("Feed.Enabled = false, want true")
	}
	if cfg.Feed.Interval != 15*time.Minute {
		t.Errorf("Feed.Interval = %v, want 15m", cfg.Feed.Interval)
	}
	if !cfg.IP.Enabled {
		t.Error("IP.Enabled = false, want true")
	}
	if cfg.IP.Interval != 15*time.Minute {
		t.Errorf("IP.Interval = %v, want 15m", cfg.IP.Interval)
	}
	if cfg.Verbose {
		t.Error("Verbose = true, want false")
	}
}

func TestParseConfig_EnvOverrides(t *testing.T) {
	envs := map[string]string{
		"RIG_HTTP_ADDR":          ":9090",
		"RIG_TELEGRAM_BOT_TOKEN": "tok123",
		"RIG_STORE_PATH":         "/tmp/store.json",
		"RIG_CORS_ORIGIN":        "https://example.com",
		"RIG_VISITOR_ENABLED":    "false",
		"RIG_VISITOR_AUTH_TOKEN": "secret",
		"RIG_VISITOR_CHAT_ID":    "chat1",
		"RIG_VISITOR_TAG":        "Custom",
		"RIG_VISITOR_GEOIP_DB":   "/custom/geo.mmdb",
		"RIG_FEED_ENABLED":       "false",
		"RIG_FEED_URL":           "https://example.com/feed",
		"RIG_FEED_INTERVAL":      "5m",
		"RIG_FEED_CHAT_ID":       "chat2",
		"RIG_IP_ENABLED":         "false",
		"RIG_IP_INTERVAL":        "30m",
		"RIG_IP_CHAT_ID":         "chat3",
	}
	getenv := func(key string) string { return envs[key] }

	cfg, err := ParseConfig([]string{"rig"}, getenv)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.TelegramBotToken != "tok123" {
		t.Errorf("Token = %q", cfg.TelegramBotToken)
	}
	if cfg.StorePath != "/tmp/store.json" {
		t.Errorf("StorePath = %q", cfg.StorePath)
	}
	if cfg.CORSOrigin != "https://example.com" {
		t.Errorf("CORSOrigin = %q, want %q", cfg.CORSOrigin, "https://example.com")
	}
	if cfg.Visitor.Enabled {
		t.Error("Visitor.Enabled = true")
	}
	if cfg.Visitor.AuthToken != "secret" {
		t.Errorf("Visitor.AuthToken = %q", cfg.Visitor.AuthToken)
	}
	if cfg.Visitor.ChatID != "chat1" {
		t.Errorf("Visitor.ChatID = %q", cfg.Visitor.ChatID)
	}
	if cfg.Visitor.Tag != "Custom" {
		t.Errorf("Visitor.Tag = %q", cfg.Visitor.Tag)
	}
	if cfg.Visitor.GeoIPDB != "/custom/geo.mmdb" {
		t.Errorf("Visitor.GeoIPDB = %q", cfg.Visitor.GeoIPDB)
	}
	if cfg.Feed.Enabled {
		t.Error("Feed.Enabled = true")
	}
	if cfg.Feed.URL != "https://example.com/feed" {
		t.Errorf("Feed.URL = %q", cfg.Feed.URL)
	}
	if cfg.Feed.Interval != 5*time.Minute {
		t.Errorf("Feed.Interval = %v", cfg.Feed.Interval)
	}
	if cfg.Feed.ChatID != "chat2" {
		t.Errorf("Feed.ChatID = %q", cfg.Feed.ChatID)
	}
	if cfg.IP.Enabled {
		t.Error("IP.Enabled = true")
	}
	if cfg.IP.Interval != 30*time.Minute {
		t.Errorf("IP.Interval = %v", cfg.IP.Interval)
	}
	if cfg.IP.ChatID != "chat3" {
		t.Errorf("IP.ChatID = %q", cfg.IP.ChatID)
	}
}

func TestParseConfig_VerboseFlag(t *testing.T) {
	envs := disabledEnv()
	getenv := func(key string) string { return envs[key] }

	cfg, err := ParseConfig([]string{"rig", "--verbose"}, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Verbose {
		t.Error("Verbose = false, want true")
	}
}

func TestParseConfig_VerboseShortFlag(t *testing.T) {
	envs := disabledEnv()
	getenv := func(key string) string { return envs[key] }

	cfg, err := ParseConfig([]string{"rig", "-v"}, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Verbose {
		t.Error("Verbose = false, want true")
	}
}

func TestParseConfig_UnknownFlag(t *testing.T) {
	getenv := func(string) string { return "" }

	_, err := ParseConfig([]string{"rig", "--unknown"}, getenv)
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestParseConfig_InvalidBoolFallback(t *testing.T) {
	envs := validEnv()
	envs["RIG_VISITOR_ENABLED"] = "maybe"
	getenv := func(key string) string { return envs[key] }

	cfg, err := ParseConfig([]string{"rig"}, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Visitor.Enabled {
		t.Error("invalid bool should fall back to default (true)")
	}
}

func TestParseConfig_InvalidDurationFallback(t *testing.T) {
	envs := validEnv()
	envs["RIG_FEED_INTERVAL"] = "not-a-duration"
	getenv := func(key string) string { return envs[key] }

	cfg, err := ParseConfig([]string{"rig"}, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Feed.Interval != 15*time.Minute {
		t.Errorf("invalid duration should fall back to default, got %v", cfg.Feed.Interval)
	}
}

func TestParseConfig_ValidationMissingBotToken(t *testing.T) {
	getenv := func(string) string { return "" }

	_, err := ParseConfig([]string{"rig"}, getenv)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "RIG_TELEGRAM_BOT_TOKEN") {
		t.Errorf("error should mention RIG_TELEGRAM_BOT_TOKEN, got: %v", err)
	}
}

func TestParseConfig_ValidationMissingVisitorFields(t *testing.T) {
	envs := map[string]string{
		"RIG_TELEGRAM_BOT_TOKEN": "tok",
		"RIG_FEED_ENABLED":       "false",
		"RIG_IP_ENABLED":         "false",
	}
	getenv := func(key string) string { return envs[key] }

	_, err := ParseConfig([]string{"rig"}, getenv)
	if err == nil {
		t.Fatal("expected validation error for missing visitor auth token")
	}
	if !strings.Contains(err.Error(), "RIG_VISITOR_AUTH_TOKEN") {
		t.Errorf("error should mention RIG_VISITOR_AUTH_TOKEN, got: %v", err)
	}
}

func TestParseConfig_ValidationMissingFeedURL(t *testing.T) {
	envs := map[string]string{
		"RIG_TELEGRAM_BOT_TOKEN": "tok",
		"RIG_VISITOR_ENABLED":    "false",
		"RIG_IP_ENABLED":         "false",
	}
	getenv := func(key string) string { return envs[key] }

	_, err := ParseConfig([]string{"rig"}, getenv)
	if err == nil {
		t.Fatal("expected validation error for missing feed URL")
	}
	if !strings.Contains(err.Error(), "RIG_FEED_URL") {
		t.Errorf("error should mention RIG_FEED_URL, got: %v", err)
	}
}

func TestParseConfig_ValidationMissingIPChatID(t *testing.T) {
	envs := map[string]string{
		"RIG_TELEGRAM_BOT_TOKEN": "tok",
		"RIG_VISITOR_ENABLED":    "false",
		"RIG_FEED_ENABLED":       "false",
	}
	getenv := func(key string) string { return envs[key] }

	_, err := ParseConfig([]string{"rig"}, getenv)
	if err == nil {
		t.Fatal("expected validation error for missing IP chat ID")
	}
	if !strings.Contains(err.Error(), "RIG_IP_CHAT_ID") {
		t.Errorf("error should mention RIG_IP_CHAT_ID, got: %v", err)
	}
}

func TestParseConfig_AllDisabledNoCredentials(t *testing.T) {
	envs := disabledEnv()
	getenv := func(key string) string { return envs[key] }

	_, err := ParseConfig([]string{"rig"}, getenv)
	if err != nil {
		t.Fatalf("all tasks disabled should not require credentials: %v", err)
	}
}
