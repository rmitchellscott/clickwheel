package sync

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/config"
	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
	"clickwheel/internal/subsonic"
)

type Progress struct {
	Phase   string  `json:"phase"`
	Current int     `json:"current"`
	Total   int     `json:"total"`
	Message string  `json:"message"`
	Percent float64 `json:"percent"`
	ETA     string  `json:"eta,omitempty"`
}

func formatETA(d time.Duration) string {
	if d < time.Minute {
		return "Less than a minute"
	}
	mins := int(d.Minutes())
	if mins < 60 {
		return fmt.Sprintf("About %d min", mins)
	}
	return fmt.Sprintf("About %dh %dm", mins/60, mins%60)
}

type transferSample struct {
	bytes    int64
	duration time.Duration
}

type etaTracker struct {
	historicalRate float64
	totalBytes    int64
	completed     int64
	samples       []transferSample
	maxSamples    int
	start         time.Time
}

func newETATracker(historicalRate float64, totalBytes int64) *etaTracker {
	return &etaTracker{
		historicalRate: historicalRate,
		totalBytes:    totalBytes,
		maxSamples:    10,
		start:         time.Now(),
	}
}

func (t *etaTracker) record(bytes int64, d time.Duration) {
	t.completed += bytes
	t.samples = append(t.samples, transferSample{bytes, d})
	if len(t.samples) > t.maxSamples {
		t.samples = t.samples[1:]
	}
}

func (t *etaTracker) eta() string {
	remaining := t.totalBytes - t.completed
	if remaining <= 0 {
		return ""
	}

	var windowBytes int64
	var windowDur time.Duration
	for _, s := range t.samples {
		windowBytes += s.bytes
		windowDur += s.duration
	}

	var sessionRate float64
	if windowDur > 0 {
		sessionRate = float64(windowBytes) / windowDur.Seconds()
	}

	var rate float64
	if sessionRate > 0 && t.historicalRate > 0 {
		alpha := math.Min(1.0, float64(t.completed)/math.Max(1, float64(t.totalBytes)*0.1))
		rate = alpha*sessionRate + (1-alpha)*t.historicalRate
	} else if sessionRate > 0 {
		rate = sessionRate
	} else if t.historicalRate > 0 {
		rate = t.historicalRate
	} else {
		return ""
	}

	d := time.Duration(float64(remaining) / rate * float64(time.Second))
	return formatETA(d)
}

func (t *etaTracker) finalRate() float64 {
	elapsed := time.Since(t.start)
	if elapsed <= 0 || t.completed <= 0 {
		return 0
	}
	return float64(t.completed) / elapsed.Seconds()
}

type ProgressFunc func(Progress)

type Engine struct {
	host   *config.HostConfig
	device *config.DeviceConfig
	sub    *subsonic.Client
	abs    *audiobookshelf.Client
}

func NewEngine(host *config.HostConfig, device *config.DeviceConfig, sub *subsonic.Client, abs *audiobookshelf.Client) *Engine {
	return &Engine{host: host, device: device, sub: sub, abs: abs}
}

type PlanSummary struct {
	AddTracks        []PlanSummaryItem `json:"addTracks"`
	AddBooks         []PlanSummaryItem `json:"addBooks"`
	AddPodcasts      []PlanSummaryItem `json:"addPodcasts"`
	RemoveTracks     []PlanSummaryItem `json:"removeTracks"`
	RemoveBooks      []PlanSummaryItem `json:"removeBooks"`
	RemovePodcasts   []PlanSummaryItem `json:"removePodcasts"`
	Playlists        []string          `json:"playlists"`
	PlaylistsChanged []string          `json:"playlistsChanged"`
	PlaysToSync      int               `json:"playsToSync"`
	BooksToIPod      []string          `json:"booksToIPod"`
	BooksFromIPod    []string          `json:"booksFromIPod"`
	PodcastsToIPod   []string          `json:"podcastsToIPod"`
	PodcastsFromIPod []string          `json:"podcastsFromIPod"`
}

type PlanSummaryItem struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Size   int64  `json:"size"`
}

