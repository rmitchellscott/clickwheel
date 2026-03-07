package itunesdb

import (
	"testing"
	"time"
)

func TestMacTimestamp(t *testing.T) {
	ts := MacTimestamp(time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC))
	expected := uint32(time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC).Unix() + MacEpochDelta)
	if ts != expected {
		t.Errorf("MacTimestamp: got %d, want %d", ts, expected)
	}

	roundTrip := FromMacTimestamp(ts)
	want := time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	if !roundTrip.Equal(want) {
		t.Errorf("FromMacTimestamp: got %v, want %v", roundTrip, want)
	}
}

func TestRoundTrip(t *testing.T) {
	db := NewDatabase()
	db.AddTrack(&Track{
		UniqueID:   1,
		Title:      "Test Song",
		Artist:     "Test Artist",
		Album:      "Test Album",
		Genre:      "Rock",
		Path:       ":iPod_Control:Music:F00:ABCD.mp3",
		Duration:   240000,
		Size:       5000000,
		BitRate:    320,
		SampleRate: 44100,
		MediaType:  MediaTypeMusic,
		PlayCount:  5,
		SourceID:   "nav-123",
	})
	db.AddTrack(&Track{
		UniqueID:         2,
		Title:            "My Audiobook",
		Artist:           "Author Name",
		Album:            "Book Title",
		Path:             ":iPod_Control:Music:F01:EFGH.m4b",
		Duration:         3600000,
		Size:             50000000,
		MediaType:        MediaTypeAudiobook,
		RememberPosition: 1,
		SkipWhenShuffling: 1,
		BookmarkTime:     1200000,
		SourceID:         "abs-456",
	})

	db.Playlists = append(db.Playlists, &Playlist{
		Name:   "Favorites",
		Tracks: []*Track{db.Tracks[0]},
	})

	data := db.Serialize()

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(parsed.Tracks))
	}

	track1 := parsed.Tracks[0]
	if track1.UniqueID != 1 {
		t.Errorf("UniqueID: got %d, want 1", track1.UniqueID)
	}
	if track1.Title != "Test Song" {
		t.Errorf("Title: got %q, want %q", track1.Title, "Test Song")
	}
	if track1.Artist != "Test Artist" {
		t.Errorf("Artist: got %q, want %q", track1.Artist, "Test Artist")
	}
	if track1.Album != "Test Album" {
		t.Errorf("Album: got %q, want %q", track1.Album, "Test Album")
	}
	if track1.Genre != "Rock" {
		t.Errorf("Genre: got %q, want %q", track1.Genre, "Rock")
	}
	if track1.Path != ":iPod_Control:Music:F00:ABCD.mp3" {
		t.Errorf("Path: got %q", track1.Path)
	}
	if track1.Duration != 240000 {
		t.Errorf("Duration: got %d, want 240000", track1.Duration)
	}
	if track1.PlayCount != 5 {
		t.Errorf("PlayCount: got %d, want 5", track1.PlayCount)
	}
	if track1.MediaType != MediaTypeMusic {
		t.Errorf("MediaType: got %d, want %d", track1.MediaType, MediaTypeMusic)
	}

	track2 := parsed.Tracks[1]
	if track2.MediaType != MediaTypeAudiobook {
		t.Errorf("MediaType: got %d, want %d", track2.MediaType, MediaTypeAudiobook)
	}
	if track2.RememberPosition != 1 {
		t.Errorf("RememberPosition: got %d, want 1", track2.RememberPosition)
	}
	if track2.SkipWhenShuffling != 1 {
		t.Errorf("SkipWhenShuffling: got %d, want 1", track2.SkipWhenShuffling)
	}
	if track2.BookmarkTime != 1200000 {
		t.Errorf("BookmarkTime: got %d, want 1200000", track2.BookmarkTime)
	}

	if len(parsed.Playlists) != 2 {
		t.Fatalf("expected 2 playlists, got %d", len(parsed.Playlists))
	}

	masterPL := parsed.Playlists[0]
	if !masterPL.IsMaster {
		t.Error("first playlist should be master")
	}
	if len(masterPL.Tracks) != 2 {
		t.Errorf("master playlist: expected 2 tracks, got %d", len(masterPL.Tracks))
	}

	favPL := parsed.Playlists[1]
	if favPL.Name != "Favorites" {
		t.Errorf("playlist name: got %q, want %q", favPL.Name, "Favorites")
	}
	if len(favPL.Tracks) != 1 {
		t.Errorf("favorites: expected 1 track, got %d", len(favPL.Tracks))
	}
}

func TestEmptyDatabase(t *testing.T) {
	db := NewDatabase()
	data := db.Serialize()

	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Tracks) != 0 {
		t.Errorf("expected 0 tracks, got %d", len(parsed.Tracks))
	}
	if len(parsed.Playlists) != 1 {
		t.Errorf("expected 1 playlist (master), got %d", len(parsed.Playlists))
	}
}

func TestUnicodeStrings(t *testing.T) {
	db := NewDatabase()
	db.AddTrack(&Track{
		UniqueID:  1,
		Title:     "Für Elise",
		Artist:    "ベートーヴェン",
		Album:     "Classique 🎵",
		Path:      ":iPod_Control:Music:F00:TEST.mp3",
		MediaType: MediaTypeMusic,
	})

	data := db.Serialize()
	parsed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	track := parsed.Tracks[0]
	if track.Title != "Für Elise" {
		t.Errorf("Title: got %q, want %q", track.Title, "Für Elise")
	}
	if track.Artist != "ベートーヴェン" {
		t.Errorf("Artist: got %q, want %q", track.Artist, "ベートーヴェン")
	}
}
