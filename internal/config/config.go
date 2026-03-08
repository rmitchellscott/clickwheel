package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
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
	TrackPlayCounts map[string]TrackSyncState `json:"trackPlayCounts"`
	TransferRate    float64                   `json:"transferRate,omitempty"`
}

type Inclusions struct {
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

type ServerURLs struct {
	SubsonicURL string `json:"subsonicUrl,omitempty"`
	ABSURL      string `json:"absUrl,omitempty"`
}

type KnownDevice struct {
	DeviceID   string `json:"deviceId"`
	Name       string `json:"name"`
	Family     string `json:"family,omitempty"`
	Generation string `json:"generation,omitempty"`
	Capacity   string `json:"capacity,omitempty"`
	Color      string `json:"color,omitempty"`
	Model      string `json:"model,omitempty"`
	Icon       string `json:"icon,omitempty"`
}

type HostConfig struct {
	Subsonic     SubsonicConfig `json:"subsonic"`
	ABS          ABSConfig      `json:"abs"`
	LastDeviceID string         `json:"lastDeviceId,omitempty"`
	KnownDevices []KnownDevice  `json:"knownDevices,omitempty"`
	path         string
}

type DeviceConfig struct {
	DeviceID     string       `json:"deviceId"`
	LastModified int64        `json:"lastModified"`
	Servers      ServerURLs   `json:"servers"`
	Inclusions   Inclusions   `json:"inclusions"`
	SyncSettings SyncSettings `json:"syncSettings"`
	SyncState    SyncState    `json:"syncState"`
	path         string
}

func hostConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "clickwheel", "config.json"), nil
}

var DeviceBackupPath = func(deviceID string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "clickwheel", "devices", deviceID, "config.json"), nil
}

func IPodConfigPath(mountPoint string) string {
	return filepath.Join(mountPoint, "iPod_Control", "Clickwheel", "config.json")
}

func DefaultHost() *HostConfig {
	p, _ := hostConfigPath()
	return &HostConfig{path: p}
}

func DefaultDevice(deviceID string) *DeviceConfig {
	return &DeviceConfig{
		DeviceID: deviceID,
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
	}
}

func LoadHost() (*HostConfig, error) {
	p, err := hostConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	cfg := &HostConfig{path: p}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *HostConfig) Save() error {
	if c.path == "" {
		p, err := hostConfigPath()
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

func loadDeviceFrom(path string) (*DeviceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &DeviceConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	cfg.path = path
	if cfg.SyncState.TrackPlayCounts == nil {
		cfg.SyncState.TrackPlayCounts = make(map[string]TrackSyncState)
	}
	return cfg, nil
}

func (c *DeviceConfig) SetPath(p string) {
	c.path = p
}

func (c *DeviceConfig) Save() error {
	if c.path == "" {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0644)
}

func (c *DeviceConfig) SaveBoth(mountPoint string) error {
	c.LastModified = now()

	ipodPath := IPodConfigPath(mountPoint)
	c.path = ipodPath
	if err := c.Save(); err != nil {
		return err
	}

	backupPath, err := DeviceBackupPath(c.DeviceID)
	if err != nil {
		return err
	}
	c.path = backupPath
	return c.Save()
}

func (c *HostConfig) UpdateKnownDevice(dev KnownDevice) {
	for i, d := range c.KnownDevices {
		if d.DeviceID == dev.DeviceID {
			c.KnownDevices[i] = dev
			return
		}
	}
	c.KnownDevices = append(c.KnownDevices, dev)
}

func LoadDeviceFromBackup(deviceID string) (*DeviceConfig, error) {
	bp, err := DeviceBackupPath(deviceID)
	if err != nil {
		return nil, err
	}
	return loadDeviceFrom(bp)
}

func now() int64 {
	return time.Now().Unix()
}
