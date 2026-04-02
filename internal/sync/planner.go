package sync

import (
	"context"
	"log"
	"strconv"
	stdsync "sync"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/config"
	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
	"clickwheel/internal/subsonic"
)

func toSet(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

type TrackItem struct {
	SourceID   string
	Title      string
	Artist     string
	Album      string
	Genre      string
	Track      int
	Year       int
	Duration   int
	Size       int64
	Suffix     string
	CoverArtID string
}

type BookItem struct {
	SourceID   string
	Title      string
	Author     string
	Duration   float64
	Size       int64
	Chapters   []audiobookshelf.Chapter
	SplitParts []config.BookSplitPart
}

type PodcastEpisodeItem struct {
	SourceID      string
	ItemID        string
	EpisodeID     string
	Title         string
	ShowName      string
	Author        string
	Duration      float64
	Size          int64
	Ino           string
	Ext           string
	EpisodeNumber int
	Season        int
	PublishedAt   int64
}

type PlaylistPlan struct {
	Name     string
	TrackIDs []string
}

type SyncPlan struct {
	RemoveTracks   []string
	RemoveBooks    []string
	RemovePodcasts []string
	AddTracks      []TrackItem
	AddBooks       []BookItem
	AddPodcasts    []PodcastEpisodeItem
	Playlists      []PlaylistPlan
}

func BuildPlan(ctx context.Context, cfg *config.DeviceConfig, sub *subsonic.Client, abs *audiobookshelf.Client, dev *ipod.Device, finishedPodcasts map[string]bool) (*SyncPlan, error) {
	plan := &SyncPlan{}

	existingIDs := make(map[string]bool)
	for _, t := range dev.DB.Tracks {
		if t.SourceID != "" {
			existingIDs[t.SourceID] = true
		}
	}

	wantedIDs := make(map[string]bool)

	includedPlaylists := toSet(cfg.Inclusions.Playlists)
	includedArtists := toSet(cfg.Inclusions.Artists)
	includedAlbums := toSet(cfg.Inclusions.Albums)
	includedBooks := toSet(cfg.Inclusions.Books)
	includedPodcasts := toSet(cfg.Inclusions.Podcasts)

	targetFormat := cfg.SyncSettings.MusicFormat
	targetBitRate := cfg.SyncSettings.MusicBitRate
	if targetBitRate <= 0 {
		targetBitRate = 256
	}

	addSong := func(song subsonic.Song) {
		if wantedIDs[song.ID] {
			return
		}
		wantedIDs[song.ID] = true
		if !existingIDs[song.ID] {
			size := song.Size
			needsTranscode := !lossyCompatible[song.Suffix] &&
				song.Suffix != targetFormat &&
				!(song.Suffix == "m4a" && targetFormat == "aac")
			if needsTranscode && song.Duration > 0 {
				if targetFormat == "alac" {
					size = int64(song.Duration) * 800 * 1000 / 8
				} else {
					size = int64(song.Duration) * int64(targetBitRate) * 1000 / 8
				}
			}
			plan.AddTracks = append(plan.AddTracks, TrackItem{
				SourceID:   song.ID,
				Title:      song.Title,
				Artist:     song.Artist,
				Album:      song.Album,
				Genre:      song.Genre,
				Track:      song.Track,
				Year:       song.Year,
				Duration:   song.Duration,
				Size:       size,
				Suffix:     song.Suffix,
				CoverArtID: song.CoverArt,
			})
		}
	}

	if sub != nil {
		var albumCacheMu stdsync.Mutex
		albumCache := make(map[string]*subsonic.AlbumDetail)
		fetchAlbum := func(id string) (*subsonic.AlbumDetail, error) {
			albumCacheMu.Lock()
			if cached, ok := albumCache[id]; ok {
				albumCacheMu.Unlock()
				return cached, nil
			}
			albumCacheMu.Unlock()
			detail, err := sub.GetAlbum(id)
			if err != nil {
				return nil, err
			}
			albumCacheMu.Lock()
			albumCache[id] = detail
			albumCacheMu.Unlock()
			return detail, nil
		}

		const fetchWorkers = 8

		playlists, err := sub.GetPlaylists()
		if err != nil {
			return nil, err
		}

		for _, pl := range playlists {
			if !includedPlaylists[pl.ID] {
				continue
			}

			detail, err := sub.GetPlaylist(pl.ID)
			if err != nil {
				continue
			}

			var plTrackIDs []string
			for _, song := range detail.Entry {
				plTrackIDs = append(plTrackIDs, song.ID)
				addSong(song)
			}

			plan.Playlists = append(plan.Playlists, PlaylistPlan{
				Name:     pl.Name,
				TrackIDs: plTrackIDs,
			})
		}

		artists, err := sub.GetArtists()
		if err == nil {
			var includedArtistIDs []string
			for _, ar := range artists {
				if includedArtists[ar.ID] {
					includedArtistIDs = append(includedArtistIDs, ar.ID)
				}
			}

			type artistResult struct {
				detail *subsonic.ArtistDetail
			}
			artistResults := make([]artistResult, len(includedArtistIDs))

			jobs := make(chan int, fetchWorkers)
			var wg stdsync.WaitGroup
			for range fetchWorkers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for i := range jobs {
						detail, err := sub.GetArtist(includedArtistIDs[i])
						if err == nil {
							artistResults[i] = artistResult{detail: detail}
						}
					}
				}()
			}
			for i := range includedArtistIDs {
				jobs <- i
			}
			close(jobs)
			wg.Wait()

			var albumIDs []string
			seen := make(map[string]bool)
			for _, ar := range artistResults {
				if ar.detail == nil {
					continue
				}
				for _, album := range ar.detail.Album {
					if !seen[album.ID] {
						seen[album.ID] = true
						albumIDs = append(albumIDs, album.ID)
					}
				}
			}

			albumResults := make([]*subsonic.AlbumDetail, len(albumIDs))
			jobs2 := make(chan int, fetchWorkers)
			var wg2 stdsync.WaitGroup
			for range fetchWorkers {
				wg2.Add(1)
				go func() {
					defer wg2.Done()
					for i := range jobs2 {
						detail, err := fetchAlbum(albumIDs[i])
						if err == nil {
							albumResults[i] = detail
						}
					}
				}()
			}
			for i := range albumIDs {
				jobs2 <- i
			}
			close(jobs2)
			wg2.Wait()

			for i, detail := range albumResults {
				if detail == nil {
					continue
				}
				_ = albumIDs[i]
				for _, song := range detail.Song {
					addSong(song)
				}
			}
		}

		const albumPageSize = 500
		var allAlbums []subsonic.Album
		for offset := 0; ; offset += albumPageSize {
			page, err := sub.GetAlbums(offset, albumPageSize)
			if err != nil {
				break
			}
			allAlbums = append(allAlbums, page...)
			if len(page) < albumPageSize {
				break
			}
		}

		var includedAlbumIDs []string
		for _, al := range allAlbums {
			if includedAlbums[al.ID] {
				includedAlbumIDs = append(includedAlbumIDs, al.ID)
			}
		}

		albumBrowseResults := make([]*subsonic.AlbumDetail, len(includedAlbumIDs))
		jobs3 := make(chan int, fetchWorkers)
		var wg3 stdsync.WaitGroup
		for range fetchWorkers {
			wg3.Add(1)
			go func() {
				defer wg3.Done()
				for i := range jobs3 {
					detail, err := fetchAlbum(includedAlbumIDs[i])
					if err == nil {
						albumBrowseResults[i] = detail
					}
				}
			}()
		}
		for i := range includedAlbumIDs {
			jobs3 <- i
		}
		close(jobs3)
		wg3.Wait()

		for _, detail := range albumBrowseResults {
			if detail == nil {
				continue
			}
			for _, song := range detail.Song {
				addSong(song)
			}
		}
	}

	if abs != nil {
		libraries, err := abs.GetLibraries()
		if err != nil {
			log.Printf("[plan] ABS GetLibraries error: %v", err)
			return nil, err
		}
		log.Printf("[plan] ABS libraries: %d", len(libraries))

		for _, lib := range libraries {
			if lib.MediaType == "podcast" {
				podcasts, err := abs.GetPodcasts(lib.ID)
				if err != nil {
					log.Printf("[plan] ABS GetPodcasts(%s) error: %v", lib.ID, err)
					continue
				}
				log.Printf("[plan] ABS library %q: %d podcasts", lib.Name, len(podcasts))

				for _, pod := range podcasts {
					if !includedPodcasts[pod.ID] {
						continue
					}

					detail, err := abs.GetPodcast(pod.ID)
					if err != nil {
						log.Printf("[plan] ABS GetPodcast(%s) error: %v", pod.ID, err)
						continue
					}

					for _, ep := range detail.Media.Episodes {
						sourceID := pod.ID + "|" + ep.ID
						if finishedPodcasts[sourceID] {
							continue
						}
						wantedIDs[sourceID] = true
						if !existingIDs[sourceID] {
							epNum, _ := strconv.Atoi(ep.Episode)
							seasonNum, _ := strconv.Atoi(ep.Season)
							plan.AddPodcasts = append(plan.AddPodcasts, PodcastEpisodeItem{
								SourceID:      sourceID,
								ItemID:        pod.ID,
								EpisodeID:     ep.ID,
								Title:         ep.Title,
								ShowName:      pod.Media.Metadata.Title,
								Author:        pod.Media.Metadata.Author,
								Duration:      ep.AudioFile.Duration,
								Size:          ep.AudioFile.Metadata.Size,
								Ino:           ep.AudioFile.Ino,
								Ext:           ep.AudioFile.Metadata.Ext,
								EpisodeNumber: epNum,
								Season:        seasonNum,
								PublishedAt:   ep.PublishedAt,
							})
						}
					}
				}
				continue
			}

			books, err := abs.GetBooks(lib.ID)
			if err != nil {
				log.Printf("[plan] ABS GetBooks(%s) error: %v", lib.ID, err)
				continue
			}
			log.Printf("[plan] ABS library %q: %d books", lib.Name, len(books))

			splitEnabled := cfg.SyncSettings.SplitLongBooks
			splitLimit := cfg.SyncSettings.SplitHoursLimit
			if splitLimit <= 0 {
				splitLimit = 8
			}

			for _, book := range books {
				if !includedBooks[book.ID] {
					continue
				}

				chapters := book.Media.Chapters
				duration := book.Media.Duration
				if splitEnabled && len(chapters) == 0 {
					detail, err := abs.GetBook(book.ID)
					if err == nil && detail != nil {
						chapters = detail.Media.Chapters
						if detail.Media.Duration > 0 {
							duration = detail.Media.Duration
						}
					}
				}

				var splitParts []config.BookSplitPart
				if splitEnabled {
					splitParts = computeBookSplitPlan(chapters, duration, splitLimit)
				}

				if splitParts != nil {
					existingSplit := cfg.SyncState.BookSplits[book.ID]
					partsMatch := len(existingSplit.Parts) == len(splitParts) && existingSplit.SplitHoursLimit == splitLimit

					for _, p := range splitParts {
						sid := bookPartSourceID(book.ID, p.Index)
						wantedIDs[sid] = true
						if !existingIDs[sid] || !partsMatch {
							plan.AddBooks = append(plan.AddBooks, BookItem{
								SourceID:   book.ID,
								Title:      book.Media.Metadata.Title,
								Author:     book.Media.Metadata.Author,
								Duration:   duration,
								Size:       book.Size,
								Chapters:   chapters,
								SplitParts: splitParts,
							})
							break
						}
					}
				} else {
					wantedIDs[book.ID] = true
					if !existingIDs[book.ID] {
						plan.AddBooks = append(plan.AddBooks, BookItem{
							SourceID: book.ID,
							Title:    book.Media.Metadata.Title,
							Author:   book.Media.Metadata.Author,
							Duration: book.Media.Duration,
							Size:     book.Size,
							Chapters: book.Media.Chapters,
						})
					}
				}
			}
		}
	}

	trackTypes := make(map[string]uint32)
	for _, t := range dev.DB.Tracks {
		if t.SourceID != "" {
			trackTypes[t.SourceID] = t.MediaType
		}
	}
	for id := range existingIDs {
		if !wantedIDs[id] {
			switch trackTypes[id] {
			case itunesdb.MediaTypeAudiobook:
				plan.RemoveBooks = append(plan.RemoveBooks, id)
			case itunesdb.MediaTypePodcast:
				plan.RemovePodcasts = append(plan.RemovePodcasts, id)
			default:
				plan.RemoveTracks = append(plan.RemoveTracks, id)
			}
		}
	}

	return plan, nil
}
