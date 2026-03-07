package itunesdb

import (
	"encoding/binary"
	"math/rand"
	"sort"
	"strings"
	"time"
)

const (
	mhbdHeaderSize   = 244
	defaultDBVersion = uint32(0x4F)
	defaultPlatform  = 2
)

func SerializeDatabase(db *Database, caps *DeviceCapabilities) []byte {
	dbID := rand.Uint64()
	id0x24 := rand.Uint64()

	for i, t := range db.Tracks {
		t.UniqueID = uint32(i + 1)
	}

	albums := BuildAlbumList(db.Tracks)
	artists := BuildArtistList(db.Tracks)

	albumMap := buildAlbumMap(db.Tracks, albums)
	artistMap := make(map[string]uint32)
	for _, a := range artists {
		artistMap[strings.ToLower(a.Name)] = a.ArtistID
	}

	for _, t := range db.Tracks {
		t.AlbumID = albumMap[t.Album+"\x00"+t.Artist]
		if t.Artist != "" {
			t.ArtistID = artistMap[strings.ToLower(t.Artist)]
		}
	}

	var trackChunks [][]byte
	for _, t := range db.Tracks {
		trackChunks = append(trackChunks, WriteMHIT(t, t.UniqueID, id0x24, caps))
	}
	mhsdType1 := WriteMHSD(1, WriteMHLT(trackChunks))

	trackIDs := make([]uint32, len(db.Tracks))
	for i, t := range db.Tracks {
		trackIDs[i] = t.UniqueID
	}

	playlistBody := buildPlaylistBody(db, id0x24, trackIDs)
	mhsdType3 := WriteMHSD(3, playlistBody)
	mhsdType2 := WriteMHSD(2, playlistBody)

	mhsdType4 := WriteMHSD(4, WriteMHLA(albums))
	mhsdType8 := WriteMHSD(8, WriteMHLI(artists))
	mhsdType5 := WriteMHSD(5, WriteMHLP(nil))
	mhsdType6 := WriteMHSDEmptyStub(6)
	mhsdType10 := WriteMHSDEmptyStub(10)

	includePodcasts := caps == nil || caps.SupportsPodcast
	var datasets [][]byte
	datasets = append(datasets, mhsdType1)
	if includePodcasts {
		datasets = append(datasets, mhsdType3)
	}
	datasets = append(datasets, mhsdType2)
	datasets = append(datasets, mhsdType4)
	datasets = append(datasets, mhsdType8)
	datasets = append(datasets, mhsdType6)
	datasets = append(datasets, mhsdType10)
	datasets = append(datasets, mhsdType5)

	var body []byte
	for _, ds := range datasets {
		body = append(body, ds...)
	}

	version := defaultDBVersion
	if caps != nil && caps.DBVersion != 0 {
		version = caps.DBVersion
	}

	header := buildMHBDHeader(len(datasets), dbID, id0x24, version, len(body))
	result := append(header, body...)

	if caps != nil && caps.FirewireID != "" {
		switch caps.Checksum {
		case ChecksumHash58:
			result = WriteHash58(result, caps.FirewireID)
		case ChecksumHashAB:
			result = WriteHashAB(result, caps.FirewireID, nil)
		}
	}

	return result
}

func (db *Database) Serialize() []byte {
	return SerializeDatabase(db, nil)
}

func buildMHBDHeader(numDataSets int, dbID, id0x24 uint64, version uint32, bodyLen int) []byte {
	totalLen := uint32(mhbdHeaderSize) + uint32(bodyLen)

	buf := make([]byte, mhbdHeaderSize)
	copy(buf[0x00:0x04], "mhbd")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], mhbdHeaderSize)
	binary.LittleEndian.PutUint32(buf[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(buf[0x0C:0x10], 1)
	binary.LittleEndian.PutUint32(buf[0x10:0x14], version)
	binary.LittleEndian.PutUint32(buf[0x14:0x18], uint32(numDataSets))
	binary.LittleEndian.PutUint64(buf[0x18:0x20], dbID)
	binary.LittleEndian.PutUint16(buf[0x20:0x22], defaultPlatform)
	binary.LittleEndian.PutUint64(buf[0x24:0x2C], id0x24)

	_, tzOffset := time.Now().Zone()
	binary.LittleEndian.PutUint32(buf[0x6C:0x70], uint32(int32(tzOffset)))

	copy(buf[0x46:0x48], "en")
	binary.LittleEndian.PutUint64(buf[0x48:0x50], dbID)

	return buf
}

func buildPlaylistBody(db *Database, id0x24 uint64, trackIDs []uint32) []byte {
	var playlistChunks [][]byte

	for _, pl := range db.Playlists {
		if pl.IsMaster {
			playlistChunks = append(playlistChunks,
				WriteMasterPlaylist(trackIDs, id0x24, db.Tracks))
			break
		}
	}

	for _, pl := range db.Playlists {
		if pl.IsMaster {
			continue
		}
		plTrackIDs := resolvePlaylistTrackIDs(pl, db.Tracks)
		playlistChunks = append(playlistChunks,
			WriteMHYP(pl, plTrackIDs, id0x24, false, db.Tracks))
	}

	return WriteMHLP(playlistChunks)
}

func resolvePlaylistTrackIDs(pl *Playlist, allTracks []*Track) []uint32 {
	var ids []uint32
	for _, pt := range pl.Tracks {
		for _, t := range allTracks {
			if t == pt {
				ids = append(ids, t.UniqueID)
				break
			}
		}
	}
	return ids
}

func buildAlbumMap(tracks []*Track, albums []*AlbumInfo) map[string]uint32 {
	type ak struct{ album, artist string }
	seen := make(map[ak]bool)
	var order []ak
	for _, t := range tracks {
		k := ak{t.Album, t.Artist}
		if !seen[k] {
			seen[k] = true
			order = append(order, k)
		}
	}

	sort.Slice(order, func(i, j int) bool {
		if order[i].album != order[j].album {
			return strings.ToLower(order[i].album) < strings.ToLower(order[j].album)
		}
		return strings.ToLower(order[i].artist) < strings.ToLower(order[j].artist)
	})

	m := make(map[string]uint32, len(order))
	for i, k := range order {
		if i < len(albums) {
			m[k.album+"\x00"+k.artist] = albums[i].AlbumID
		}
	}
	return m
}
