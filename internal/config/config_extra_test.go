package config

import (
	"path/filepath"
	"testing"
)

func TestNormalizeDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := &Config{}
	cfg.normalize()

	if cfg.Theme != "terminal" {
		t.Fatalf("theme %q", cfg.Theme)
	}
	if cfg.Volume != 70 {
		t.Fatalf("volume %d", cfg.Volume)
	}
	if cfg.DefaultSource != "all" {
		t.Fatalf("default_source %q", cfg.DefaultSource)
	}
	if cfg.Debounce() != 300 {
		t.Fatalf("debounce %d", cfg.Debounce())
	}
	if cfg.RestoreSession != false {
		t.Fatal("restore_session should default to false on a bare struct")
	}
}

func TestNormalizeClamps(t *testing.T) {
	cfg := &Config{
		Volume:           900,
		SearchDebounceMs: 5,
		DefaultSource:    "  YouTube ",
	}
	cfg.normalize()

	if cfg.Volume != 70 {
		t.Fatalf("volume should clamp to default, got %d", cfg.Volume)
	}
	if cfg.Debounce() != 100 {
		t.Fatalf("debounce lower clamp, got %d", cfg.Debounce())
	}
	if cfg.DefaultSource != "youtube" {
		t.Fatalf("default_source should be lowercased/trimmed, got %q", cfg.DefaultSource)
	}

	cfg.SearchDebounceMs = 99999
	cfg.normalize()
	if cfg.Debounce() != 800 {
		t.Fatalf("debounce upper clamp, got %d", cfg.Debounce())
	}
}

func TestLoadDefaultsWhenNoFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("NAVIDROME_URL", "")
	t.Setenv("NAVIDROME_USER", "")
	t.Setenv("NAVIDROME_PASS", "")
	t.Setenv("MINITONE_THEME", "")

	cfg := Load()
	if cfg.Theme != "terminal" || cfg.Volume != 70 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.DefaultSource != "all" {
		t.Fatalf("default_source %q", cfg.DefaultSource)
	}
	if cfg.SearchDebounceMs != 300 {
		t.Fatalf("search_debounce_ms %d", cfg.SearchDebounceMs)
	}
	// No ~/Music in a temp home, so no library paths.
	if len(cfg.LibraryPaths) != 0 {
		t.Fatalf("library paths %v", cfg.LibraryPaths)
	}
	// A config dir should be resolvable even with no file present.
	if Path() != filepath.Join(home, ".config", "minitone", "config.json") {
		t.Fatalf("path %q", Path())
	}
}

func TestEnvSessionOverrides(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MINITONE_RESTORE_SESSION", "false")
	t.Setenv("MINITONE_VOLUME", "33")

	cfg := Load()
	if cfg.RestoreSession {
		t.Fatal("env should disable restore_session")
	}
	if cfg.Volume != 33 {
		t.Fatalf("volume %d", cfg.Volume)
	}
}

func TestSaveLoadNewFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := Load()
	cfg.RestoreSession = true
	cfg.DefaultSource = "radio"
	cfg.SearchDebounceMs = 450
	cfg.KeyBindings = map[string]string{"queue": "ctrl+q"}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded := Load()
	if !loaded.RestoreSession {
		t.Fatal("restore_session did not round-trip")
	}
	if loaded.DefaultSource != "radio" {
		t.Fatalf("default_source %q", loaded.DefaultSource)
	}
	if loaded.Debounce() != 450 {
		t.Fatalf("debounce %d", loaded.Debounce())
	}
	if loaded.KeyBindings["queue"] != "ctrl+q" {
		t.Fatalf("keybindings %v", loaded.KeyBindings)
	}
}
