package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/config"
	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
	"clickwheel/internal/subsonic"
	"clickwheel/internal/sync"
)

type App struct {
	ctx        context.Context
	host       *config.HostConfig
	device     *config.DeviceConfig
	subClient  *subsonic.Client
	absClient  *audiobookshelf.Client
	cancelSync context.CancelFunc
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	host, err := config.LoadHost()
	if err != nil {
		host = config.DefaultHost()
	}
	a.host = host

	if a.host.LastDeviceID != "" {
		if devCfg, err := config.LoadDeviceFromBackup(a.host.LastDeviceID); err == nil {
			a.device = devCfg
		}
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.host != nil {
		_ = a.host.Save()
	}
}

type MergedConfig struct {
	Subsonic     config.SubsonicConfig `json:"subsonic"`
	ABS          config.ABSConfig      `json:"abs"`
	SyncSettings config.SyncSettings   `json:"syncSettings"`
	Inclusions   config.Inclusions     `json:"inclusions"`
}

func (a *App) GetConfig() *MergedConfig {
	mc := &MergedConfig{
		Subsonic: a.host.Subsonic,
		ABS:      a.host.ABS,
	}
	if a.device != nil {
		mc.SyncSettings = a.device.SyncSettings
		mc.Inclusions = a.device.Inclusions
	}
	return mc
}

func (a *App) GetTimezone() string {
	if tz := os.Getenv("TZ"); tz != "" && tz != "Local" {
		return tz
	}
	if target, err := os.Readlink("/etc/localtime"); err == nil {
		if idx := strings.Index(target, "zoneinfo/"); idx != -1 {
			return target[idx+len("zoneinfo/"):]
		}
	}
	loc := time.Now().Location().String()
	if loc != "Local" {
		return loc
	}
	return "UTC"
}

func (a *App) SaveSubsonicConfig(serverURL, username, password string) error {
	a.host.Subsonic.ServerURL = serverURL
	a.host.Subsonic.Username = username
	a.host.Subsonic.Password = password
	a.subClient = subsonic.NewClient(serverURL, username, password)
	if err := a.host.Save(); err != nil {
		return err
	}
	if a.device != nil {
		a.device.Servers.SubsonicURL = serverURL
		return a.saveDevice()
	}
	return nil
}

func (a *App) SaveABSConfig(serverURL, token string) error {
	a.host.ABS.ServerURL = serverURL
	a.host.ABS.Token = token
	a.absClient = audiobookshelf.NewClient(serverURL, token)
	if err := a.host.Save(); err != nil {
		return err
	}
	if a.device != nil {
		a.device.Servers.ABSURL = serverURL
		return a.saveDevice()
	}
	return nil
}

func (a *App) TestSubsonicConnection() error {
	if a.subClient == nil {
		a.subClient = subsonic.NewClient(
			a.host.Subsonic.ServerURL,
			a.host.Subsonic.Username,
			a.host.Subsonic.Password,
		)
	}
	return a.subClient.Ping()
}

func (a *App) TestABSConnection() error {
	if a.absClient == nil {
		a.absClient = audiobookshelf.NewClient(a.host.ABS.ServerURL, a.host.ABS.Token)
	}
	return a.absClient.Ping()
}

func (a *App) DetectIPod() (*ipod.DeviceInfo, error) {
	info, err := ipod.Detect()
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	deviceID, err := config.ResolveDeviceID(info.MountPoint, info.SerialNumber)
	if err != nil {
		return nil, err
	}

	devCfg, err := config.LoadDeviceConfig(info.MountPoint, deviceID)
	if err != nil {
		return nil, err
	}

	devCfg.Servers.SubsonicURL = a.host.Subsonic.ServerURL
	devCfg.Servers.ABSURL = a.host.ABS.ServerURL
	a.device = devCfg

	a.host.LastDeviceID = deviceID
	a.host.UpdateKnownDevice(config.KnownDevice{
		DeviceID:   deviceID,
		Name:       info.Name,
		Family:     info.Family,
		Generation: info.Generation,
		Capacity:   info.Capacity,
		Color:      info.Color,
		Model:      info.Model,
		Icon:       info.Icon,
	})
	_ = a.host.Save()

	return info, nil
}

func (a *App) EjectIPod() error {
	info, err := ipod.Detect()
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("no iPod connected")
	}
	if a.device != nil {
		_ = a.saveDevice()
	}
	return ipod.Eject(info.MountPoint)
}

func (a *App) GetKnownDevices() []config.KnownDevice {
	if a.host == nil {
		return nil
	}
	return a.host.KnownDevices
}

func (a *App) GetActiveDeviceID() string {
	if a.device == nil {
		return ""
	}
	return a.device.DeviceID
}

func (a *App) SwitchDevice(deviceID string) error {
	devCfg, err := config.LoadDeviceFromBackup(deviceID)
	if err != nil {
		return fmt.Errorf("no backup found for device %s", deviceID)
	}
	a.device = devCfg
	a.host.LastDeviceID = deviceID
	return a.host.Save()
}

