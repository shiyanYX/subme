package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadProviderConfig(t *testing.T) {
	yaml := `clash_name: my-provider
panel_url: https://panel.example.com
landing_page: https://example.com/blog
username: user@example.com
password: secret123
interval: 1800
`
	path := writeTempYAML(t, yaml)
	pc, err := LoadProviderConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if pc.ClashName != "my-provider" {
		t.Errorf("ClashName: got %q, want %q", pc.ClashName, "my-provider")
	}
	if pc.PanelURL != "https://panel.example.com" {
		t.Errorf("PanelURL: got %q, want %q", pc.PanelURL, "https://panel.example.com")
	}
	if pc.LandingPage != "https://example.com/blog" {
		t.Errorf("LandingPage: got %q, want %q", pc.LandingPage, "https://example.com/blog")
	}
	if pc.Username != "user@example.com" {
		t.Errorf("Username: got %q, want %q", pc.Username, "user@example.com")
	}
	if pc.Password != "secret123" {
		t.Errorf("Password: got %q, want %q", pc.Password, "secret123")
	}
	if pc.Interval != 1800 {
		t.Errorf("Interval: got %d, want %d", pc.Interval, 1800)
	}
}

func TestLoadProviderConfig_DefaultInterval(t *testing.T) {
	yaml := `clash_name: my-provider
`
	path := writeTempYAML(t, yaml)
	pc, err := LoadProviderConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if pc.Interval != 3600 {
		t.Errorf("default interval: got %d, want %d", pc.Interval, 3600)
	}
}

func TestLoadProviderConfig_MissingClashName(t *testing.T) {
	yaml := `panel_url: https://example.com`
	path := writeTempYAML(t, yaml)
	_, err := LoadProviderConfig(path)
	if err == nil {
		t.Fatal("expected error for missing clash_name")
	}
}

func TestLoadProviderConfig_FileNotFound(t *testing.T) {
	_, err := LoadProviderConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadProviderConfig_InvalidYAML(t *testing.T) {
	path := writeTempYAML(t, `invalid yaml: [bad`)
	_, err := LoadProviderConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "roundtrip.yaml")

	orig := &ProviderConfig{
		ClashName:   "test-provider",
		Interval:    7200,
		PanelURL:    "https://panel.test.com",
		LandingPage: "https://landing.test.com",
		Username:    "admin",
		Password:    "p@ss",
	}

	if err := SaveProviderConfig(path, orig); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadProviderConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.ClashName != orig.ClashName {
		t.Errorf("ClashName: got %q, want %q", loaded.ClashName, orig.ClashName)
	}
	if loaded.Interval != orig.Interval {
		t.Errorf("Interval: got %d, want %d", loaded.Interval, orig.Interval)
	}
	if loaded.PanelURL != orig.PanelURL {
		t.Errorf("PanelURL: got %q, want %q", loaded.PanelURL, orig.PanelURL)
	}
	if loaded.Username != orig.Username {
		t.Errorf("Username: got %q, want %q", loaded.Username, orig.Username)
	}
}

func TestUpdateProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "update.yaml")

	orig := &ProviderConfig{
		ClashName:   "test",
		PanelURL:    "https://old.example.com",
		LandingPage: "https://old-landing.example.com",
		Username:    "olduser",
		Password:    "oldpass",
	}
	if err := SaveProviderConfig(path, orig); err != nil {
		t.Fatal(err)
	}

	updates := map[string]string{
		"panel_url":    "https://new.example.com",
		"landing_page": "https://new-landing.example.com",
		"username":     "newuser",
		"password":     "newpass",
	}
	if err := UpdateProviderConfig(path, updates); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadProviderConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.PanelURL != "https://new.example.com" {
		t.Errorf("PanelURL: got %q", loaded.PanelURL)
	}
	if loaded.LandingPage != "https://new-landing.example.com" {
		t.Errorf("LandingPage: got %q", loaded.LandingPage)
	}
	if loaded.Username != "newuser" {
		t.Errorf("Username: got %q", loaded.Username)
	}
	if loaded.Password != "newpass" {
		t.Errorf("Password: got %q", loaded.Password)
	}
}

func TestUpdateProviderConfig_UnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unknown-key.yaml")

	orig := &ProviderConfig{ClashName: "test"}
	if err := SaveProviderConfig(path, orig); err != nil {
		t.Fatal(err)
	}

	if err := UpdateProviderConfig(path, map[string]string{"unknown": "val"}); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadProviderConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ClashName != "test" {
		t.Errorf("ClashName changed: got %q", loaded.ClashName)
	}
}
