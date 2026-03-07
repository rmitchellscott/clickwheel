package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type NavidromeConfig struct {
	ServerURL string `json:"serverUrl"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type ABSConfig struct {
	ServerURL string `json:"serverUrl"`
	Token     string `json:"token"`
}

type TrackSyncState struct {
	PlayCount int   `json:"playCount"`
	LastSync  int64 `json:"lastSync"`
}

type BookSyncState struct {
	CurrentTime float64 `json:"currentTime"`
	Duration    float64 `json:"duration"`
	LastSync    int64   `json:"lastSync"`
}

type SyncState struct {
	TrackPlayCounts map[string]TrackSyncState `json:"trackPlayCounts"`
	BookProgress    map[string]BookSyncState  `json:"bookProgress"`
}

type Exclusions struct {
	Playlists []string `json:"playlists"`
	Albums    []string `json:"albums"`
	Books     []string `json:"books"`
}

type Config struct {
	Navidrome  NavidromeConfig `json:"navidrome"`
	ABS        ABSConfig       `json:"abs"`
	SyncState  SyncState       `json:"syncState"`
	Exclusions Exclusions      `json:"exclusions"`
	path       string
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "clickwheel", "config.json"), nil
}

func Default() *Config {
	p, _ := configPath()
	return &Config{
		SyncState: SyncState{
			TrackPlayCounts: make(map[string]TrackSyncState),
			BookProgress:    make(map[string]BookSyncState),
		},
		path: p,
	}
}

func Load() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	cfg := Default()
	cfg.path = p
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.SyncState.TrackPlayCounts == nil {
		cfg.SyncState.TrackPlayCounts = make(map[string]TrackSyncState)
	}
	if cfg.SyncState.BookProgress == nil {
		cfg.SyncState.BookProgress = make(map[string]BookSyncState)
	}

	return cfg, nil
}

func (c *Config) Save() error {
	if c.path == "" {
		p, err := configPath()
		if err != nil {
			return err
		}
		c.path = p
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0600)
}
