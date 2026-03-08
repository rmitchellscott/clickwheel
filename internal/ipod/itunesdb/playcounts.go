package itunesdb

import (
	"encoding/binary"
)

type PlayCountEntry struct {
	PlayCount    uint32
	LastPlayed   uint32
	BookmarkTime uint32
	Rating       uint8
	SkipCount    uint32
	LastSkipped  uint32
}

func ParsePlayCounts(data []byte) ([]PlayCountEntry, error) {
	if len(data) < 16 {
		return nil, nil
	}

	entrySize := binary.LittleEndian.Uint32(data[8:12])
	entryCount := binary.LittleEndian.Uint32(data[12:16])

	headerSize := uint32(16)
	if len(data) >= 20 {
		hs := binary.LittleEndian.Uint32(data[4:8])
		if hs >= 16 && hs <= uint32(len(data)) {
			headerSize = hs
		}
	}

	if entrySize == 0 || entryCount == 0 {
		return nil, nil
	}

	entries := make([]PlayCountEntry, 0, entryCount)
	pos := headerSize

	for i := uint32(0); i < entryCount; i++ {
		if pos+entrySize > uint32(len(data)) {
			break
		}
		e := data[pos : pos+entrySize]
		var entry PlayCountEntry

		if entrySize >= 4 {
			entry.PlayCount = binary.LittleEndian.Uint32(e[0:4])
		}
		if entrySize >= 8 {
			entry.LastPlayed = binary.LittleEndian.Uint32(e[4:8])
		}
		if entrySize >= 12 {
			entry.BookmarkTime = binary.LittleEndian.Uint32(e[8:12])
		}
		if entrySize >= 16 {
			entry.Rating = e[12]
		}
		if entrySize >= 24 {
			entry.SkipCount = binary.LittleEndian.Uint32(e[16:20])
		}
		if entrySize >= 28 {
			entry.LastSkipped = binary.LittleEndian.Uint32(e[20:24])
		}

		entries = append(entries, entry)
		pos += entrySize
	}

	return entries, nil
}

func MergePlayCounts(db *Database, entries []PlayCountEntry) {
	if len(entries) == 0 {
		return
	}

	for i, entry := range entries {
		if i >= len(db.Tracks) {
			break
		}
		t := db.Tracks[i]

		t.PlayCount += entry.PlayCount

		if entry.LastPlayed != 0 && entry.LastPlayed > t.LastPlayed {
			t.LastPlayed = entry.LastPlayed
		}

		if entry.BookmarkTime != 0 {
			t.BookmarkTime = entry.BookmarkTime
		}

		if entry.Rating != 0 {
			t.Rating = entry.Rating
		}

		t.SkipCount += entry.SkipCount
		if entry.LastSkipped != 0 {
			ls := FromMacTimestamp(entry.LastSkipped)
			if t.LastSkipped.IsZero() || ls.After(t.LastSkipped) {
				t.LastSkipped = ls
			}
		}
	}
}
