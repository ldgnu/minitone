package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	NavidromeURL  string   `json:"navidrome_url"`
	NavidromeUser string   `json:"navidrome_user"`
	NavidromePass string   `json:"navidrome_pass"`
	Theme         string   `json:"theme"`
	LibraryPaths  []string `json:"library_paths"`
	Volume        int      `json:"volume"`
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "minitone", "config.json")
}

func Load() *Config {
	cfg := &Config{
		Theme:  "terminal",
		Volume: 70,
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
	if v := os.Getenv("AMUSIC_THEME"); v != "" {
		cfg.Theme = v
	}
	if v := os.Getenv("MINITONE_THEME"); v != "" {
		cfg.Theme = v
	}
	if v := os.Getenv("MINITONE_LIBRARY"); v != "" {
		for _, p := range strings.Split(v, string(os.PathListSeparator)) {
			p = strings.TrimSpace(p)
			if p != "" {
				cfg.LibraryPaths = append(cfg.LibraryPaths, p)
			}
		}
	}

	// Sensible defaults if nothing configured.
	if len(cfg.LibraryPaths) == 0 {
		if home, err := os.UserHomeDir(); err == nil {
			for _, name := range []string{"Music", "Música", "music"} {
				dir := filepath.Join(home, name)
				if st, err := os.Stat(dir); err == nil && st.IsDir() {
					cfg.LibraryPaths = append(cfg.LibraryPaths, dir)
				}
			}
		}
	}

	if cfg.Volume <= 0 || cfg.Volume > 100 {
		cfg.Volume = 70
	}
	return cfg
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
