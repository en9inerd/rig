package ipwatch

import (
	"time"

	"github.com/en9inerd/rig/internal/config"
)

type Config struct {
	Interval time.Duration
	ChatID   string
}

func LoadConfig(getenv func(string) string) (*Config, error) {
	if !config.EnvBool(getenv, "RIG_IP_ENABLED", true) {
		return nil, nil
	}

	chatID, err := config.RequireEnv(getenv, "RIG_IP_CHAT_ID")
	if err != nil {
		return nil, err
	}

	return &Config{
		Interval: config.EnvDuration(getenv, "RIG_IP_INTERVAL", 15*time.Minute),
		ChatID:   chatID,
	}, nil
}
