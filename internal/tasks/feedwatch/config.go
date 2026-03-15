package feedwatch

import (
	"time"

	"github.com/en9inerd/rig/internal/config"
)

type Config struct {
	URL      string
	Interval time.Duration
	ChatID   string
}

func LoadConfig(getenv func(string) string) (*Config, error) {
	if !config.EnvBool(getenv, "RIG_FEED_ENABLED", true) {
		return nil, nil
	}

	url, err := config.RequireEnv(getenv, "RIG_FEED_URL")
	if err != nil {
		return nil, err
	}
	chatID, err := config.RequireEnv(getenv, "RIG_FEED_CHAT_ID")
	if err != nil {
		return nil, err
	}

	return &Config{
		URL:      url,
		Interval: config.EnvDuration(getenv, "RIG_FEED_INTERVAL", 15*time.Minute),
		ChatID:   chatID,
	}, nil
}