func (e *Engine) Preview(ctx context.Context) (*PlanSummary, error) {
	info, err := ipod.Detect()
	if err != nil {
		return nil, fmt.Errorf("detecting iPod: %w", err)
	}
	if info == nil {
		return nil, fmt.Errorf("no iPod found")
	}

	dev, err := ipod.OpenDevice(info)
	if err != nil {
		return nil, fmt.Errorf("opening device: %w", err)
	}

	purgeUnmanagedTracks(dev)

	plan, err := BuildPlan(ctx, e.device, e.sub, e.abs, dev)
	if err != nil {
		return nil, fmt.Errorf("building sync plan: %w", err)
	}

	trackBySource := make(map[string]*itunesdb.Track, len(dev.DB.Tracks))
	for _, t := range dev.DB.Tracks {
		if t.SourceID != "" {
			trackBySource[t.SourceID] = t
		}
	}

	summary := &PlanSummary{}
	for _, id := range plan.RemoveTracks {
		item := PlanSummaryItem{Title: id}
		if t := trackBySource[id]; t != nil {
			item = PlanSummaryItem{Title: t.Title, Artist: t.Artist, Size: int64(t.Size)}
		}
		summary.RemoveTracks = append(summary.RemoveTracks, item)
	}
	for _, id := range plan.RemoveBooks {
		item := PlanSummaryItem{Title: id}
		if t := trackBySource[id]; t != nil {
			item = PlanSummaryItem{Title: t.Title, Artist: t.Artist, Size: int64(t.Size)}
		}
		summary.RemoveBooks = append(summary.RemoveBooks, item)
	}
	for _, id := range plan.RemovePodcasts {
		item := PlanSummaryItem{Title: id}
		if t := trackBySource[id]; t != nil {
			item = PlanSummaryItem{Title: t.Title, Artist: t.Artist, Size: int64(t.Size)}
		}
		summary.RemovePodcasts = append(summary.RemovePodcasts, item)
	}

	for _, t := range plan.AddTracks {
		summary.AddTracks = append(summary.AddTracks, PlanSummaryItem{
			Title: t.Title, Artist: t.Artist, Size: t.Size,
		})
	}
	for _, b := range plan.AddBooks {
		summary.AddBooks = append(summary.AddBooks, PlanSummaryItem{
			Title: b.Title, Artist: b.Author, Size: b.Size,
		})
	}
	for _, p := range plan.AddPodcasts {
		summary.AddPodcasts = append(summary.AddPodcasts, PlanSummaryItem{
			Title: p.Title, Artist: p.ShowName, Size: p.Size,
		})
	}
	ipodPlaylists := make(map[string][]string)
	for _, pl := range dev.DB.Playlists {
		if pl.IsMaster {
			continue
		}
		var ids []string
		for _, t := range pl.Tracks {
			ids = append(ids, t.SourceID)
		}
		ipodPlaylists[pl.Name] = ids
	}

	for _, p := range plan.Playlists {
		summary.Playlists = append(summary.Playlists, p.Name)
		existing, ok := ipodPlaylists[p.Name]
		if !ok {
			continue
		}
		if !slices.Equal(p.TrackIDs, existing) {
			summary.PlaylistsChanged = append(summary.PlaylistsChanged, p.Name)
		}
	}

	if e.device.SyncSettings.SyncPlayCounts && e.sub != nil {
		for _, track := range dev.DB.Tracks {
			if track.SourceID == "" || track.MediaType != itunesdb.MediaTypeMusic {
				continue
			}
			prev := e.device.SyncState.TrackPlayCounts[track.SourceID]
			if delta := int(track.PlayCount) - prev.PlayCount; delta > 0 {
				summary.PlaysToSync += delta
			}
		}
	}

	if e.device.SyncSettings.SyncBookPosition && e.abs != nil {
		type splitPreviewGroup struct {
			bookTitle string
			tracks    []*itunesdb.Track
			parts     []config.BookSplitPart
		}
		splitGroups := make(map[string]*splitPreviewGroup)
		seenBooks := make(map[string]bool)

		for _, track := range dev.DB.Tracks {
			if track.SourceID == "" || track.MediaType != itunesdb.MediaTypeAudiobook {
				continue
			}

			bookID, _, isSplit := splitBookSourceID(track.SourceID)
			if isSplit {
				g, ok := splitGroups[bookID]
				if !ok {
					info := e.device.SyncState.BookSplits[bookID]
					g = &splitPreviewGroup{bookTitle: track.Album, parts: info.Parts}
					splitGroups[bookID] = g
				}
				g.tracks = append(g.tracks, track)
				continue
			}

			progress, err := e.abs.GetProgress(track.SourceID)
			if err != nil {
				continue
			}

			ipodTime := float64(track.BookmarkTime) / 1000.0
			prev := e.device.SyncState.BookmarkStates[track.SourceID]

			absTime := 0.0
			if progress != nil {
				absTime = progress.CurrentTime
			}

			ipodChanged := math.Round(ipodTime) != math.Round(prev.CurrentTime)
			absChanged := progress != nil && math.Round(absTime) != math.Round(prev.CurrentTime) && progress.LastUpdate/1000 > prev.LastSync

			if !ipodChanged && !absChanged {
				continue
			}

			if ipodChanged && (!absChanged || ipodTime > absTime) {
				summary.BooksFromIPod = append(summary.BooksFromIPod, track.Title)
			} else if absChanged {
				summary.BooksToIPod = append(summary.BooksToIPod, track.Title)
			}
		}

		for bookID, g := range splitGroups {
			if seenBooks[bookID] || len(g.parts) == 0 {
				continue
			}
			seenBooks[bookID] = true

			progress, err := e.abs.GetProgress(bookID)
			if err != nil {
				continue
			}

			ipodGlobal := splitBookGlobalPosition(g.tracks, g.parts)
			prev := e.device.SyncState.BookmarkStates[bookID]

			absGlobal := 0.0
			if progress != nil {
				absGlobal = progress.CurrentTime
			}

			ipodChanged := math.Round(ipodGlobal) != math.Round(prev.CurrentTime)
			absChanged := progress != nil && math.Round(absGlobal) != math.Round(prev.CurrentTime) && progress.LastUpdate/1000 > prev.LastSync

			if !ipodChanged && !absChanged {
				continue
			}

			if ipodChanged && (!absChanged || ipodGlobal > absGlobal) {
				summary.BooksFromIPod = append(summary.BooksFromIPod, g.bookTitle)
			} else if absChanged {
				summary.BooksToIPod = append(summary.BooksToIPod, g.bookTitle)
			}
		}
	}

	if e.device.SyncSettings.SyncPodcastPosition && e.abs != nil {
		for _, track := range dev.DB.Tracks {
			if track.SourceID == "" || track.MediaType != itunesdb.MediaTypePodcast {
				continue
			}

			itemID, episodeID := splitPodcastSourceID(track.SourceID)
			if itemID == "" {
				continue
			}

			progress, err := e.abs.GetEpisodeProgress(itemID, episodeID)
			if err != nil {
				continue
			}

			ipodTime := float64(track.BookmarkTime) / 1000.0
			prev := e.device.SyncState.BookmarkStates[track.SourceID]

			absTime := 0.0
			if progress != nil {
				absTime = progress.CurrentTime
			}

			ipodChanged := math.Round(ipodTime) != math.Round(prev.CurrentTime)
			absChanged := progress != nil && math.Round(absTime) != math.Round(prev.CurrentTime) && progress.LastUpdate/1000 > prev.LastSync

			if !ipodChanged && !absChanged {
				continue
			}

			if ipodChanged && (!absChanged || ipodTime > absTime) {
				summary.PodcastsFromIPod = append(summary.PodcastsFromIPod, track.Title)
			} else if absChanged {
				summary.PodcastsToIPod = append(summary.PodcastsToIPod, track.Title)
			}
		}
	}

	return summary, nil
}

