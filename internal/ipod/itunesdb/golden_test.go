package itunesdb

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type goldenDataset struct {
	Type       int              `json:"type"`
	TotalLen   int              `json:"total_len"`
	ChildMagic string           `json:"child_magic"`
	ChildCount int              `json:"child_count"`
	Tracks     []goldenTrack    `json:"tracks,omitempty"`
	Playlists  []goldenPlaylist `json:"playlists,omitempty"`
}

type goldenTrack struct {
	TrackID   int   `json:"track_id"`
	MhodCount int   `json:"mhod_count"`
	MhodTypes []int `json:"mhod_types"`
}

type goldenPlaylist struct {
	MhodCount int `json:"mhod_count"`
	ItemCount int `json:"item_count"`
	IsMaster  int `json:"is_master"`
}

type goldenStructure struct {
	Size        int             `json:"size"`
	Version     int             `json:"version"`
	NumDatasets int             `json:"num_datasets"`
	Datasets    []goldenDataset `json:"datasets"`
}

func extractStructure(data []byte) *goldenStructure {
	if len(data) < 244 || string(data[0:4]) != "mhbd" {
		return nil
	}

	s := &goldenStructure{
		Size:        len(data),
		Version:     int(binary.LittleEndian.Uint32(data[0x10:0x14])),
		NumDatasets: int(binary.LittleEndian.Uint32(data[0x14:0x18])),
	}

	hdrLen := int(binary.LittleEndian.Uint32(data[4:8]))
	pos := hdrLen

	for i := 0; i < s.NumDatasets && pos+16 <= len(data); i++ {
		if string(data[pos:pos+4]) != "mhsd" {
			break
		}

		dsTotal := int(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))
		dsType := int(binary.LittleEndian.Uint32(data[pos+12 : pos+16]))
		dsHdr := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))

		ds := goldenDataset{
			Type:     dsType,
			TotalLen: dsTotal,
		}

		childPos := pos + dsHdr
		if childPos+12 <= len(data) {
			ds.ChildMagic = string(data[childPos : childPos+4])
			ds.ChildCount = int(binary.LittleEndian.Uint32(data[childPos+8 : childPos+12]))

			dsEnd := pos + dsTotal
			if ds.ChildMagic == "mhlt" {
				ds.Tracks = extractTracks(data, childPos, dsEnd)
			} else if ds.ChildMagic == "mhlp" {
				ds.Playlists = extractPlaylists(data, childPos, dsEnd)
			}
		}

		s.Datasets = append(s.Datasets, ds)
		pos += dsTotal
	}

	return s
}

func extractTracks(data []byte, mhltPos, dsEnd int) []goldenTrack {
	hdrLen := int(binary.LittleEndian.Uint32(data[mhltPos+4 : mhltPos+8]))
	count := int(binary.LittleEndian.Uint32(data[mhltPos+8 : mhltPos+12]))
	var tracks []goldenTrack
	pos := mhltPos + hdrLen

	for i := 0; i < count && pos+16 <= dsEnd; i++ {
		if string(data[pos:pos+4]) != "mhit" {
			break
		}
		trackHdrLen := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		totalLen := int(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))
		mhodCount := int(binary.LittleEndian.Uint32(data[pos+0x0C : pos+0x10]))
		trackID := int(binary.LittleEndian.Uint32(data[pos+0x10 : pos+0x14]))

		var mhodTypes []int
		mpos := pos + trackHdrLen
		for j := 0; j < mhodCount && mpos+16 <= pos+totalLen; j++ {
			if string(data[mpos:mpos+4]) != "mhod" {
				break
			}
			mt := int(binary.LittleEndian.Uint32(data[mpos+12 : mpos+16]))
			mhodTypes = append(mhodTypes, mt)
			mpos += int(binary.LittleEndian.Uint32(data[mpos+8 : mpos+12]))
		}

		tracks = append(tracks, goldenTrack{
			TrackID:   trackID,
			MhodCount: mhodCount,
			MhodTypes: mhodTypes,
		})
		pos += totalLen
	}
	return tracks
}

