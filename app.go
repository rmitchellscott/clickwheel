package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/config"
	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
	"clickwheel/internal/restore"
	"clickwheel/internal/secrets"
	"clickwheel/internal/subsonic"
	"clickwheel/internal/sync"
)

type App struct {
	ctx            context.Context
	host           *config.HostConfig
	device         *config.DeviceConfig
	secrets        *secrets.Store
	subClient      *subsonic.Client
	absClient      *audiobookshelf.Client
	cancelSync     context.CancelFunc
	cancelRestore  context.CancelFunc
	connectedIPods []*ipod.DeviceInfo
	syncEngine     *sync.Engine
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

	a.secrets = secrets.NewStore()
	if pw, err := a.secrets.Get("subsonic-password"); err == nil {
		a.host.Subsonic.Password = pw
	}
	if tok, err := a.secrets.Get("abs-token"); err == nil {
		a.host.ABS.Token = tok
	}

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

func (a *App) HasSubsonicPassword() bool {
	return a.host.Subsonic.Password != ""
}

func (a *App) HasABSToken() bool {
	return a.host.ABS.Token != ""
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
	if password != "" && password != "••••••••" {
		a.host.Subsonic.Password = password
		if err := a.secrets.Set("subsonic-password", password); err != nil {
			return fmt.Errorf("save password to keychain: %w", err)
		}
	}
	a.subClient = subsonic.NewClient(serverURL, username, a.host.Subsonic.Password)
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
	if token != "" && token != "••••••••" {
		a.host.ABS.Token = token
		if err := a.secrets.Set("abs-token", token); err != nil {
			return fmt.Errorf("save token to keychain: %w", err)
		}
	}
	a.absClient = audiobookshelf.NewClient(serverURL, a.host.ABS.Token)
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

func (a *App) DetectIPods() ([]*ipod.DeviceInfo, error) {
	all, err := ipod.DetectAll()
	if err != nil {
		return nil, err
	}

	for _, info := range all {
		if pending := restore.ReadPendingDeviceName(info.MountPoint); pending != "" {
			log.Printf("[detect] found pending device name: %q at %s", pending, info.MountPoint)
			info.Name = pending
			a.applyPendingName(info, pending)
			restore.ClearPendingDeviceName(info.MountPoint)
		}

		serial := info.SerialNumber
		if serial == "" {
			serial = info.FirewireGUID
		}
		log.Printf("[detect] resolving device ID for %s: serial=%q fwGuid=%q resolved=%q", info.MountPoint, info.SerialNumber, info.FirewireGUID, serial)
		deviceID, err := config.ResolveDeviceID(info.MountPoint, serial)
		if err != nil {
			log.Printf("[detect] failed to resolve device ID for %s: %v", info.MountPoint, err)
			continue
		}
		info.DeviceID = deviceID

		devCfg, err := config.LoadDeviceConfig(info.MountPoint, deviceID)
		if err != nil {
			log.Printf("[detect] failed to load device config for %s: %v", deviceID, err)
			devCfg = config.DefaultDevice(deviceID)
		}
		devCfg.Servers.SubsonicURL = a.host.Subsonic.ServerURL
		devCfg.Servers.ABSURL = a.host.ABS.ServerURL

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

		if a.device == nil || a.device.DeviceID == deviceID {
			a.device = devCfg
			a.host.LastDeviceID = deviceID
		}
	}

	a.connectedIPods = all

	if len(all) == 1 {
		a.device, _ = config.LoadDeviceConfig(all[0].MountPoint, all[0].DeviceID)
		if a.device != nil {
			a.device.Servers.SubsonicURL = a.host.Subsonic.ServerURL
			a.device.Servers.ABSURL = a.host.ABS.ServerURL
		}
		a.host.LastDeviceID = all[0].DeviceID
	}

	_ = a.host.Save()
	return all, nil
}

func (a *App) DetectIPod() (*ipod.DeviceInfo, error) {
	all, err := a.DetectIPods()
	if err != nil {
		return nil, err
	}
	return a.activeIPodInfo(all), nil
}

func (a *App) SetActiveIPod(deviceID string) error {
	log.Printf("[SetActiveIPod] called with deviceID=%q, %d connected iPods", deviceID, len(a.connectedIPods))
	for i, info := range a.connectedIPods {
		log.Printf("[SetActiveIPod] connectedIPods[%d]: DeviceID=%q Name=%q Mount=%q", i, info.DeviceID, info.Name, info.MountPoint)
	}
	if a.cancelSync != nil {
		return fmt.Errorf("cannot switch devices while sync is in progress")
	}
	for _, info := range a.connectedIPods {
		if info.DeviceID == deviceID {
			log.Printf("[SetActiveIPod] matched device %q at %s", info.Name, info.MountPoint)
			devCfg, err := config.LoadDeviceConfig(info.MountPoint, deviceID)
			if err != nil {
				log.Printf("[SetActiveIPod] LoadDeviceConfig error: %v", err)
				return err
			}
			devCfg.Servers.SubsonicURL = a.host.Subsonic.ServerURL
			devCfg.Servers.ABSURL = a.host.ABS.ServerURL
			a.device = devCfg
			a.host.LastDeviceID = deviceID
			_ = a.host.Save()
			log.Printf("[SetActiveIPod] switched to %q", deviceID)
			return nil
		}
	}
	log.Printf("[SetActiveIPod] no match found for %q", deviceID)
	return fmt.Errorf("device %s is not currently connected", deviceID)
}

func (a *App) activeIPodInfo(list []*ipod.DeviceInfo) *ipod.DeviceInfo {
	if list == nil {
		list = a.connectedIPods
	}
	if a.device == nil {
		if len(list) > 0 {
			return list[0]
		}
		return nil
	}
	for _, info := range list {
		if info.DeviceID == a.device.DeviceID {
			return info
		}
	}
	if len(list) > 0 {
		return list[0]
	}
	return nil
}

func (a *App) EjectIPod() error {
	info := a.activeIPodInfo(nil)
	if info == nil {
		return fmt.Errorf("no iPod connected")
	}
	if a.device != nil {
		_ = a.saveDevice()
	}
	mountPoint := info.MountPoint
	err := ipod.Eject(mountPoint)
	if err != nil {
		return err
	}

	var remaining []*ipod.DeviceInfo
	for _, di := range a.connectedIPods {
		if di.MountPoint != mountPoint {
			remaining = append(remaining, di)
		}
	}
	a.connectedIPods = remaining

	if len(remaining) > 0 {
		_ = a.SetActiveIPod(remaining[0].DeviceID)
	}
	return nil
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

func (a *App) ForgetDevice(deviceID string) error {
	if a.cancelSync != nil {
		return fmt.Errorf("cannot forget device while sync is in progress")
	}

	for _, info := range a.connectedIPods {
		if info.DeviceID == deviceID {
			return fmt.Errorf("cannot forget a currently connected device — eject it first")
		}
	}

	a.host.RemoveKnownDevice(deviceID)

	if dir, err := config.DeviceBackupDir(deviceID); err == nil {
		_ = os.RemoveAll(dir)
	}

	if a.host.LastDeviceID == deviceID {
		if len(a.host.KnownDevices) > 0 {
			a.host.LastDeviceID = a.host.KnownDevices[0].DeviceID
		} else {
			a.host.LastDeviceID = ""
		}
	}

	if a.device != nil && a.device.DeviceID == deviceID {
		if a.host.LastDeviceID != "" {
			if devCfg, err := config.LoadDeviceFromBackup(a.host.LastDeviceID); err == nil {
				a.device = devCfg
			} else {
				a.device = nil
			}
		} else {
			a.device = nil
		}
	}

	return a.host.Save()
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
	info := a.activeIPodInfo(nil)
	if info == nil {
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
	mountPoint := ""
	if info := a.activeIPodInfo(nil); info != nil {
		mountPoint = info.MountPoint
	}
	return sync.NewEngine(a.host, a.device, a.subClient, a.absClient, mountPoint)
}

func (a *App) PreviewSync() (*sync.PlanSummary, error) {
	e := a.newSyncEngine()
	summary, err := e.Preview(a.ctx)
	if err != nil {
		return nil, err
	}
	a.syncEngine = e
	return summary, nil
}

func (a *App) StartSync() error {
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelSync = cancel
	e := a.syncEngine
	if e == nil {
		e = a.newSyncEngine()
	}
	a.syncEngine = nil
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
	Format       string `json:"format"`
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
	info := a.activeIPodInfo(nil)
	if info == nil {
		return nil, nil
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

		format := t.FiletypeKey
		if format == "m4a" && t.Duration > 0 {
			kbps := t.Size * 8 / t.Duration
			if kbps > 500 {
				format = "alac"
			}
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
			Format:       format,
			BookmarkTime: int(t.BookmarkTime),
		})
	}
	return tracks, nil
}

func (a *App) GetIPodPlaylists() ([]IPodPlaylistInfo, error) {
	info := a.activeIPodInfo(nil)
	if info == nil {
		return nil, nil
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
		trackIDs := make([]string, 0, len(pl.Tracks))
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
	info := a.activeIPodInfo(nil)
	if info == nil {
		return nil, nil
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
	info := a.activeIPodInfo(nil)
	if info == nil {
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
		_, err := ipod.ExportTracks(ipod.ExportOptions{
			Tracks:         tracks,
			Playlists:      dev.DB.Playlists,
			MountPoint:     info.MountPoint,
			DestDir:        destDir,
			EmbedArtwork:   true,
			ExportPlaylist: true,
			OnProgress: func(current, total int, title string) {
				runtime.EventsEmit(a.ctx, "copy:progress", map[string]interface{}{
					"current":     current,
					"total":       total,
					"currentFile": title,
				})
			},
		})
		if err != nil {
			runtime.EventsEmit(a.ctx, "copy:error", err.Error())
			return
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

func (a *App) DetectUSBIPod() (*restore.USBDeviceInfo, error) {
	devs, err := restore.EnumerateIPods()
	if err != nil {
		return nil, err
	}
	if len(devs) == 0 {
		return nil, nil
	}
	return &devs[0], nil
}

func (a *App) DetectUSBIPods() []restore.USBIPod {
	ipods, err := restore.DetectUSBIPods()
	if err != nil {
		log.Printf("[restore] DetectUSBIPods error: %v", err)
		return nil
	}
	for i, ip := range ipods {
		log.Printf("[restore] USB iPod[%d]: %s mode=%s disk=%s restorable=%v", i, ip.Model.Name, ip.Mode, ip.DiskPath, ip.Model.Restorable)
	}
	return ipods
}



func (a *App) MountIPod(rawDiskPath string) error {
	log.Printf("[mount] MountIPod called with diskPath=%q", rawDiskPath)
	if rawDiskPath == "" {
		ipods, err := restore.DetectUSBIPods()
		if err != nil || len(ipods) == 0 {
			log.Printf("[mount] no iPod found to mount")
			return fmt.Errorf("no iPod found to mount")
		}
		rawDiskPath = ipods[0].DiskPath
		log.Printf("[mount] resolved diskPath=%q", rawDiskPath)
		if rawDiskPath == "" {
			return fmt.Errorf("could not determine disk path")
		}
	}
	err := restore.MountDataPartition(rawDiskPath)
	if err != nil {
		log.Printf("[mount] failed: %v", err)
	} else {
		log.Printf("[mount] success")
	}
	return err
}

func (a *App) GetAvailableFirmware(model string) []restore.IPSWEntry {
	return restore.FindFirmware(model)
}

func (a *App) GetIPSWCatalog() []restore.IPSWEntry {
	return restore.GetCatalog()
}

func (a *App) GetRecommendedFirmware() []restore.FirmwareMatch {
	info := a.activeIPodInfo(nil)
	if info != nil {
		matches := restore.MatchFirmware(info.Family, info.Generation, info.Model)
		log.Printf("[restore] MatchFirmware (mounted): %d matches for %s %s", len(matches), info.Family, info.Generation)
		return matches
	}

	usbIPods, _ := restore.DetectUSBIPods()
	if len(usbIPods) > 0 && usbIPods[0].Model != nil {
		m := usbIPods[0].Model
		matches := restore.MatchFirmware(m.Family, m.Generation, "")
		log.Printf("[restore] MatchFirmware (USB): %d matches for %s %s", len(matches), m.Family, m.Generation)
		return matches
	}

	return nil
}


func (a *App) StartRestore(ipswIndex int, deviceName string, rawDiskPath string, password string) error {
	restore.SetPassword(password)

	catalog := restore.GetCatalog()
	if ipswIndex < 0 || ipswIndex >= len(catalog) {
		return fmt.Errorf("invalid firmware index")
	}
	entry := catalog[ipswIndex]

	info := a.activeIPodInfo(nil)

	var model *restore.IPodModel
	var rawDisk string

	if info != nil {
		model = restore.ModelByFamilyGeneration(info.Family, info.Generation)
		if rd, err := restore.RawDiskPath(info.MountPoint); err == nil {
			rawDisk = rd
		}
	} else {
		usbIPods, _ := restore.DetectUSBIPods()
		for _, u := range usbIPods {
			if u.DiskPath != "" && u.Model != nil {
				model = u.Model
				rawDisk = u.DiskPath
				log.Printf("[restore] Found iPod via USB: %s at %s", model.Name, rawDisk)
				break
			}
		}
	}

	if model == nil {
		model = restore.ModelForFirmwareIndex(ipswIndex)
	}

	if rawDiskPath != "" {
		rawDisk = rawDiskPath
	}

	if rawDisk == "" {
		return fmt.Errorf("no iPod detected")
	}

	if model == nil {
		return fmt.Errorf("could not determine iPod model for selected firmware")
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelRestore = cancel

	engine := restore.NewRestoreEngine(entry, model, deviceName, func(progress restore.RestoreProgress) {
		runtime.EventsEmit(a.ctx, "restore:progress", progress)
	})
	if rawDisk != "" {
		engine.SetRawDisk(rawDisk)
	}

	go func() {
		defer restore.ClearPassword()
		defer func() { a.cancelRestore = nil }()
		err := engine.Run(ctx)
		if err != nil {
			if ctx.Err() != nil {
				runtime.EventsEmit(a.ctx, "restore:error", "Restore cancelled")
			} else {
				runtime.EventsEmit(a.ctx, "restore:error", err.Error())
			}
			return
		}
		if a.device != nil && deviceName != "" {
			a.device.DeviceName = deviceName
			_ = a.saveDevice()
		}
		if deviceName != "" {
			log.Printf("[restore] mounting data partition for name write: %s", rawDisk)
			if err := restore.MountDataPartition(rawDisk); err != nil {
				log.Printf("[restore] mount failed: %v", err)
			} else {
				mp, err := restore.FindMountPoint(rawDisk)
				if err != nil {
					log.Printf("[restore] find mount point failed: %v", err)
				} else {
					log.Printf("[restore] writing device name %q to %s", deviceName, mp)
					restore.WriteDeviceName(mp, deviceName)
				}
			}
		}
		runtime.EventsEmit(a.ctx, "restore:done", nil)
	}()
	return nil
}

func (a *App) CancelRestore() {
	if a.cancelRestore != nil {
		a.cancelRestore()
	}
}

func (a *App) applyPendingName(info *ipod.DeviceInfo, name string) {
	dev, err := ipod.OpenDevice(info)
	if err != nil {
		return
	}
	for _, pl := range dev.DB.Playlists {
		if pl.IsMaster {
			pl.Name = name
			break
		}
	}
	_ = dev.Save()
	_ = restore.RenameVolume(info.MountPoint, name)
}

func (a *App) ApproveSysInfoRepair(proposedSysInfo string) error {
	info := a.activeIPodInfo(nil)
	if info == nil {
		return fmt.Errorf("no iPod connected")
	}

	sysInfoPath := fmt.Sprintf("%s/iPod_Control/Device/SysInfo", info.MountPoint)
	if err := os.MkdirAll(fmt.Sprintf("%s/iPod_Control/Device", info.MountPoint), 0755); err != nil {
		return fmt.Errorf("create Device dir: %w", err)
	}
	if err := os.WriteFile(sysInfoPath, []byte(proposedSysInfo), 0644); err != nil {
		return fmt.Errorf("write SysInfo: %w", err)
	}

	log.Printf("[detect] SysInfo repaired at %s", sysInfoPath)
	return nil
}

func (a *App) RenameIPod(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	info := a.activeIPodInfo(nil)
	if info == nil {
		return fmt.Errorf("no iPod connected")
	}

	dev, err := ipod.OpenDevice(info)
	if err != nil {
		return fmt.Errorf("open device: %w", err)
	}

	for _, pl := range dev.DB.Playlists {
		if pl.IsMaster {
			pl.Name = name
			break
		}
	}

	if err := dev.Save(); err != nil {
		return fmt.Errorf("save iTunesDB: %w", err)
	}

	if err := restore.RenameVolume(info.MountPoint, name); err != nil {
		fmt.Fprintf(os.Stderr, "volume rename (will apply after eject): %v\n", err)
	}

	if a.device != nil {
		a.device.DeviceName = name
		_ = a.saveDevice()
	}

	return nil
}

