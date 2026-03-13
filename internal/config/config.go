package config

import (
	"flag"
	"fmt"
	"strconv"
	"time"
)

type VisitorConfig struct {
	Enabled   bool
	AuthToken string
	ChatID    string
	Tag       string
	GeoIPDB   string
}

type FeedConfig struct {
	Enabled  bool
	URL      string
	Interval time.Duration
	ChatID   string
}

type IPConfig struct {
	Enabled  bool
	Interval time.Duration
	ChatID   string
}

type Config struct {
	HTTPAddr         string
	TelegramBotToken string
	StorePath        string
	CORSOrigin       string

	Visitor VisitorConfig
	Feed    FeedConfig
	IP      IPConfig

	Verbose bool
}

func ParseConfig(args []string, getenv func(string) string) (*Config, error) {
	env := func(key, fallback string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return fallback
	}

	envBool := func(key string, fallback bool) bool {
		if v := getenv(key); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
		}
		return fallback
	}

	envDuration := func(key string, fallback time.Duration) time.Duration {
		if v := getenv(key); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				return d
			}
		}
		return fallback
	}

	fs := flag.NewFlagSet("rig", flag.ContinueOnError)

	verbose := fs.Bool("verbose", false, "Enable verbose logging")
	fs.BoolVar(verbose, "v", false, "Enable verbose logging (shorthand)")

	if err := fs.Parse(args[1:]); err != nil {
		return nil, err
	}

	cfg := &Config{
		HTTPAddr:         env("RIG_HTTP_ADDR", ":8080"),
		TelegramBotToken: env("RIG_TELEGRAM_BOT_TOKEN", ""),
		StorePath:        env("RIG_STORE_PATH", "/data/rig.json"),
		CORSOrigin:       env("RIG_CORS_ORIGIN", ""),

		Visitor: VisitorConfig{
			Enabled:   envBool("RIG_VISITOR_ENABLED", true),
			AuthToken: env("RIG_VISITOR_AUTH_TOKEN", ""),
			ChatID:    env("RIG_VISITOR_CHAT_ID", ""),
			Tag:       env("RIG_VISITOR_TAG", "EngiNerd"),
			GeoIPDB:   env("RIG_VISITOR_GEOIP_DB", "/data/geoip/GeoLite2-City.mmdb"),
		},
		Feed: FeedConfig{
			Enabled:  envBool("RIG_FEED_ENABLED", true),
			URL:      env("RIG_FEED_URL", ""),
			Interval: envDuration("RIG_FEED_INTERVAL", 15*time.Minute),
			ChatID:   env("RIG_FEED_CHAT_ID", ""),
		},
		IP: IPConfig{
			Enabled:  envBool("RIG_IP_ENABLED", true),
			Interval: envDuration("RIG_IP_INTERVAL", 15*time.Minute),
			ChatID:   env("RIG_IP_CHAT_ID", ""),
		},

		Verbose: *verbose,
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	anyTask := c.Visitor.Enabled || c.Feed.Enabled || c.IP.Enabled
	if anyTask && c.TelegramBotToken == "" {
		return fmt.Errorf("RIG_TELEGRAM_BOT_TOKEN is required when tasks are enabled")
	}

	if c.Visitor.Enabled {
		if c.Visitor.AuthToken == "" {
			return fmt.Errorf("RIG_VISITOR_AUTH_TOKEN is required when visitor task is enabled")
		}
		if c.Visitor.ChatID == "" {
			return fmt.Errorf("RIG_VISITOR_CHAT_ID is required when visitor task is enabled")
		}
	}

	if c.Feed.Enabled {
		if c.Feed.URL == "" {
			return fmt.Errorf("RIG_FEED_URL is required when feed task is enabled")
		}
		if c.Feed.ChatID == "" {
			return fmt.Errorf("RIG_FEED_CHAT_ID is required when feed task is enabled")
		}
	}

	if c.IP.Enabled {
		if c.IP.ChatID == "" {
			return fmt.Errorf("RIG_IP_CHAT_ID is required when IP task is enabled")
		}
	}

	return nil
}
