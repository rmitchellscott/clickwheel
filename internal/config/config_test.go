package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultDevice(t *testing.T) {
	d := DefaultDevice("test-123")
	if d.DeviceID != "test-123" {
		t.Errorf("expected DeviceID test-123, got %s", d.DeviceID)
	}
	if d.SyncSettings.MusicFormat != "aac" {
		t.Errorf("expected default music format aac, got %s", d.SyncSettings.MusicFormat)
	}
	if d.SyncState.TrackPlayCounts == nil {
		t.Error("expected TrackPlayCounts to be initialized")
	}
}

func TestDeviceConfigSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := DefaultDevice("dev-abc")
	cfg.Inclusions.Playlists = []string{"pl-1", "pl-2"}
	cfg.SyncSettings.MusicBitRate = 128
	cfg.SetPath(path)

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := loadDeviceFrom(path)
	if err != nil {
		t.Fatalf("loadDeviceFrom: %v", err)
	}

	if loaded.DeviceID != "dev-abc" {
		t.Errorf("DeviceID: got %s, want dev-abc", loaded.DeviceID)
	}
	if len(loaded.Inclusions.Playlists) != 2 {
		t.Errorf("Playlists: got %d, want 2", len(loaded.Inclusions.Playlists))
	}
	if loaded.SyncSettings.MusicBitRate != 128 {
		t.Errorf("MusicBitRate: got %d, want 128", loaded.SyncSettings.MusicBitRate)
	}
}

func TestHostConfigSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "host.json")

	cfg := &HostConfig{
		Subsonic: SubsonicConfig{
			ServerURL: "https://music.example.com",
			Username:  "user",
			Password:  "pass",
		},
		ABS: ABSConfig{
			ServerURL: "https://abs.example.com",
			Token:     "tok123",
		},
	}
	cfg.path = path

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var loaded HostConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if loaded.Subsonic.ServerURL != "https://music.example.com" {
		t.Errorf("ServerURL: got %s", loaded.Subsonic.ServerURL)
	}
	if loaded.ABS.Token != "tok123" {
		t.Errorf("Token: got %s", loaded.ABS.Token)
	}
}

func TestDeviceConfigSaveBoth(t *testing.T) {
	ipodDir := t.TempDir()
	ipodPath := filepath.Join(ipodDir, "iPod_Control", "Clickwheel", "config.json")

	origBackupPath := DeviceBackupPath
	_ = origBackupPath

	cfg := DefaultDevice("serial-xyz")
	cfg.Inclusions.Books = []string{"book-1"}

	if err := os.MkdirAll(filepath.Join(ipodDir, "iPod_Control", "Clickwheel"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg.SetPath(ipodPath)
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save iPod: %v", err)
	}

	loaded, err := loadDeviceFrom(ipodPath)
	if err != nil {
		t.Fatalf("loadDeviceFrom: %v", err)
	}
	if len(loaded.Inclusions.Books) != 1 || loaded.Inclusions.Books[0] != "book-1" {
		t.Errorf("Books: got %v", loaded.Inclusions.Books)
	}
}