func (e *Engine) Run(ctx context.Context, onProgress ProgressFunc) error {
	onProgress(Progress{Phase: "detect", Message: "Detecting iPod..."})

	info, err := ipod.Detect()
	if err != nil {
		return fmt.Errorf("detecting iPod: %w", err)
	}
	if info == nil {
		return fmt.Errorf("no iPod found")
	}

	dev, err := ipod.OpenDevice(info)
	if err != nil {
		return fmt.Errorf("opening device: %w", err)
	}

	purgeUnmanagedTracks(dev)
	log.Printf("[sync] %d managed tracks on iPod after purge", len(dev.DB.Tracks))

	if e.device.SyncSettings.SyncPlayCounts {
		if err := e.syncPlayCounts(ctx, dev, onProgress); err != nil {
			return fmt.Errorf("syncing play counts: %w", err)
		}
	}

	if e.device.SyncSettings.SyncBookPosition {
		if err := e.syncBookmarks(ctx, dev, onProgress); err != nil {
			return fmt.Errorf("syncing bookmarks: %w", err)
		}
	}

	if e.device.SyncSettings.SyncPodcastPosition {
		if err := e.syncPodcastProgress(ctx, dev, onProgress); err != nil {
			return fmt.Errorf("syncing podcast progress: %w", err)
		}
	}

	plan, err := e.buildPlan(ctx, dev, onProgress)
	if err != nil {
		return fmt.Errorf("building sync plan: %w", err)
	}

	log.Printf("[sync] plan: add=%d remove=%d playlists=%d", len(plan.AddTracks)+len(plan.AddBooks)+len(plan.AddPodcasts), len(plan.RemoveTracks)+len(plan.RemoveBooks)+len(plan.RemovePodcasts), len(plan.Playlists))

	if err := e.executePlan(ctx, dev, plan, onProgress); err != nil {
		return fmt.Errorf("executing sync plan: %w", err)
	}

	buildPlaylists(dev, plan)
	log.Printf("[sync] built %d playlists, total %d playlists in DB", len(plan.Playlists), len(dev.DB.Playlists))

	syncArtwork(dev, e.sub, onProgress)

	onProgress(Progress{Phase: "cleanup", Message: "Cleaning up orphaned files..."})
	knownPaths := make(map[string]bool)
	for _, t := range dev.DB.Tracks {
		if t.Path != "" {
			knownPaths[t.Path] = true
		}
	}
	ipod.CleanOrphans(dev.Info.MountPoint, knownPaths)

	onProgress(Progress{Phase: "save", Message: "Writing iTunesDB..."})
	if err := dev.Save(); err != nil {
		return fmt.Errorf("saving iTunesDB: %w", err)
	}

	if err := e.device.SaveBoth(dev.Info.MountPoint); err != nil {
		return fmt.Errorf("saving device config: %w", err)
	}

	onProgress(Progress{Phase: "done", Message: "Sync complete!", Percent: 100})
	return nil
}

