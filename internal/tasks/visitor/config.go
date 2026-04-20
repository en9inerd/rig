package visitor

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/en9inerd/rig/internal/config"
)

type Site struct {
	Name      string `json:"name"`
	AuthToken string `json:"authToken"`
	ChatID    string `json:"chatId"`
	Tag       string `json:"tag"`
}

type Config struct {
	GeoIPDB     string
	Sites       []Site
	Dedup       bool
	DedupWindow time.Duration
}

func LoadConfig(getenv func(string) string) (*Config, error) {
	if !config.EnvBool(getenv, "RIG_VISITOR_ENABLED", true) {
		return nil, nil
	}

	sitesFile := config.Env(getenv, "RIG_VISITOR_SITES_FILE", "")
	if sitesFile == "" {
		return nil, fmt.Errorf("RIG_VISITOR_SITES_FILE is required")
	}

	sites, err := loadSites(sitesFile)
	if err != nil {
		return nil, fmt.Errorf("visitor sites: %w", err)
	}

	dedupWindow := config.EnvDuration(getenv, "RIG_VISITOR_DEDUP_WINDOW", 10*time.Minute)
	if dedupWindow <= 0 {
		return nil, fmt.Errorf("RIG_VISITOR_DEDUP_WINDOW must be positive")
	}

	return &Config{
		GeoIPDB:     config.Env(getenv, "RIG_VISITOR_GEOIP_DB", "/data/geoip/GeoLite2-City.mmdb"),
		Sites:       sites,
		Dedup:       config.EnvBool(getenv, "RIG_VISITOR_DEDUP", false),
		DedupWindow: dedupWindow,
	}, nil
}

func loadSites(path string) ([]Site, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sites []Site
	if err := json.Unmarshal(data, &sites); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if len(sites) == 0 {
		return nil, fmt.Errorf("no sites defined")
	}

	seenNames := make(map[string]bool, len(sites))
	seenTokens := make(map[string]bool, len(sites))
	for i, s := range sites {
		if s.Name == "" {
			return nil, fmt.Errorf("site[%d]: name is required", i)
		}
		if s.AuthToken == "" {
			return nil, fmt.Errorf("site %q: authToken is required", s.Name)
		}
		if s.ChatID == "" {
			return nil, fmt.Errorf("site %q: chatId is required", s.Name)
		}
		if seenNames[s.Name] {
			return nil, fmt.Errorf("site %q: duplicate name", s.Name)
		}
		if seenTokens[s.AuthToken] {
			return nil, fmt.Errorf("site %q: duplicate authToken", s.Name)
		}
		seenNames[s.Name] = true
		seenTokens[s.AuthToken] = true
		if s.Tag == "" {
			sites[i].Tag = s.Name
		}
	}
	return sites, nil
}
