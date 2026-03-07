package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/config"
	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
	"clickwheel/internal/navidrome"
)

type Progress struct {
	Phase   string  `json:"phase"`
	Current int     `json:"current"`
	Total   int     `json:"total"`
	Message string  `json:"message"`
	Percent float64 `json:"percent"`
}

type ProgressFunc func(Progress)

type Engine struct {
	cfg       *config.Config
	nav       *navidrome.Client
	abs       *audiobookshelf.Client
}

func NewEngine(cfg *config.Config, nav *navidrome.Client, abs *audiobookshelf.Client) *Engine {
	return &Engine{cfg: cfg, nav: nav, abs: abs}
}

type PlanSummary struct {
	AddTracks []PlanSummaryItem `json:"addTracks"`
	AddBooks  []PlanSummaryItem `json:"addBooks"`
	Remove    int               `json:"remove"`
	Playlists []string          `json:"playlists"`
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

	plan, err := BuildPlan(ctx, e.cfg, e.nav, e.abs, dev)
	if err != nil {
		return nil, fmt.Errorf("building sync plan: %w", err)
	}

	summary := &PlanSummary{
		Remove: len(plan.Remove),
	}

	for _, t := range plan.AddTracks {
		summary.AddTracks = append(summary.AddTracks, PlanSummaryItem{
			Title: t.Title, Artist: t.Artist, Size: t.Size,
		})
	}
	for _, b := range plan.AddBooks {
		summary.AddBooks = append(summary.AddBooks, PlanSummaryItem{
			Title: b.Title, Artist: b.Author,
		})
	}
	for _, p := range plan.Playlists {
		summary.Playlists = append(summary.Playlists, p.Name)
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

	if err := e.syncPlayCounts(ctx, dev, onProgress); err != nil {
		return fmt.Errorf("syncing play counts: %w", err)
	}

	if err := e.syncBookmarks(ctx, dev, onProgress); err != nil {
		return fmt.Errorf("syncing bookmarks: %w", err)
	}

	plan, err := e.buildPlan(ctx, dev, onProgress)
	if err != nil {
		return fmt.Errorf("building sync plan: %w", err)
	}

	log.Printf("[sync] plan: add=%d remove=%d playlists=%d", len(plan.AddTracks)+len(plan.AddBooks), len(plan.Remove), len(plan.Playlists))

	if err := e.executePlan(ctx, dev, plan, onProgress); err != nil {
		return fmt.Errorf("executing sync plan: %w", err)
	}

	buildPlaylists(dev, plan)
	log.Printf("[sync] built %d playlists, total %d playlists in DB", len(plan.Playlists), len(dev.DB.Playlists))

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

	if err := e.cfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	onProgress(Progress{Phase: "done", Message: "Sync complete!", Percent: 100})
	return nil
}

func (e *Engine) syncPlayCounts(ctx context.Context, dev *ipod.Device, onProgress ProgressFunc) error {
	if e.nav == nil {
		return nil
	}
	onProgress(Progress{Phase: "scrobble", Message: "Syncing play counts..."})

	for _, track := range dev.DB.Tracks {
		if track.SourceID == "" || track.MediaType != itunesdb.MediaTypeMusic {
			continue
		}

		prev := e.cfg.SyncState.TrackPlayCounts[track.SourceID]
		delta := int(track.PlayCount) - prev.PlayCount

		for range delta {
			if err := e.nav.Scrobble(track.SourceID); err != nil {
				return err
			}
		}

		if delta > 0 {
			e.cfg.SyncState.TrackPlayCounts[track.SourceID] = config.TrackSyncState{
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

	for _, track := range dev.DB.Tracks {
		if track.SourceID == "" || track.MediaType != itunesdb.MediaTypeAudiobook {
			continue
		}

		progress, err := e.abs.GetProgress(track.SourceID)
		if err != nil {
			continue
		}

		iPodTime := float64(track.BookmarkTime) / 1000.0
		prev := e.cfg.SyncState.BookProgress[track.SourceID]

		if progress != nil {
			absTime := progress.CurrentTime

			ipodNewer := iPodTime != prev.CurrentTime && track.LastPlayed != 0
			absNewer := absTime != prev.CurrentTime && progress.UpdatedAt > prev.LastSync

			if ipodNewer && (!absNewer || int64(track.LastPlayed)-itunesdb.MacEpochDelta > progress.UpdatedAt/1000) {
				if err := e.abs.UpdateProgress(track.SourceID, iPodTime, progress.Duration); err != nil {
					continue
				}
			} else if absNewer {
				track.BookmarkTime = uint32(absTime * 1000)
			}
		} else if iPodTime > 0 {
			_ = e.abs.UpdateProgress(track.SourceID, iPodTime, float64(track.Duration)/1000.0)
		}

		e.cfg.SyncState.BookProgress[track.SourceID] = config.BookSyncState{
			CurrentTime: float64(track.BookmarkTime) / 1000.0,
			Duration:    prev.Duration,
			LastSync:    time.Now().Unix(),
		}
	}

	return nil
}

func (e *Engine) buildPlan(ctx context.Context, dev *ipod.Device, onProgress ProgressFunc) (*SyncPlan, error) {
	onProgress(Progress{Phase: "plan", Message: "Building sync plan..."})
	return BuildPlan(ctx, e.cfg, e.nav, e.abs, dev)
}

func (e *Engine) executePlan(ctx context.Context, dev *ipod.Device, plan *SyncPlan, onProgress ProgressFunc) error {
	for _, id := range plan.Remove {
		removed := dev.DB.RemoveTrack(id)
		if removed != nil && removed.Path != "" {
			absPath := ipod.FromiPodPath(dev.Info.MountPoint, removed.Path)
			os.Remove(absPath)
		}
	}

	total := len(plan.AddTracks) + len(plan.AddBooks)
	for i, item := range plan.AddTracks {
		onProgress(Progress{
			Phase:   "transfer",
			Current: i + 1,
			Total:   total,
			Message: fmt.Sprintf("Transferring: %s", item.Title),
			Percent: float64(i+1) / float64(total) * 100,
		})

		if err := TransferTrack(ctx, e.nav, dev, item); err != nil {
			return err
		}
	}

	for i, item := range plan.AddBooks {
		idx := len(plan.AddTracks) + i
		onProgress(Progress{
			Phase:   "transfer",
			Current: idx + 1,
			Total:   total,
			Message: fmt.Sprintf("Downloading: %s", item.Title),
			Percent: float64(idx) / float64(total) * 100,
		})

		if err := TransferBook(ctx, e.abs, dev, item, func(step string) {
			onProgress(Progress{
				Phase:   "transfer",
				Current: idx + 1,
				Total:   total,
				Message: fmt.Sprintf("%s: %s", step, item.Title),
				Percent: float64(idx+1) / float64(total) * 100,
			})
		}); err != nil {
			return err
		}
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
	for _, pl := range dev.DB.Playlists {
		if pl.IsMaster {
			pl.Tracks = managed
			break
		}
	}
}