func (e *Engine) syncPlayCounts(ctx context.Context, dev *ipod.Device, onProgress ProgressFunc) error {
	if e.sub == nil {
		return nil
	}
	onProgress(Progress{Phase: "scrobble", Message: "Syncing play counts..."})

	for _, track := range dev.DB.Tracks {
		if track.SourceID == "" || track.MediaType != itunesdb.MediaTypeMusic {
			continue
		}

		prev := e.device.SyncState.TrackPlayCounts[track.SourceID]
		delta := int(track.PlayCount) - prev.PlayCount

		for range delta {
			if err := e.sub.Scrobble(track.SourceID); err != nil {
				return err
			}
		}

		if delta > 0 {
			e.device.SyncState.TrackPlayCounts[track.SourceID] = config.TrackSyncState{
				PlayCount: int(track.PlayCount),
				LastSync:  time.Now().Unix(),
			}
		}
	}

	return nil
}

func (e *Engine) syncBookmarks(ctx context.Context, dev *ipod.Device, onProgress ProgressFunc) error {
	if e.abs == nil {
		return nil
	}
	onProgress(Progress{Phase: "bookmarks", Message: "Syncing audiobook progress..."})

	type splitGroup struct {
		tracks []*itunesdb.Track
		parts  []config.BookSplitPart
	}
	splitBooks := make(map[string]*splitGroup)

	for _, track := range dev.DB.Tracks {
		if track.SourceID == "" || track.MediaType != itunesdb.MediaTypeAudiobook {
			continue
		}

		bookID, _, isSplit := splitBookSourceID(track.SourceID)
		if !isSplit {
			e.syncSingleBookmark(track)
			continue
		}

		g, ok := splitBooks[bookID]
		if !ok {
			info := e.device.SyncState.BookSplits[bookID]
			g = &splitGroup{parts: info.Parts}
			splitBooks[bookID] = g
		}
		g.tracks = append(g.tracks, track)
	}

	for bookID, g := range splitBooks {
		parts := g.parts
		if len(parts) == 0 {
			log.Printf("[bookmarks] no split info for %s, skipping", bookID)
			continue
		}

		progress, err := e.abs.GetProgress(bookID)
		if err != nil {
			continue
		}

		ipodGlobal := splitBookGlobalPosition(g.tracks, parts)
		prev := e.device.SyncState.BookmarkStates[bookID]

		absGlobal := 0.0
		absDuration := parts[len(parts)-1].EndSec
		if progress != nil {
			absGlobal = progress.CurrentTime
			absDuration = progress.Duration
		}

		ipodChanged := math.Round(ipodGlobal) != math.Round(prev.CurrentTime)
		absChanged := progress != nil && math.Round(absGlobal) != math.Round(prev.CurrentTime) && progress.LastUpdate/1000 > prev.LastSync

		if ipodChanged && (!absChanged || ipodGlobal > absGlobal) {
			_ = e.abs.UpdateProgress(bookID, ipodGlobal, absDuration)
		} else if absChanged {
			distributePositionToParts(g.tracks, parts, absGlobal)
		}

		e.device.SyncState.BookmarkStates[bookID] = config.PositionSyncState{
			CurrentTime: splitBookGlobalPosition(g.tracks, parts),
			LastSync:    time.Now().Unix(),
		}
	}

	return nil
}

