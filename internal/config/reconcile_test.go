package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDeviceID_Serial(t *testing.T) {
	id, err := ResolveDeviceID(t.TempDir(), "ABC123")
	if err != nil {
		t.Fatal(err)
	}
	if id != "ABC123" {
		t.Errorf("expected ABC123, got %s", id)
	}
}

func TestResolveDeviceID_FromExistingConfig(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "iPod_Control", "Clickwheel")
	os.MkdirAll(cfgDir, 0755)

	cfg := DefaultDevice("existing-id")
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(cfgDir, "config.json"), data, 0644)

	id, err := ResolveDeviceID(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if id != "existing-id" {
		t.Errorf("expected existing-id, got %s", id)
	}
}

func TestResolveDeviceID_GeneratesUUID(t *testing.T) {
	id, err := ResolveDeviceID(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Error("expected non-empty UUID")
	}
	if len(id) < 32 {
		t.Errorf("UUID too short: %s", id)
	}
}

func TestLoadDeviceConfig_IPodOnly(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "iPod_Control", "Clickwheel")
	os.MkdirAll(cfgDir, 0755)

	cfg := DefaultDevice("dev-1")
	cfg.Inclusions.Playlists = []string{"pl-a"}
	cfg.LastModified = 1000
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(cfgDir, "config.json"), data, 0644)

	loaded, err := LoadDeviceConfig(dir, "dev-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Inclusions.Playlists) != 1 || loaded.Inclusions.Playlists[0] != "pl-a" {
		t.Errorf("Playlists: got %v", loaded.Inclusions.Playlists)
	}
}

func TestLoadDeviceConfig_NeitherExists(t *testing.T) {
	loaded, err := LoadDeviceConfig(t.TempDir(), "new-device")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DeviceID != "new-device" {
		t.Errorf("DeviceID: got %s", loaded.DeviceID)
	}
	if loaded.SyncSettings.MusicFormat != "aac" {
		t.Errorf("expected default music format")
	}
}

func TestLoadDeviceConfig_BothExist_NewerWins(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "iPod_Control", "Clickwheel")
	os.MkdirAll(cfgDir, 0755)

	ipodCfg := DefaultDevice("dev-2")
	ipodCfg.Inclusions.Playlists = []string{"old"}
	ipodCfg.LastModified = 100
	ipodData, _ := json.MarshalIndent(ipodCfg, "", "  ")
	os.WriteFile(filepath.Join(cfgDir, "config.json"), ipodData, 0644)

	backupDir := t.TempDir()
	backupCfg := DefaultDevice("dev-2")
	backupCfg.Inclusions.Playlists = []string{"newer"}
	backupCfg.LastModified = 200
	backupData, _ := json.MarshalIndent(backupCfg, "", "  ")

	origFunc := DeviceBackupPath
	DeviceBackupPath = func(id string) (string, error) {
		p := filepath.Join(backupDir, id, "config.json")
		os.MkdirAll(filepath.Dir(p), 0755)
		return p, nil
	}
	defer func() { DeviceBackupPath = origFunc }()

	os.MkdirAll(filepath.Join(backupDir, "dev-2"), 0755)
	os.WriteFile(filepath.Join(backupDir, "dev-2", "config.json"), backupData, 0644)

	loaded, err := LoadDeviceConfig(dir, "dev-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Inclusions.Playlists) != 1 || loaded.Inclusions.Playlists[0] != "newer" {
		t.Errorf("expected newer playlists, got %v", loaded.Inclusions.Playlists)
	}
}
