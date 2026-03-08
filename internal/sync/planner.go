package sync

import (
	"context"
	"log"

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

type PodcastEpisodeItem struct {
	SourceID  string
	ItemID    string
	EpisodeID string
	Title     string
	ShowName  string
	Author    string
	Duration  float64
	Size      int64
	Ino       string
	Ext       string
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

func BuildPlan(ctx context.Context, cfg *config.DeviceConfig, sub *subsonic.Client, abs *audiobookshelf.Client, dev *ipod.Device) (*SyncPlan, error) {
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
			needsTranscode := song.Suffix != targetFormat && !(song.Suffix == "m4a" && targetFormat == "aac")
			if needsTranscode && song.Duration > 0 {
				size = int64(song.Duration) * int64(targetBitRate) * 1000 / 8
			}
			plan.AddTracks = append(plan.AddTracks, TrackItem{
				SourceID: song.ID,
				Title:    song.Title,
				Artist:   song.Artist,
				Album:    song.Album,
				Genre:    song.Genre,
				Track:    song.Track,
				Year:     song.Year,
				Duration: song.Duration,
				Size:     size,
				Suffix:   song.Suffix,
			})
		}
	}

	if sub!= nil {
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
			for _, ar := range artists {
				if !includedArtists[ar.ID] {
					continue
				}
				detail, err := sub.GetArtist(ar.ID)
				if err != nil {
					continue
				}
				for _, album := range detail.Album {
					albumDetail, err := sub.GetAlbum(album.ID)
					if err != nil {
						continue
					}
					for _, song := range albumDetail.Song {
						addSong(song)
					}
				}
			}
		}

		albums, err := sub.GetAlbums(0, 500)
		if err == nil {
			for _, al := range albums {
				if !includedAlbums[al.ID] {
					continue
				}
				albumDetail, err := sub.GetAlbum(al.ID)
				if err != nil {
					continue
				}
				for _, song := range albumDetail.Song {
					addSong(song)
				}
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
						wantedIDs[sourceID] = true
						if !existingIDs[sourceID] {
							plan.AddPodcasts = append(plan.AddPodcasts, PodcastEpisodeItem{
								SourceID:  sourceID,
								ItemID:    pod.ID,
								EpisodeID: ep.ID,
								Title:     ep.Title,
								ShowName:  pod.Media.Metadata.Title,
								Author:    pod.Media.Metadata.Author,
								Duration:  ep.AudioFile.Duration,
								Size:      ep.AudioFile.Metadata.Size,
								Ino:       ep.AudioFile.Ino,
								Ext:       ep.AudioFile.Metadata.Ext,
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

			for _, book := range books {
				if !includedBooks[book.ID] {
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