func splitBookGlobalPosition(tracks []*itunesdb.Track, parts []config.BookSplitPart) float64 {
	var best float64
	for _, t := range tracks {
		if t.BookmarkTime == 0 {
			continue
		}
		_, idx, _ := splitBookSourceID(t.SourceID)
		if idx >= len(parts) {
			continue
		}
		global := parts[idx].StartSec + float64(t.BookmarkTime)/1000.0
		if global > best {
			best = global
		}
	}
	return best
}

func distributePositionToParts(tracks []*itunesdb.Track, parts []config.BookSplitPart, globalSec float64) {
	for _, t := range tracks {
		_, partIdx, _ := splitBookSourceID(t.SourceID)
		if partIdx >= len(parts) {
			continue
		}
		part := parts[partIdx]
		if globalSec >= part.StartSec && globalSec < part.EndSec {
			t.BookmarkTime = uint32((globalSec - part.StartSec) * 1000)
		} else if globalSec >= part.EndSec {
			t.BookmarkTime = uint32((part.EndSec - part.StartSec) * 1000)
		} else {
			t.BookmarkTime = 0
		}
	}
}

func (e *Engine) syncSingleBookmark(track *itunesdb.Track) {
	progress, err := e.abs.GetProgress(track.SourceID)
	if err != nil {
		return
	}

	ipodTime := float64(track.BookmarkTime) / 1000.0
	prev := e.device.SyncState.BookmarkStates[track.SourceID]

	if progress != nil {
		absTime := progress.CurrentTime

		ipodChanged := math.Round(ipodTime) != math.Round(prev.CurrentTime)
		absChanged := math.Round(absTime) != math.Round(prev.CurrentTime) && progress.LastUpdate/1000 > prev.LastSync

		if ipodChanged && (!absChanged || ipodTime > absTime) {
			_ = e.abs.UpdateProgress(track.SourceID, ipodTime, progress.Duration)
		} else if absChanged {
			track.BookmarkTime = uint32(absTime * 1000)
		}
	} else if ipodTime > 0 {
		_ = e.abs.UpdateProgress(track.SourceID, ipodTime, float64(track.Duration)/1000.0)
	}

	e.device.SyncState.BookmarkStates[track.SourceID] = config.PositionSyncState{
		CurrentTime: float64(track.BookmarkTime) / 1000.0,
		LastSync:    time.Now().Unix(),
	}
}

