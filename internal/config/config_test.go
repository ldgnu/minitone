package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHasNavidrome(t *testing.T) {
	c := &Config{}
	if c.HasNavidrome() {
		t.Fatal("empty should be false")
	}
	c.NavidromeURL = "http://x"
	c.NavidromeUser = "u"
	c.NavidromePass = "p"
	if !c.HasNavidrome() {
		t.Fatal("expected true")
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("NAVIDROME_URL", "http://nav.example")
	t.Setenv("NAVIDROME_USER", "alice")
	t.Setenv("NAVIDROME_PASS", "secret")
	t.Setenv("MINITONE_THEME", "nord")

	cfg := Load()
	if cfg.NavidromeURL != "http://nav.example" {
		t.Fatalf("url %q", cfg.NavidromeURL)
	}
	if cfg.NavidromeUser != "alice" {
		t.Fatalf("user %q", cfg.NavidromeUser)
	}
	if cfg.Theme != "nord" {
		t.Fatalf("theme %q", cfg.Theme)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	// Point HOME so config lands in temp.
	t.Setenv("HOME", dir)
	// Also clear navidrome env so file wins for theme etc.
	t.Setenv("NAVIDROME_URL", "")
	t.Setenv("NAVIDROME_USER", "")
	t.Setenv("NAVIDROME_PASS", "")
	t.Setenv("MINITONE_THEME", "")
	t.Setenv("AMUSIC_THEME", "")

	cfg := &Config{
		Theme:         "dracula",
		Volume:        42,
		NavidromeURL:  "http://localhost:4533",
		NavidromeUser: "u",
		NavidromePass: "p",
		LibraryPaths:  []string{"/music"},
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".config", "minitone", "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	loaded := Load()
	if loaded.Theme != "dracula" {
		t.Fatalf("theme %q", loaded.Theme)
	}
	if loaded.Volume != 42 {
		t.Fatalf("vol %d", loaded.Volume)
	}
	if !loaded.HasNavidrome() {
		t.Fatal("navidrome")
	}
}
