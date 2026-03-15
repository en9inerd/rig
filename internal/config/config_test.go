package config

import (
	"testing"
	"time"
)

func TestParseConfig_Defaults(t *testing.T) {
	getenv := func(string) string { return "" }

	cfg := ParseConfig(getenv)

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want \":8080\"", cfg.HTTPAddr)
	}
	if cfg.StorePath != "/data/rig.json" {
		t.Errorf("StorePath = %q, want \"/data/rig.json\"", cfg.StorePath)
	}
	if cfg.TelegramBotToken != "" {
		t.Errorf("TelegramBotToken = %q, want empty", cfg.TelegramBotToken)
	}
	if cfg.CORSOrigin != "" {
		t.Errorf("CORSOrigin = %q, want empty", cfg.CORSOrigin)
	}
	if cfg.TLS.Enabled() {
		t.Error("TLS should be disabled by default")
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
		"RIG_TLS_CERT":           "/certs/cert.pem",
		"RIG_TLS_KEY":            "/certs/key.pem",
		"RIG_VERBOSE":            "true",
	}
	getenv := func(key string) string { return envs[key] }

	cfg := ParseConfig(getenv)

	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.TelegramBotToken != "tok123" {
		t.Errorf("TelegramBotToken = %q", cfg.TelegramBotToken)
	}
	if cfg.StorePath != "/tmp/store.json" {
		t.Errorf("StorePath = %q", cfg.StorePath)
	}
	if cfg.CORSOrigin != "https://example.com" {
		t.Errorf("CORSOrigin = %q", cfg.CORSOrigin)
	}
	if !cfg.TLS.Enabled() {
		t.Error("TLS should be enabled")
	}
	if cfg.TLS.CertFile != "/certs/cert.pem" {
		t.Errorf("TLS.CertFile = %q", cfg.TLS.CertFile)
	}
	if !cfg.Verbose {
		t.Error("Verbose = false, want true")
	}
}

func TestParseConfig_InvalidBoolFallback(t *testing.T) {
	envs := map[string]string{"RIG_VERBOSE": "maybe"}
	getenv := func(key string) string { return envs[key] }

	cfg := ParseConfig(getenv)

	if cfg.Verbose {
		t.Error("invalid bool should fall back to default (false)")
	}
}

func TestEnv(t *testing.T) {
	getenv := func(key string) string {
		if key == "EXISTS" {
			return "value"
		}
		return ""
	}

	if got := Env(getenv, "EXISTS", "default"); got != "value" {
		t.Errorf("Env(EXISTS) = %q, want \"value\"", got)
	}
	if got := Env(getenv, "MISSING", "default"); got != "default" {
		t.Errorf("Env(MISSING) = %q, want \"default\"", got)
	}
}

func TestEnvBool(t *testing.T) {
	envs := map[string]string{
		"TRUE":    "true",
		"FALSE":   "false",
		"INVALID": "maybe",
	}
	getenv := func(key string) string { return envs[key] }

	if got := EnvBool(getenv, "TRUE", false); !got {
		t.Error("EnvBool(TRUE) = false, want true")
	}
	if got := EnvBool(getenv, "FALSE", true); got {
		t.Error("EnvBool(FALSE) = true, want false")
	}
	if got := EnvBool(getenv, "INVALID", true); !got {
		t.Error("EnvBool(INVALID) should fall back to true")
	}
	if got := EnvBool(getenv, "MISSING", true); !got {
		t.Error("EnvBool(MISSING) should fall back to true")
	}
}

func TestEnvDuration(t *testing.T) {
	envs := map[string]string{
		"VALID":   "5m",
		"INVALID": "not-a-duration",
	}
	getenv := func(key string) string { return envs[key] }

	if got := EnvDuration(getenv, "VALID", time.Minute); got != 5*time.Minute {
		t.Errorf("EnvDuration(VALID) = %v, want 5m", got)
	}
	if got := EnvDuration(getenv, "INVALID", 15*time.Minute); got != 15*time.Minute {
		t.Errorf("EnvDuration(INVALID) = %v, want 15m (fallback)", got)
	}
	if got := EnvDuration(getenv, "MISSING", 15*time.Minute); got != 15*time.Minute {
		t.Errorf("EnvDuration(MISSING) = %v, want 15m (fallback)", got)
	}
}

func TestRequireEnv(t *testing.T) {
	getenv := func(key string) string {
		if key == "EXISTS" {
			return "value"
		}
		return ""
	}

	v, err := RequireEnv(getenv, "EXISTS")
	if err != nil || v != "value" {
		t.Errorf("RequireEnv(EXISTS) = (%q, %v)", v, err)
	}

	_, err = RequireEnv(getenv, "MISSING")
	if err == nil {
		t.Error("RequireEnv(MISSING) should return error")
	}
}