func extractPlaylists(data []byte, mhlpPos, dsEnd int) []goldenPlaylist {
	hdrLen := int(binary.LittleEndian.Uint32(data[mhlpPos+4 : mhlpPos+8]))
	count := int(binary.LittleEndian.Uint32(data[mhlpPos+8 : mhlpPos+12]))
	var playlists []goldenPlaylist
	pos := mhlpPos + hdrLen

	for i := 0; i < count && pos+24 <= dsEnd; i++ {
		if string(data[pos:pos+4]) != "mhyp" {
			break
		}
		totalLen := int(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))
		mhodCount := int(binary.LittleEndian.Uint32(data[pos+0x0C : pos+0x10]))
		itemCount := int(binary.LittleEndian.Uint32(data[pos+0x10 : pos+0x14]))
		isMaster := int(binary.LittleEndian.Uint32(data[pos+0x14 : pos+0x18]))

		playlists = append(playlists, goldenPlaylist{
			MhodCount: mhodCount,
			ItemCount: itemCount,
			IsMaster:  isMaster,
		})
		pos += totalLen
	}
	return playlists
}

func buildTestDB(scenario string, caps *DeviceCapabilities) *Database {
	db := NewDatabase()

	switch scenario {
	case "basic_music":
		addBasicMusic(db)
	case "multi_album":
		addMultiAlbum(db)
	case "audiobook":
		addAudiobook(db)
	case "podcast":
		addPodcast(db)
	case "compilation":
		addCompilation(db)
	case "unicode":
		addUnicode(db)
	case "mixed":
		addMixed(db)
	case "empty":
		// no tracks
	case "playlist":
		addPlaylist(db)
	case "smart_playlist":
		addSmartPlaylist(db)
	}

	return db
}

func addBasicMusic(db *Database) {
	tracks := []*Track{
		{Title: "Track One", Artist: "Artist A", Album: "Album X", Genre: "Rock",
			Path: ":iPod_Control:Music:F00:AAAA.mp3", Size: 5000000, Duration: 240000,
			BitRate: 320, SampleRate: 44100, FiletypeKey: "mp3", Year: 2020,
			TrackNumber: 1, TotalTracks: 3, MediaType: MediaTypeMusic},
		{Title: "Track Two", Artist: "Artist A", Album: "Album X", Genre: "Rock",
			Path: ":iPod_Control:Music:F01:BBBB.mp3", Size: 4500000, Duration: 200000,
			BitRate: 320, SampleRate: 44100, FiletypeKey: "mp3", Year: 2020,
			TrackNumber: 2, TotalTracks: 3, MediaType: MediaTypeMusic},
		{Title: "Track Three", Artist: "Artist A", Album: "Album X", Genre: "Rock",
			Path: ":iPod_Control:Music:F02:CCCC.mp3", Size: 4800000, Duration: 220000,
			BitRate: 320, SampleRate: 44100, FiletypeKey: "mp3", Year: 2020,
			TrackNumber: 3, TotalTracks: 3, MediaType: MediaTypeMusic},
	}
	for _, t := range tracks {
		db.AddTrack(t)
	}
}

func addMultiAlbum(db *Database) {
	tracks := []*Track{
		{Title: "Alpha", Artist: "Band One", Album: "First Album", Genre: "Pop",
			Path: ":iPod_Control:Music:F00:AAA1.mp3", Size: 5000000, Duration: 200000,
			BitRate: 256, SampleRate: 44100, FiletypeKey: "mp3", Year: 2018,
			TrackNumber: 1, TotalTracks: 2, MediaType: MediaTypeMusic},
		{Title: "Beta", Artist: "Band One", Album: "First Album", Genre: "Pop",
			Path: ":iPod_Control:Music:F01:AAA2.mp3", Size: 4800000, Duration: 190000,
			BitRate: 256, SampleRate: 44100, FiletypeKey: "mp3", Year: 2018,
			TrackNumber: 2, TotalTracks: 2, MediaType: MediaTypeMusic},
		{Title: "Gamma", Artist: "Singer Two", Album: "Second Record", Genre: "Jazz",
			Path: ":iPod_Control:Music:F02:BBB1.mp3", Size: 6000000, Duration: 300000,
			BitRate: 320, SampleRate: 48000, FiletypeKey: "m4a", Year: 2019,
			TrackNumber: 1, TotalTracks: 2, MediaType: MediaTypeMusic},
		{Title: "Delta", Artist: "Singer Two", Album: "Second Record", Genre: "Jazz",
			Path: ":iPod_Control:Music:F03:BBB2.mp3", Size: 5500000, Duration: 280000,
			BitRate: 320, SampleRate: 48000, FiletypeKey: "m4a", Year: 2019,
			TrackNumber: 2, TotalTracks: 2, MediaType: MediaTypeMusic},
		{Title: "Epsilon", Artist: "DJ Three", Album: "Third Mix", Genre: "Electronic",
			Path: ":iPod_Control:Music:F04:CCC1.mp3", Size: 7000000, Duration: 360000,
			BitRate: 320, SampleRate: 44100, FiletypeKey: "mp3", Year: 2021,
			TrackNumber: 1, TotalTracks: 2, MediaType: MediaTypeMusic},
		{Title: "Zeta", Artist: "DJ Three", Album: "Third Mix", Genre: "Electronic",
			Path: ":iPod_Control:Music:F05:CCC2.mp3", Size: 6500000, Duration: 340000,
			BitRate: 320, SampleRate: 44100, FiletypeKey: "mp3", Year: 2021,
			TrackNumber: 2, TotalTracks: 2, MediaType: MediaTypeMusic},
	}
	for _, t := range tracks {
		db.AddTrack(t)
	}
}

