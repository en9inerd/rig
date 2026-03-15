package config

import (
	"fmt"
	"strconv"
	"time"
)

type TLSConfig struct {
	CertFile string
	KeyFile  string
}

func (t TLSConfig) Enabled() bool {
	return t.CertFile != "" && t.KeyFile != ""
}

type Config struct {
	HTTPAddr         string
	TelegramBotToken string
	StorePath        string
	CORSOrigin       string
	TLS              TLSConfig
	Verbose          bool
}

func ParseConfig(getenv func(string) string) *Config {
	return &Config{
		HTTPAddr:         Env(getenv, "RIG_HTTP_ADDR", ":8080"),
		TelegramBotToken: Env(getenv, "RIG_TELEGRAM_BOT_TOKEN", ""),
		StorePath:        Env(getenv, "RIG_STORE_PATH", "/data/rig.json"),
		CORSOrigin:       Env(getenv, "RIG_CORS_ORIGIN", ""),
		TLS: TLSConfig{
			CertFile: Env(getenv, "RIG_TLS_CERT", ""),
			KeyFile:  Env(getenv, "RIG_TLS_KEY", ""),
		},
		Verbose: EnvBool(getenv, "RIG_VERBOSE", false),
	}
}

func Env(getenv func(string) string, key, fallback string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return fallback
}

func EnvBool(getenv func(string) string, key string, fallback bool) bool {
	if v := getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func EnvDuration(getenv func(string) string, key string, fallback time.Duration) time.Duration {
	if v := getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func RequireEnv(getenv func(string) string, key string) (string, error) {
	v := getenv(key)
	if v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}