func splitPodcastSourceID(sourceID string) (itemID, episodeID string) {
	parts := strings.SplitN(sourceID, "|", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func (e *Engine) syncPodcastProgress(ctx context.Context, dev *ipod.Device, onProgress ProgressFunc) error {
	if e.abs == nil {
		return nil
	}
	onProgress(Progress{Phase: "bookmarks", Message: "Syncing podcast progress..."})

	for _, track := range dev.DB.Tracks {
		if track.SourceID == "" || track.MediaType != itunesdb.MediaTypePodcast {
			continue
		}

		itemID, episodeID := splitPodcastSourceID(track.SourceID)
		if itemID == "" {
			continue
		}

		progress, err := e.abs.GetEpisodeProgress(itemID, episodeID)
		if err != nil {
			continue
		}

		ipodTime := float64(track.BookmarkTime) / 1000.0
		prev := e.device.SyncState.BookmarkStates[track.SourceID]

		if progress != nil {
			absTime := progress.CurrentTime

			ipodChanged := math.Round(ipodTime) != math.Round(prev.CurrentTime)
			absChanged := math.Round(absTime) != math.Round(prev.CurrentTime) && progress.LastUpdate/1000 > prev.LastSync

			if ipodChanged && (!absChanged || ipodTime > absTime) {
				if e.device.SyncSettings.TwoWayPodcastSync {
					_ = e.abs.UpdateEpisodeProgress(itemID, episodeID, ipodTime, progress.Duration)
				}
			} else if absChanged {
				track.BookmarkTime = uint32(absTime * 1000)
			}
		} else if ipodTime > 0 && e.device.SyncSettings.TwoWayPodcastSync {
			_ = e.abs.UpdateEpisodeProgress(itemID, episodeID, ipodTime, float64(track.Duration)/1000.0)
		}

		e.device.SyncState.BookmarkStates[track.SourceID] = config.PositionSyncState{
			CurrentTime: float64(track.BookmarkTime) / 1000.0,
			LastSync:    time.Now().Unix(),
		}
	}

	return nil
}

func (e *Engine) buildPlan(ctx context.Context, dev *ipod.Device, onProgress ProgressFunc) (*SyncPlan, error) {
	onProgress(Progress{Phase: "plan", Message: "Building sync plan..."})
	return BuildPlan(ctx, e.device, e.sub, e.abs, dev)
}

func (e *Engine) executePlan(ctx context.Context, dev *ipod.Device, plan *SyncPlan, onProgress ProgressFunc) error {
	allRemovals := append(append(plan.RemoveTracks, plan.RemoveBooks...), plan.RemovePodcasts...)
	removedBaseBooks := make(map[string]bool)
	for _, id := range allRemovals {
		removed := dev.DB.RemoveTrack(id)
		if removed != nil && removed.Path != "" {
			absPath := ipod.FromiPodPath(dev.Info.MountPoint, removed.Path)
			os.Remove(absPath)
		}
		if baseID, _, isSplit := splitBookSourceID(id); isSplit {
			removedBaseBooks[baseID] = true
		}
	}
	for baseID := range removedBaseBooks {
		hasRemaining := false
		for _, t := range dev.DB.Tracks {
			if tb, _, ts := splitBookSourceID(t.SourceID); ts && tb == baseID {
				hasRemaining = true
				break
			}
		}
		if !hasRemaining {
			delete(e.device.SyncState.BookSplits, baseID)
		}
	}

	total := len(plan.AddTracks) + len(plan.AddBooks) + len(plan.AddPodcasts)
	var totalBytes int64
	for _, t := range plan.AddTracks {
		totalBytes += t.Size
	}
	for _, b := range plan.AddBooks {
		if b.Size > 0 {
			totalBytes += b.Size
		} else {
			totalBytes += int64(b.Duration * 64 * 1000 / 8)
		}
	}
	for _, p := range plan.AddPodcasts {
		totalBytes += p.Size
	}

	tracker := newETATracker(e.device.SyncState.TransferRate, totalBytes)

	format := e.device.SyncSettings.MusicFormat
	bitRate := e.device.SyncSettings.MusicBitRate

	if len(plan.AddTracks) > 0 {
		workers := runtime.NumCPU()
		if workers > 4 {
			workers = 4
		}
		if workers > len(plan.AddTracks) {
			workers = len(plan.AddTracks)
		}

		jobs := make(chan int, workers)
		results := make([]chan *preparedTrack, len(plan.AddTracks))
		for i := range results {
			results[i] = make(chan *preparedTrack, 1)
		}

		var wg sync.WaitGroup
		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for idx := range jobs {
					results[idx] <- DownloadAndTranscode(ctx, e.sub, plan.AddTracks[idx], format, bitRate)
				}
			}()
		}

		go func() {
			for i := range plan.AddTracks {
				if ctx.Err() != nil {
					break
				}
				jobs <- i
			}
			close(jobs)
			wg.Wait()
		}()

		lastDone := time.Now()
		for i, item := range plan.AddTracks {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			onProgress(Progress{
				Phase:   "transfer",
				Current: i + 1,
				Total:   total,
				Message: fmt.Sprintf("Transferring: %s", item.Title),
				Percent: float64(i+1) / float64(total) * 100,
				ETA:     tracker.eta(),
			})

			p := <-results[i]
			if err := InstallTrack(dev, p, format, bitRate); err != nil {
				return err
			}
			tracker.record(item.Size, time.Since(lastDone))
			lastDone = time.Now()
		}
	}

	for i, item := range plan.AddBooks {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		idx := len(plan.AddTracks) + i
		bookSizeEst := int64(item.Duration * 64 * 1000 / 8)

		onProgress(Progress{
			Phase:   "transfer",
			Current: idx + 1,
			Total:   total,
			Message: fmt.Sprintf("Downloading: %s", item.Title),
			Percent: float64(idx+1) / float64(total) * 100,
			ETA:     tracker.eta(),
		})

		itemStart := time.Now()
		if err := TransferBook(ctx, e.abs, dev, item, func(step string) {
			onProgress(Progress{
				Phase:   "transfer",
				Current: idx + 1,
				Total:   total,
				Message: fmt.Sprintf("%s: %s", step, item.Title),
				Percent: float64(idx+1) / float64(total) * 100,
				ETA:     tracker.eta(),
			})
		}); err != nil {
			return err
		}
		if item.SplitParts != nil {
			e.device.SyncState.BookSplits[item.SourceID] = config.BookSplitInfo{
				SplitHoursLimit: e.device.SyncSettings.SplitHoursLimit,
				Parts:           item.SplitParts,
			}
		}
		tracker.record(bookSizeEst, time.Since(itemStart))
	}

	for i, item := range plan.AddPodcasts {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		idx := len(plan.AddTracks) + len(plan.AddBooks) + i

		onProgress(Progress{
			Phase:   "transfer",
			Current: idx + 1,
			Total:   total,
			Message: fmt.Sprintf("Downloading: %s", item.Title),
			Percent: float64(idx+1) / float64(total) * 100,
			ETA:     tracker.eta(),
		})

		itemStart := time.Now()
		if err := TransferPodcastEpisode(ctx, e.abs, dev, item, func(step string) {
			onProgress(Progress{
				Phase:   "transfer",
				Current: idx + 1,
				Total:   total,
				Message: fmt.Sprintf("%s: %s", step, item.Title),
				Percent: float64(idx+1) / float64(total) * 100,
				ETA:     tracker.eta(),
			})
		}); err != nil {
			return err
		}
		tracker.record(item.Size, time.Since(itemStart))
	}

	if rate := tracker.finalRate(); rate > 0 {
		e.device.SyncState.TransferRate = rate
	}

	return nil
}