func addAudiobook(db *Database) {
	db.AddTrack(&Track{
		Title: "Chapter 1 - The Beginning", Artist: "Author Name", Album: "My Audiobook",
		Path: ":iPod_Control:Music:F00:BOOK.m4b", Size: 50000000, Duration: 3600000,
		FiletypeKey: "m4b", MediaType: MediaTypeAudiobook,
		RememberPosition: 1, SkipWhenShuffling: 1, BookmarkTime: 1200000,
	})
}

func addPodcast(db *Database) {
	db.AddTrack(&Track{
		Title: "Episode 1: Pilot", Artist: "Podcast Host", Album: "My Podcast",
		Path: ":iPod_Control:Music:F00:POD1.mp3", Size: 30000000, Duration: 1800000,
		BitRate: 128, SampleRate: 44100, FiletypeKey: "mp3", MediaType: MediaTypePodcast,
		PodcastFlag: 1, RememberPosition: 1, SkipWhenShuffling: 1,
		PodcastEnclosureURL: "https://example.com/ep1.mp3",
		PodcastRSSURL:       "https://example.com/feed.xml",
		Category:            "Technology",
	})
	db.AddTrack(&Track{
		Title: "Episode 2: Follow Up", Artist: "Podcast Host", Album: "My Podcast",
		Path: ":iPod_Control:Music:F01:POD2.mp3", Size: 35000000, Duration: 2100000,
		BitRate: 128, SampleRate: 44100, FiletypeKey: "mp3", MediaType: MediaTypePodcast,
		PodcastFlag: 1, RememberPosition: 1, SkipWhenShuffling: 1,
		PodcastEnclosureURL: "https://example.com/ep2.mp3",
		PodcastRSSURL:       "https://example.com/feed.xml",
		Category:            "Technology",
	})
}

func addCompilation(db *Database) {
	artists := []string{"Artist X", "Artist Y", "Artist Z"}
	for i, art := range artists {
		db.AddTrack(&Track{
			Title: fmt.Sprintf("Comp Track %d", i+1), Artist: art,
			Album: "Various Artists Compilation", AlbumArtist: "Various Artists",
			Genre: "Pop",
			Path:  fmt.Sprintf(":iPod_Control:Music:F0%d:COMP%d.mp3", i, i),
			Size:  4000000, Duration: 180000, BitRate: 256, SampleRate: 44100,
			FiletypeKey: "mp3", Year: 2022, TrackNumber: uint16(i + 1), TotalTracks: 3,
			Compilation: true, MediaType: MediaTypeMusic,
		})
	}
}

func addUnicode(db *Database) {
	db.AddTrack(&Track{
		Title: "Für Elise", Artist: "ベートーヴェン", Album: "Classique", Genre: "Classique",
		Path: ":iPod_Control:Music:F00:UNIC.mp3", Size: 3000000, Duration: 180000,
		BitRate: 256, SampleRate: 44100, FiletypeKey: "mp3", Year: 1810,
		MediaType: MediaTypeMusic,
	})
	db.AddTrack(&Track{
		Title: "Ça Plane Pour Moi", Artist: "Plastic Bertrand", Album: "Ça Plane Pour Moi",
		Genre: "Punk",
		Path:  ":iPod_Control:Music:F01:FREN.mp3", Size: 3500000, Duration: 190000,
		BitRate: 256, SampleRate: 44100, FiletypeKey: "mp3", Year: 1977,
		MediaType: MediaTypeMusic,
	})
}

