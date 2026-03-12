package services

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kingknull/oblivrashell/internal/platform"
)

type AppConfig struct {
	WindowWidth       int     `json:"window_width"`
	WindowHeight      int     `json:"window_height"`
	Maximized         bool    `json:"maximized"`
	FontFamily        string  `json:"font_family"`
	FontSize          int     `json:"font_size"`
	LineHeight        float64 `json:"line_height"`
	CursorStyle       string  `json:"cursor_style"`
	CursorBlink       bool    `json:"cursor_blink"`
	ScrollbackLines   int     `json:"scrollback_lines"`
	Theme             string  `json:"theme"`
	AutoLockTimeout   int     `json:"auto_lock_timeout"`
	ClipboardClear    int     `json:"clipboard_clear"`
	LogSessions       bool    `json:"log_sessions"`
	KeepAliveInterval int     `json:"keepalive_interval"`
	ConnectionTimeout int     `json:"connection_timeout"`
}

func DefaultConfig() *AppConfig {
	return &AppConfig{
		WindowWidth:       1280,
		WindowHeight:      800,
		Maximized:         false,
		FontFamily:        "JetBrains Mono, Fira Code, monospace",
		FontSize:          14,
		LineHeight:        1.2,
		CursorStyle:       "block",
		CursorBlink:       true,
		ScrollbackLines:   10000,
		Theme:             "dark",
		AutoLockTimeout:   15,
		ClipboardClear:    30,
		LogSessions:       true,
		KeepAliveInterval: 30,
		ConnectionTimeout: 10,
	}
}

func LoadConfig() (*AppConfig, error) {
	configPath := filepath.Join(platform.ConfigDir(), "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			return cfg, SaveConfig(cfg)
		}
		return nil, err
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveConfig(cfg *AppConfig) error {
	configPath := filepath.Join(platform.ConfigDir(), "config.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}
