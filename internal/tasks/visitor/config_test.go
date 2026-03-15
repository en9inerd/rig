package visitor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSitesFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "visitors.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_Valid(t *testing.T) {
	sitesFile := writeSitesFile(t, `[{"name":"blog","authToken":"abc","chatId":"123","tag":"Blog"}]`)
	envs := map[string]string{
		"RIG_VISITOR_SITES_FILE": sitesFile,
		"RIG_VISITOR_GEOIP_DB":  "/custom/geo.mmdb",
	}
	getenv := func(key string) string { return envs[key] }

	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.GeoIPDB != "/custom/geo.mmdb" {
		t.Errorf("GeoIPDB = %q", cfg.GeoIPDB)
	}
	if len(cfg.Sites) != 1 {
		t.Fatalf("Sites len = %d, want 1", len(cfg.Sites))
	}
	if cfg.Sites[0].Tag != "Blog" {
		t.Errorf("Sites[0].Tag = %q, want \"Blog\"", cfg.Sites[0].Tag)
	}
}

func TestLoadConfig_Disabled(t *testing.T) {
	envs := map[string]string{"RIG_VISITOR_ENABLED": "false"}
	getenv := func(key string) string { return envs[key] }

	cfg, err := LoadConfig(getenv)
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Error("expected nil config when disabled")
	}
}

func TestLoadConfig_MissingSitesFile(t *testing.T) {
	getenv := func(string) string { return "" }

	_, err := LoadConfig(getenv)
	if err == nil {
		t.Fatal("expected error for missing sites file env")
	}
	if !strings.Contains(err.Error(), "RIG_VISITOR_SITES_FILE") {
		t.Errorf("error should mention RIG_VISITOR_SITES_FILE, got: %v", err)
	}
}

func TestLoadSites_Valid(t *testing.T) {
	path := writeSitesFile(t, `[
		{"name":"blog","authToken":"abc","chatId":"123","tag":"Blog"},
		{"name":"portfolio","authToken":"def","chatId":"456"}
	]`)

	sites, err := loadSites(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(sites) != 2 {
		t.Fatalf("got %d sites, want 2", len(sites))
	}
	if sites[0].Tag != "Blog" {
		t.Errorf("sites[0].Tag = %q, want \"Blog\"", sites[0].Tag)
	}
	if sites[1].Tag != "portfolio" {
		t.Errorf("sites[1].Tag = %q, want \"portfolio\" (default from name)", sites[1].Tag)
	}
}

func TestLoadSites_MissingFile(t *testing.T) {
	_, err := loadSites("/nonexistent/visitors.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadSites_InvalidJSON(t *testing.T) {
	path := writeSitesFile(t, `not json`)
	_, err := loadSites(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadSites_Empty(t *testing.T) {
	path := writeSitesFile(t, `[]`)
	_, err := loadSites(path)
	if err == nil {
		t.Fatal("expected error for empty sites")
	}
}

func TestLoadSites_MissingName(t *testing.T) {
	path := writeSitesFile(t, `[{"authToken":"abc","chatId":"123"}]`)
	_, err := loadSites(path)
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("expected name required error, got: %v", err)
	}
}

func TestLoadSites_MissingToken(t *testing.T) {
	path := writeSitesFile(t, `[{"name":"blog","chatId":"123"}]`)
	_, err := loadSites(path)
	if err == nil || !strings.Contains(err.Error(), "authToken is required") {
		t.Fatalf("expected authToken required error, got: %v", err)
	}
}

func TestLoadSites_MissingChatID(t *testing.T) {
	path := writeSitesFile(t, `[{"name":"blog","authToken":"abc"}]`)
	_, err := loadSites(path)
	if err == nil || !strings.Contains(err.Error(), "chatId is required") {
		t.Fatalf("expected chatId required error, got: %v", err)
	}
}

func TestLoadSites_DuplicateName(t *testing.T) {
	path := writeSitesFile(t, `[
		{"name":"blog","authToken":"abc","chatId":"123"},
		{"name":"blog","authToken":"def","chatId":"456"}
	]`)
	_, err := loadSites(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate name") {
		t.Fatalf("expected duplicate name error, got: %v", err)
	}
}

func TestLoadSites_DuplicateToken(t *testing.T) {
	path := writeSitesFile(t, `[
		{"name":"blog","authToken":"abc","chatId":"123"},
		{"name":"portfolio","authToken":"abc","chatId":"456"}
	]`)
	_, err := loadSites(path)
	if err == nil || !strings.Contains(err.Error(), "duplicate authToken") {
		t.Fatalf("expected duplicate authToken error, got: %v", err)
	}
}
