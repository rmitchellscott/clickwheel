package sync

import (
	"context"
	"slices"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/config"
	"clickwheel/internal/ipod"
	"clickwheel/internal/navidrome"
)

type TrackItem struct {
	SourceID string
	Title    string
	Artist   string
	Album    string
	Genre    string
	Track    int
	Year     int
	Duration int
	Size     int64
	Suffix   string
}

type BookItem struct {
	SourceID string
	Title    string
	Author   string
	Duration float64
	Chapters []audiobookshelf.Chapter
}

type PlaylistPlan struct {
	Name     string
	TrackIDs []string
}

type SyncPlan struct {
	Remove    []string
	AddTracks []TrackItem
	AddBooks  []BookItem
	Playlists []PlaylistPlan
}

func BuildPlan(ctx context.Context, cfg *config.Config, nav *navidrome.Client, abs *audiobookshelf.Client, dev *ipod.Device) (*SyncPlan, error) {
	plan := &SyncPlan{}

	existingIDs := make(map[string]bool)
	for _, t := range dev.DB.Tracks {
		if t.SourceID != "" {
			existingIDs[t.SourceID] = true
		}
	}

	wantedIDs := make(map[string]bool)

	if nav != nil {
		playlists, err := nav.GetPlaylists()
		if err != nil {
			return nil, err
		}

		for _, pl := range playlists {
			if slices.Contains(cfg.Exclusions.Playlists, pl.ID) {
				continue
			}

			detail, err := nav.GetPlaylist(pl.ID)
			if err != nil {
				continue
			}

			var plTrackIDs []string
			for _, song := range detail.Entry {
				wantedIDs[song.ID] = true
				plTrackIDs = append(plTrackIDs, song.ID)
				if !existingIDs[song.ID] {
					plan.AddTracks = append(plan.AddTracks, TrackItem{
						SourceID: song.ID,
						Title:    song.Title,
						Artist:   song.Artist,
						Album:    song.Album,
						Genre:    song.Genre,
						Track:    song.Track,
						Year:     song.Year,
						Duration: song.Duration,
						Size:     song.Size,
						Suffix:   song.Suffix,
					})
				}
			}

			plan.Playlists = append(plan.Playlists, PlaylistPlan{
				Name:     pl.Name,
				TrackIDs: plTrackIDs,
			})
		}
	}

	if abs != nil {
		libraries, err := abs.GetLibraries()
		if err != nil {
			return nil, err
		}

		for _, lib := range libraries {
			books, err := abs.GetBooks(lib.ID)
			if err != nil {
				continue
			}

			for _, book := range books {
				if slices.Contains(cfg.Exclusions.Books, book.ID) {
					continue
				}

				wantedIDs[book.ID] = true
				if !existingIDs[book.ID] {
					plan.AddBooks = append(plan.AddBooks, BookItem{
						SourceID: book.ID,
						Title:    book.Media.Metadata.Title,
						Author:   book.Media.Metadata.Author,
						Duration: book.Media.Duration,
						Chapters: book.Media.Chapters,
					})
				}
			}
		}
	}

	for id := range existingIDs {
		if !wantedIDs[id] {
			plan.Remove = append(plan.Remove, id)
		}
	}

	return plan, nil
}