func buildPlaylists(dev *ipod.Device, plan *SyncPlan) {
	// Remove existing non-master playlists (rebuild from scratch)
	var kept []*itunesdb.Playlist
	for _, pl := range dev.DB.Playlists {
		if pl.IsMaster {
			kept = append(kept, pl)
		}
	}
	dev.DB.Playlists = kept

	// Build a sourceID → track pointer map
	trackBySource := make(map[string]*itunesdb.Track, len(dev.DB.Tracks))
	for _, t := range dev.DB.Tracks {
		if t.SourceID != "" {
			trackBySource[t.SourceID] = t
		}
	}

	for _, pp := range plan.Playlists {
		pl := &itunesdb.Playlist{Name: pp.Name}
		for _, sid := range pp.TrackIDs {
			if t, ok := trackBySource[sid]; ok {
				pl.Tracks = append(pl.Tracks, t)
			}
		}
		dev.DB.Playlists = append(dev.DB.Playlists, pl)
	}

	var podcastTracks []*itunesdb.Track
	for _, t := range dev.DB.Tracks {
		if t.MediaType == itunesdb.MediaTypePodcast {
			podcastTracks = append(podcastTracks, t)
		}
	}
	if len(podcastTracks) > 0 {
		podcastPL := &itunesdb.Playlist{
			Name:        "Podcasts",
			PodcastFlag: 1,
			Tracks:      podcastTracks,
		}
		dev.DB.Playlists = append(dev.DB.Playlists, podcastPL)
	}
}

func purgeUnmanagedTracks(dev *ipod.Device) {
	var managed []*itunesdb.Track
	for _, t := range dev.DB.Tracks {
		if t.SourceID != "" {
			managed = append(managed, t)
		} else if t.Path != "" {
			absPath := ipod.FromiPodPath(dev.Info.MountPoint, t.Path)
			os.Remove(absPath)
		}
	}
	dev.DB.Tracks = managed
	var nonPodcast []*itunesdb.Track
	for _, t := range managed {
		if t.MediaType != itunesdb.MediaTypePodcast {
			nonPodcast = append(nonPodcast, t)
		}
	}
	for _, pl := range dev.DB.Playlists {
		if pl.IsMaster {
			pl.Tracks = nonPodcast
			break
		}
	}
}
