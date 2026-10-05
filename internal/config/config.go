package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// DefaultSearchDebounce is the delay before a keystroke turns into a search.
	DefaultSearchDebounce = 300 * 1000 * 1000 // 300ms in nanoseconds
	// DefaultDebounceMin / Max bound the configurable debounce.
	DefaultDebounceMin = 100 * 1000 * 1000 // 100ms
	DefaultDebounceMax = 800 * 1000 * 1000 // 800ms
)

// Config is the on-disk configuration (~/.config/minitone/config.json).
//
// Every field is optional: Load fills sensible defaults for anything missing so
// an empty (or absent) config file is always valid.
type Config struct {
	NavidromeURL  string   `json:"navidrome_url"`
	NavidromeUser string   `json:"navidrome_user"`
	NavidromePass string   `json:"navidrome_pass"`
	Theme         string   `json:"theme"`
	LibraryPaths  []string `json:"library_paths"`
	Volume        int      `json:"volume"`

	// RestoreSession saves and restores the last track, position, queue,
	// volume, shuffle and repeat across runs.
	RestoreSession bool `json:"restore_session"`

	// DefaultSource limits searches to a single source ("all" for every one).
	DefaultSource string `json:"default_source"`

	// SearchDebounceMs is the keystroke debounce, in milliseconds.
	SearchDebounceMs int `json:"search_debounce_ms"`

	// KeyBindings allows overriding a few actions: {"queue": "ctrl+q"}.
	KeyBindings map[string]string `json:"keybindings,omitempty"`
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "minitone", "config.json")
}

// Path returns the config file location ("" when the home dir is unknown).
func Path() string { return configPath() }

func Load() *Config {
	cfg := &Config{
		Theme:            "terminal",
		Volume:           70,
		RestoreSession:   true,
		DefaultSource:    "all",
		SearchDebounceMs: 300,
	}

	if p := configPath(); p != "" {
		if data, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(data, cfg)
		}
	}

	if v := os.Getenv("NAVIDROME_URL"); v != "" {
		cfg.NavidromeURL = v
	}
	if v := os.Getenv("NAVIDROME_USER"); v != "" {
		cfg.NavidromeUser = v
	}
	if v := os.Getenv("NAVIDROME_PASS"); v != "" {
		cfg.NavidromePass = v
	}
	if v := os.Getenv("MINITONE_THEME"); v != "" {
		cfg.Theme = v
	}
	if v := os.Getenv("MINITONE_VOLUME"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Volume = n
		}
	}
	if v := os.Getenv("MINITONE_RESTORE_SESSION"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.RestoreSession = b
		}
	}
	if v := os.Getenv("MINITONE_LIBRARY"); v != "" {
		for _, p := range strings.Split(v, string(os.PathListSeparator)) {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.LibraryPaths = append(cfg.LibraryPaths, p)
			}
		}
	}

	cfg.normalize()
	return cfg
}

// normalize repairs out-of-range or missing values after unmarshalling.
func (c *Config) normalize() {
	if c.Theme == "" {
		c.Theme = "terminal"
	}
	if c.Volume <= 0 || c.Volume > 100 {
		c.Volume = 70
	}
	if c.DefaultSource == "" {
		c.DefaultSource = "all"
	}
	c.DefaultSource = strings.ToLower(strings.TrimSpace(c.DefaultSource))
	if c.SearchDebounceMs <= 0 {
		c.SearchDebounceMs = 300
	}
	if c.SearchDebounceMs < 100 {
		c.SearchDebounceMs = 100
	}
	if c.SearchDebounceMs > 800 {
		c.SearchDebounceMs = 800
	}

	// Sensible default: ~/Music when the user configured nothing.
	if len(c.LibraryPaths) == 0 {
		if home, err := os.UserHomeDir(); err == nil {
			for _, name := range []string{"Music", "Música", "music"} {
				dir := filepath.Join(home, name)
				if st, err := os.Stat(dir); err == nil && st.IsDir() {
					c.LibraryPaths = append(c.LibraryPaths, dir)
				}
			}
		}
	}
}

// Debounce returns the configured debounce clamped to a sane range.
func (c *Config) Debounce() int {
	n := c.SearchDebounceMs
	if n <= 0 {
		n = 300
	}
	if n < 100 {
		n = 100
	}
	if n > 800 {
		n = 800
	}
	return n
}

func (c *Config) HasNavidrome() bool {
	return c.NavidromeURL != "" && c.NavidromeUser != "" && c.NavidromePass != ""
}

// Save writes the config to the default path (best-effort).
func (c *Config) Save() error {
	p := configPath()
	if p == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}
