package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type SubsonicConfig struct {
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

type SyncState struct {
	TrackPlayCounts  map[string]TrackSyncState `json:"trackPlayCounts"`
	TransferRate     float64                   `json:"transferRate,omitempty"`
}

type Exclusions struct {
	Playlists []string `json:"playlists"`
	Albums    []string `json:"albums"`
	Artists   []string `json:"artists"`
	Books     []string `json:"books"`
	Podcasts  []string `json:"podcasts"`
}

type SyncSettings struct {
	SyncPlayCounts      bool   `json:"syncPlayCounts"`
	SyncBookPosition    bool   `json:"syncBookPosition"`
	TwoWayBookSync      bool   `json:"twoWayBookSync"`
	SplitLongBooks      bool   `json:"splitLongBooks"`
	SplitHoursLimit     int    `json:"splitHoursLimit"`
	MusicFormat         string `json:"musicFormat"`
	MusicBitRate        int    `json:"musicBitRate"`
	SyncPodcastPosition bool   `json:"syncPodcastPosition"`
	TwoWayPodcastSync   bool   `json:"twoWayPodcastSync"`
}

type Config struct {
	Subsonic     SubsonicConfig  `json:"subsonic"`
	ABS          ABSConfig       `json:"abs"`
	SyncState    SyncState       `json:"syncState"`
	Exclusions   Exclusions      `json:"exclusions"`
	SyncSettings SyncSettings    `json:"syncSettings"`
	path         string
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
		},
		SyncSettings: SyncSettings{
			SyncPlayCounts:      true,
			SyncBookPosition:    true,
			TwoWayBookSync:      false,
			SyncPodcastPosition: true,
			TwoWayPodcastSync:   false,
			SplitLongBooks:      true,
			SplitHoursLimit:     8,
			MusicFormat:         "aac",
			MusicBitRate:        256,
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