func (a *App) GetSubsonicPlaylists() ([]subsonic.Playlist, error) {
	if a.subClient == nil {
		return nil, nil
	}
	return a.subClient.GetPlaylists()
}

func (a *App) GetSubsonicAlbums() ([]subsonic.Album, error) {
	if a.subClient == nil {
		return nil, nil
	}
	return a.subClient.GetAlbums(0, 500)
}

func (a *App) GetSubsonicArtists() ([]subsonic.Artist, error) {
	if a.subClient == nil {
		return nil, nil
	}
	return a.subClient.GetArtists()
}

func (a *App) GetABSLibraries() ([]audiobookshelf.Library, error) {
	if a.absClient == nil {
		return nil, nil
	}
	return a.absClient.GetLibraries()
}

func (a *App) GetABSBooks(libraryID string) ([]audiobookshelf.Book, error) {
	if a.absClient == nil {
		return nil, nil
	}
	return a.absClient.GetBooks(libraryID)
}

func (a *App) GetABSPodcasts(libraryID string) ([]audiobookshelf.Podcast, error) {
	if a.absClient == nil {
		return nil, nil
	}
	return a.absClient.GetPodcasts(libraryID)
}

func (a *App) GetABSProgress() (map[string]audiobookshelf.MediaProgress, error) {
	if a.absClient == nil {
		return nil, nil
	}
	return a.absClient.GetAllProgress()
}

func (a *App) GetInclusions() config.Inclusions {
	if a.device == nil {
		return config.Inclusions{}
	}
	return a.device.Inclusions
}

func (a *App) SetInclusions(inclusions config.Inclusions) error {
	if a.device == nil {
		return fmt.Errorf("no device connected")
	}
	a.device.Inclusions = inclusions
	return a.saveDevice()
}

func (a *App) GetSyncSettings() config.SyncSettings {
	if a.device != nil {
		return a.device.SyncSettings
	}
	return config.DefaultDevice("").SyncSettings
}

func (a *App) SaveSyncSettings(settings config.SyncSettings) error {
	if a.device == nil {
		return fmt.Errorf("no device connected")
	}
	a.device.SyncSettings = settings
	return a.saveDevice()
}

func (a *App) saveDevice() error {
	if a.device == nil {
		return nil
	}
	info, err := ipod.Detect()
	if err != nil || info == nil {
		bp, err := config.DeviceBackupPath(a.device.DeviceID)
		if err != nil {
			return err
		}
		a.device.LastModified = time.Now().Unix()
		a.device.SetPath(bp)
		return a.device.Save()
	}
	return a.device.SaveBoth(info.MountPoint)
}

func (a *App) newSyncEngine() *sync.Engine {
	return sync.NewEngine(a.host, a.device, a.subClient, a.absClient)
}

func (a *App) PreviewSync() (*sync.PlanSummary, error) {
	return a.newSyncEngine().Preview(a.ctx)
}

func (a *App) StartSync() error {
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelSync = cancel
	e := a.newSyncEngine()
	go func() {
		defer func() { a.cancelSync = nil }()
		err := e.Run(ctx, func(progress sync.Progress) {
			runtime.EventsEmit(a.ctx, "sync:progress", progress)
		})
		if err != nil {
			if ctx.Err() != nil {
				runtime.EventsEmit(a.ctx, "sync:error", "Sync cancelled")
			} else {
				runtime.EventsEmit(a.ctx, "sync:error", err.Error())
			}
			return
		}
		runtime.EventsEmit(a.ctx, "sync:done", nil)
	}()
	return nil
}

func (a *App) CancelSync() {
	if a.cancelSync != nil {
		a.cancelSync()
	}
}

type IPodTrackInfo struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	Album        string `json:"album"`
	Genre        string `json:"genre"`
	Duration     int    `json:"duration"`
	PlayCount    int    `json:"playCount"`
	LastPlayed   int64  `json:"lastPlayed"`
	DateAdded    int64  `json:"dateAdded"`
	Size         int    `json:"size"`
	Type         string `json:"type"`
	BookmarkTime int    `json:"bookmarkTime"`
}

type IPodPlaylistInfo struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	TrackIDs []string `json:"trackIds"`
}

type StorageBreakdown struct {
	Music      int64 `json:"music"`
	Audiobooks int64 `json:"audiobooks"`
	Podcasts   int64 `json:"podcasts"`
	Other      int64 `json:"other"`
	Free       int64 `json:"free"`
	Total      int64 `json:"total"`
}

func mediaTypeString(mt uint32) string {
	switch mt {
	case itunesdb.MediaTypeAudiobook:
		return "audiobook"
	case itunesdb.MediaTypePodcast, itunesdb.MediaTypeVideoPodcast:
		return "podcast"
	default:
		return "music"
	}
}

