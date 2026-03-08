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
	cfg        *config.Config
	subClient  *subsonic.Client
	absClient  *audiobookshelf.Client
	cancelSync context.CancelFunc
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}
	a.cfg = cfg
}

func (a *App) shutdown(ctx context.Context) {
	if a.cfg != nil {
		_ = a.cfg.Save()
	}
}

func (a *App) GetConfig() *config.Config {
	return a.cfg
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
	a.cfg.Subsonic.ServerURL = serverURL
	a.cfg.Subsonic.Username = username
	a.cfg.Subsonic.Password = password
	a.subClient = subsonic.NewClient(serverURL, username, password)
	return a.cfg.Save()
}

func (a *App) SaveABSConfig(serverURL, token string) error {
	a.cfg.ABS.ServerURL = serverURL
	a.cfg.ABS.Token = token
	a.absClient = audiobookshelf.NewClient(serverURL, token)
	return a.cfg.Save()
}

func (a *App) TestSubsonicConnection() error {
	if a.subClient == nil {
		a.subClient = subsonic.NewClient(
			a.cfg.Subsonic.ServerURL,
			a.cfg.Subsonic.Username,
			a.cfg.Subsonic.Password,
		)
	}
	return a.subClient.Ping()
}

func (a *App) TestABSConnection() error {
	if a.absClient == nil {
		a.absClient = audiobookshelf.NewClient(a.cfg.ABS.ServerURL, a.cfg.ABS.Token)
	}
	return a.absClient.Ping()
}

func (a *App) DetectIPod() (*ipod.DeviceInfo, error) {
	return ipod.Detect()
}

func (a *App) EjectIPod() error {
	info, err := ipod.Detect()
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("no iPod connected")
	}
	return ipod.Eject(info.MountPoint)
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

func (a *App) GetExclusions() config.Exclusions {
	return a.cfg.Exclusions
}

func (a *App) SetExclusions(exclusions config.Exclusions) error {
	a.cfg.Exclusions = exclusions
	return a.cfg.Save()
}

func (a *App) GetSyncSettings() config.SyncSettings {
	return a.cfg.SyncSettings
}

func (a *App) SaveSyncSettings(settings config.SyncSettings) error {
	a.cfg.SyncSettings = settings
	return a.cfg.Save()
}

func (a *App) newSyncEngine() *sync.Engine {
	return sync.NewEngine(a.cfg, a.subClient, a.absClient)
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
			// Play Counts timestamps from iPod firmware are in local time, not UTC;
			// correct using the timezone offset stored in the iTunesDB header
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