func addMixed(db *Database) {
	// 2 music + 1 audiobook + 1 podcast
	db.AddTrack(&Track{
		Title: "Track One", Artist: "Artist A", Album: "Album X", Genre: "Rock",
		Path: ":iPod_Control:Music:F00:AAAA.mp3", Size: 5000000, Duration: 240000,
		BitRate: 320, SampleRate: 44100, FiletypeKey: "mp3", Year: 2020,
		TrackNumber: 1, TotalTracks: 3, MediaType: MediaTypeMusic,
	})
	db.AddTrack(&Track{
		Title: "Track Two", Artist: "Artist A", Album: "Album X", Genre: "Rock",
		Path: ":iPod_Control:Music:F01:BBBB.mp3", Size: 4500000, Duration: 200000,
		BitRate: 320, SampleRate: 44100, FiletypeKey: "mp3", Year: 2020,
		TrackNumber: 2, TotalTracks: 3, MediaType: MediaTypeMusic,
	})
	db.AddTrack(&Track{
		Title: "Chapter 1 - The Beginning", Artist: "Author Name", Album: "My Audiobook",
		Path: ":iPod_Control:Music:F00:BOOK.m4b", Size: 50000000, Duration: 3600000,
		FiletypeKey: "m4b", MediaType: MediaTypeAudiobook,
		RememberPosition: 1, SkipWhenShuffling: 1, BookmarkTime: 1200000,
	})
	db.AddTrack(&Track{
		Title: "Episode 1: Pilot", Artist: "Podcast Host", Album: "My Podcast",
		Path: ":iPod_Control:Music:F00:POD1.mp3", Size: 30000000, Duration: 1800000,
		BitRate: 128, SampleRate: 44100, FiletypeKey: "mp3", MediaType: MediaTypePodcast,
		PodcastFlag: 1, RememberPosition: 1, SkipWhenShuffling: 1,
		PodcastEnclosureURL: "https://example.com/ep1.mp3",
		PodcastRSSURL:       "https://example.com/feed.xml",
		Category:            "Technology",
	})
}

func addPlaylist(db *Database) {
	addMultiAlbum(db)

	db.Playlists = append(db.Playlists, &Playlist{
		Name:   "Favorites",
		Tracks: []*Track{db.Tracks[0], db.Tracks[2], db.Tracks[4]},
	})
	db.Playlists = append(db.Playlists, &Playlist{
		Name:   "Chill",
		Tracks: []*Track{db.Tracks[1], db.Tracks[3]},
	})
}

func addSmartPlaylist(db *Database) {
	addMultiAlbum(db)

	db.Playlists = append(db.Playlists, &Playlist{
		Name:    "Rock Songs",
		Tracks:  []*Track{db.Tracks[0], db.Tracks[1]},
		IsSmart: true,
		SmartPrefs: &SmartPlaylistPrefs{
			LiveUpdate:  true,
			CheckRules:  true,
			CheckLimits: false,
			LimitType:   0x03,
			LimitSort:   0x02,
			LimitValue:  25,
		},
		SmartRules: &SmartPlaylistRules{
			Conjunction: "AND",
			Rules: []SmartPlaylistRule{
				{
					FieldID:     0x08,
					ActionID:    0x01000002,
					StringValue: "Rock",
				},
			},
		},
	})
}

func capsForProfile(profile string) *DeviceCapabilities {
	switch profile {
	case "ipod1g":
		return &DeviceCapabilities{
			SupportsPodcast: false,
			SupportsVideo:   false,
			SupportsGapless: false,
			DBVersion:       0x13,
		}
	case "ipod4g":
		return &DeviceCapabilities{
			SupportsPodcast: true,
			SupportsVideo:   false,
			SupportsGapless: false,
			DBVersion:       0x13,
		}
	case "ipodvideo5g":
		return &DeviceCapabilities{
			SupportsPodcast: true,
			SupportsVideo:   true,
			SupportsGapless: true,
			DBVersion:       0x19,
		}
	case "classic":
		return &DeviceCapabilities{
			SupportsPodcast: true,
			SupportsVideo:   true,
			SupportsGapless: true,
			DBVersion:       0x30,
		}
	case "none":
		return nil
	}
	return nil
}

func TestGoldenFiles(t *testing.T) {
	testdataDir := "testdata"

	scenarios := []string{
		"basic_music", "multi_album", "audiobook", "podcast",
		"compilation", "unicode", "mixed", "empty",
		"playlist", "smart_playlist",
	}
	profiles := []string{"ipod1g", "ipod4g", "ipodvideo5g", "classic", "none"}

	for _, profile := range profiles {
		for _, scenario := range scenarios {
			caps := capsForProfile(profile)
			if scenario == "podcast" || scenario == "mixed" {
				if caps != nil && !caps.SupportsPodcast {
					continue
				}
			}

			name := fmt.Sprintf("golden_%s_%s", scenario, profile)
			t.Run(name, func(t *testing.T) {
				jsonPath := filepath.Join(testdataDir, name+".json")
				jsonData, err := os.ReadFile(jsonPath)
				if err != nil {
					t.Skipf("golden file not found: %s", jsonPath)
					return
				}

				var golden goldenStructure
				if err := json.Unmarshal(jsonData, &golden); err != nil {
					t.Fatalf("parse golden JSON: %v", err)
				}

				db := buildTestDB(scenario, caps)
				ourData := SerializeDatabase(db, caps)
				ours := extractStructure(ourData)
				if ours == nil {
					t.Fatal("failed to extract structure from our output")
				}

				compareStructures(t, &golden, ours)
			})
		}
	}
}