func (a *App) GetIPodTracks() ([]IPodTrackInfo, error) {
	info, err := ipod.Detect()
	if err != nil || info == nil {
		return nil, err
	}

	dev, err := ipod.OpenDevice(info)
	if err != nil {
		return nil, err
	}

	var tracks []IPodTrackInfo
	for _, t := range dev.DB.Tracks {
		lastPlayed := int64(0)
		if t.LastPlayed != 0 {
			lastPlayed = itunesdb.FromMacTimestamp(t.LastPlayed).Add(time.Duration(-dev.DB.TZOffset) * time.Second).Unix()
		}
		dateAdded := int64(0)
		if !t.DateAdded.IsZero() {
			dateAdded = t.DateAdded.Unix()
		}

		tracks = append(tracks, IPodTrackInfo{
			ID:           fmt.Sprintf("ipod-%d", t.UniqueID),
			Title:        t.Title,
			Artist:       t.Artist,
			Album:        t.Album,
			Genre:        t.Genre,
			Duration:     int(t.Duration / 1000),
			PlayCount:    int(t.PlayCount),
			LastPlayed:   lastPlayed,
			DateAdded:    dateAdded,
			Size:         int(t.Size),
			Type:         mediaTypeString(t.MediaType),
			BookmarkTime: int(t.BookmarkTime),
		})
	}
	return tracks, nil
}

func (a *App) GetIPodPlaylists() ([]IPodPlaylistInfo, error) {
	info, err := ipod.Detect()
	if err != nil || info == nil {
		return nil, err
	}

	dev, err := ipod.OpenDevice(info)
	if err != nil {
		return nil, err
	}

	var playlists []IPodPlaylistInfo
	for i, pl := range dev.DB.Playlists {
		if pl.IsMaster {
			continue
		}
		var trackIDs []string
		for _, t := range pl.Tracks {
			trackIDs = append(trackIDs, fmt.Sprintf("ipod-%d", t.UniqueID))
		}
		playlists = append(playlists, IPodPlaylistInfo{
			ID:       fmt.Sprintf("ipl-%d", i),
			Name:     pl.Name,
			TrackIDs: trackIDs,
		})
	}
	return playlists, nil
}

func (a *App) GetIPodStorage() (*StorageBreakdown, error) {
	info, err := ipod.Detect()
	if err != nil || info == nil {
		return nil, err
	}

	dev, err := ipod.OpenDevice(info)
	if err != nil {
		return nil, err
	}

	breakdown := &StorageBreakdown{
		Free:  info.FreeSpace,
		Total: info.TotalSpace,
	}

	for _, t := range dev.DB.Tracks {
		size := int64(t.Size)
		switch mediaTypeString(t.MediaType) {
		case "audiobook":
			breakdown.Audiobooks += size
		case "podcast":
			breakdown.Podcasts += size
		default:
			breakdown.Music += size
		}
	}

	used := info.TotalSpace - info.FreeSpace
	contentSize := breakdown.Music + breakdown.Audiobooks + breakdown.Podcasts
	breakdown.Other = used - contentSize
	if breakdown.Other < 0 {
		breakdown.Other = 0
	}

	return breakdown, nil
}

func (a *App) BrowseDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Destination Folder",
	})
}

func (a *App) CopyTracksToComputer(trackIDs []string, destDir string) error {
	info, err := ipod.Detect()
	if err != nil || info == nil {
		return fmt.Errorf("no iPod found")
	}

	dev, err := ipod.OpenDevice(info)
	if err != nil {
		return err
	}

	idSet := make(map[string]bool, len(trackIDs))
	for _, id := range trackIDs {
		idSet[id] = true
	}

	var tracks []*itunesdb.Track
	for _, t := range dev.DB.Tracks {
		if idSet[fmt.Sprintf("ipod-%d", t.UniqueID)] {
			tracks = append(tracks, t)
		}
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	total := len(tracks)
	go func() {
		for i, t := range tracks {
			srcPath := ipod.FromiPodPath(info.MountPoint, t.Path)

			ext := filepath.Ext(srcPath)
			artist := sanitizeFilename(t.Artist)
			title := sanitizeFilename(t.Title)
			destName := fmt.Sprintf("%s - %s%s", artist, title, ext)
			destPath := filepath.Join(destDir, destName)

			runtime.EventsEmit(a.ctx, "copy:progress", map[string]interface{}{
				"current":     i,
				"total":       total,
				"currentFile": t.Title,
			})

			if err := copyFile(srcPath, destPath); err != nil {
				runtime.EventsEmit(a.ctx, "copy:error", err.Error())
				return
			}
		}
		runtime.EventsEmit(a.ctx, "copy:progress", map[string]interface{}{
			"current":     total,
			"total":       total,
			"currentFile": "",
		})
		runtime.EventsEmit(a.ctx, "copy:done", nil)
	}()
	return nil
}

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(s)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