func compareStructures(t *testing.T, golden, ours *goldenStructure) {
	t.Helper()

	if golden.Version != ours.Version {
		t.Errorf("version: golden=0x%X ours=0x%X", golden.Version, ours.Version)
	}

	if golden.NumDatasets != ours.NumDatasets {
		t.Errorf("num_datasets: golden=%d ours=%d", golden.NumDatasets, ours.NumDatasets)
	}

	minDS := len(golden.Datasets)
	if len(ours.Datasets) < minDS {
		minDS = len(ours.Datasets)
	}

	for i := 0; i < minDS; i++ {
		gd := golden.Datasets[i]
		od := ours.Datasets[i]
		prefix := fmt.Sprintf("dataset[%d](type=%d)", i, gd.Type)

		if gd.Type != od.Type {
			t.Errorf("%s type: golden=%d ours=%d", prefix, gd.Type, od.Type)
			continue
		}

		if gd.ChildMagic != od.ChildMagic {
			t.Errorf("%s child_magic: golden=%q ours=%q", prefix, gd.ChildMagic, od.ChildMagic)
		}

		if gd.ChildCount != od.ChildCount {
			t.Errorf("%s child_count: golden=%d ours=%d", prefix, gd.ChildCount, od.ChildCount)
		}

		if gd.Tracks != nil {
			compareTracks(t, prefix, gd.Tracks, od.Tracks)
		}

		if gd.Playlists != nil {
			comparePlaylists(t, prefix, gd.Playlists, od.Playlists)
		}
	}

	for i := minDS; i < len(golden.Datasets); i++ {
		t.Errorf("missing dataset[%d] type=%d", i, golden.Datasets[i].Type)
	}
	for i := minDS; i < len(ours.Datasets); i++ {
		t.Errorf("extra dataset[%d] type=%d", i, ours.Datasets[i].Type)
	}
}

func compareTracks(t *testing.T, prefix string, golden, ours []goldenTrack) {
	t.Helper()
	if len(golden) != len(ours) {
		t.Errorf("%s track_count: golden=%d ours=%d", prefix, len(golden), len(ours))
		return
	}

	for i := range golden {
		gt := golden[i]
		ot := ours[i]
		tp := fmt.Sprintf("%s track[%d]", prefix, i)

		if gt.MhodCount != ot.MhodCount {
			t.Errorf("%s mhod_count: golden=%d ours=%d", tp, gt.MhodCount, ot.MhodCount)
		}

		if len(gt.MhodTypes) != len(ot.MhodTypes) {
			t.Errorf("%s mhod_types len: golden=%v ours=%v", tp, gt.MhodTypes, ot.MhodTypes)
		} else {
			gTypes := intSliceToStr(gt.MhodTypes)
			oTypes := intSliceToStr(ot.MhodTypes)
			if gTypes != oTypes {
				t.Errorf("%s mhod_types: golden=%v ours=%v", tp, gt.MhodTypes, ot.MhodTypes)
			}
		}
	}
}

func comparePlaylists(t *testing.T, prefix string, golden, ours []goldenPlaylist) {
	t.Helper()
	if len(golden) != len(ours) {
		t.Errorf("%s playlist_count: golden=%d ours=%d", prefix, len(golden), len(ours))
		return
	}

	for i := range golden {
		gp := golden[i]
		op := ours[i]
		pp := fmt.Sprintf("%s playlist[%d]", prefix, i)

		if gp.IsMaster != op.IsMaster {
			t.Errorf("%s is_master: golden=%d ours=%d", pp, gp.IsMaster, op.IsMaster)
		}
		if gp.ItemCount != op.ItemCount {
			t.Errorf("%s item_count: golden=%d ours=%d", pp, gp.ItemCount, op.ItemCount)
		}
	}
}

func intSliceToStr(s []int) string {
	parts := make([]string, len(s))
	for i, v := range s {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}
