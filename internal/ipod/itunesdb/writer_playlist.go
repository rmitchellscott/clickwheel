package itunesdb

import (
	"encoding/binary"
	"math/rand"
	"time"
	"unicode/utf16"
)

const (
	mhypHeaderSize  = 184
	mhipHeaderSize  = 76
	mhlpHeaderSize  = 92
	splPrefBodySize = 132
	slstHeaderSize  = 136
)

const (
	SPLFieldString   = 1
	SPLFieldInt      = 2
	SPLFieldBoolean  = 3
	SPLFieldDate     = 4
	SPLFieldPlaylist = 5
	SPLFieldUnknown  = 6
	SPLFieldBinaryAnd = 7
)

var splFieldTypeMap = map[uint32]int{
	0x02: SPLFieldString, 0x03: SPLFieldString, 0x04: SPLFieldString,
	0x08: SPLFieldString, 0x09: SPLFieldString, 0x0E: SPLFieldString,
	0x12: SPLFieldString, 0x27: SPLFieldString, 0x36: SPLFieldString,
	0x37: SPLFieldString, 0x3E: SPLFieldString, 0x47: SPLFieldString,
	0x4E: SPLFieldString, 0x4F: SPLFieldString, 0x50: SPLFieldString,
	0x51: SPLFieldString, 0x52: SPLFieldString, 0x53: SPLFieldString,
	0x05: SPLFieldInt, 0x06: SPLFieldInt, 0x07: SPLFieldInt,
	0x0B: SPLFieldInt, 0x0C: SPLFieldInt, 0x0D: SPLFieldInt,
	0x16: SPLFieldInt, 0x18: SPLFieldInt, 0x19: SPLFieldInt,
	0x23: SPLFieldInt, 0x3F: SPLFieldInt, 0x44: SPLFieldInt,
	0x5A: SPLFieldInt, 0x39: SPLFieldInt,
	0x0A: SPLFieldDate, 0x10: SPLFieldDate, 0x17: SPLFieldDate,
	0x45: SPLFieldDate,
	0x1F: SPLFieldBoolean, 0x29: SPLFieldBoolean,
	0x28: SPLFieldPlaylist,
	0x3C: SPLFieldBinaryAnd,
}

func splFieldType(fieldID uint32) int {
	if ft, ok := splFieldTypeMap[fieldID]; ok {
		return ft
	}
	return SPLFieldUnknown
}


func WriteMHYP(pl *Playlist, trackIDs []uint32, id0x24 uint64, isMaster bool, tracks []*Track) []byte {
	return writeMHYPWithFormat(pl, trackIDs, id0x24, isMaster, tracks, false)
}

func writeMHYPWithFormat(pl *Playlist, trackIDs []uint32, id0x24 uint64, isMaster bool, tracks []*Track, podcastGrouped bool) []byte {
	playlistID := rand.Uint64()
	now := MacTimestamp(time.Now())

	mhodTitle := WriteMHODString(1, pl.Name)
	mhodPrefs := WriteMHODPlaylistPrefs()

	var items []byte
	itemCount := uint32(len(trackIDs))

	if podcastGrouped && pl.PodcastFlag == 1 {
		var mhipCount int
		items, mhipCount = WritePodcastMHIPs(pl, tracks)
		itemCount = uint32(mhipCount)
	} else {
		for i, tid := range trackIDs {
			items = append(items, WriteMHIP(tid, i)...)
		}
	}

	mhodCount := uint32(2)
	totalLen := uint32(mhypHeaderSize) + uint32(len(mhodTitle)) + uint32(len(mhodPrefs)) + uint32(len(items))

	buf := make([]byte, mhypHeaderSize)
	copy(buf[0:4], "mhyp")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], mhypHeaderSize)
	binary.LittleEndian.PutUint32(buf[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(buf[0x0C:0x10], mhodCount)
	binary.LittleEndian.PutUint32(buf[0x10:0x14], itemCount)

	if isMaster {
		binary.LittleEndian.PutUint32(buf[0x14:0x18], 1)
	}

	binary.LittleEndian.PutUint32(buf[0x18:0x1C], now)
	binary.LittleEndian.PutUint64(buf[0x1C:0x24], playlistID)
	binary.LittleEndian.PutUint16(buf[0x28:0x2A], 1)
	buf[0x2A] = pl.PodcastFlag

	if !isMaster {
		binary.LittleEndian.PutUint64(buf[0x3C:0x44], id0x24)
		binary.LittleEndian.PutUint64(buf[0x44:0x4C], playlistID)
	}

	binary.LittleEndian.PutUint32(buf[0x58:0x5C], now)

	result := append(buf, mhodTitle...)
	result = append(result, mhodPrefs...)
	return append(result, items...)
}

func WriteMasterPlaylist(name string, trackIDs []uint32, id0x24 uint64, tracks []*Track) []byte {
	if name == "" {
		name = "clickwheel"
	}
	pl := &Playlist{Name: name, IsMaster: true, Tracks: tracks}
	return WriteMHYP(pl, trackIDs, id0x24, true, tracks)
}

func WriteMHIP(trackID uint32, position int) []byte {
	posMHOD := WriteMHODPosition(position)

	totalLen := uint32(mhipHeaderSize) + uint32(len(posMHOD))

	buf := make([]byte, mhipHeaderSize)
	copy(buf[0:4], "mhip")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], mhipHeaderSize)
	binary.LittleEndian.PutUint32(buf[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(buf[0x0C:0x10], 1)
	binary.LittleEndian.PutUint32(buf[0x18:0x1C], trackID)

	return append(buf, posMHOD...)
}

func writeMHIPPodcast(childCount, groupFlag, groupID, trackID, groupRef uint32) []byte {
	buf := make([]byte, mhipHeaderSize)
	copy(buf[0:4], "mhip")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], mhipHeaderSize)
	binary.LittleEndian.PutUint32(buf[0x0C:0x10], childCount)
	binary.LittleEndian.PutUint32(buf[0x10:0x14], groupFlag)
	binary.LittleEndian.PutUint32(buf[0x14:0x18], groupID)
	binary.LittleEndian.PutUint32(buf[0x18:0x1C], trackID)
	binary.LittleEndian.PutUint32(buf[0x20:0x24], groupRef)
	return buf
}

func WritePodcastMHIPs(pl *Playlist, allTracks []*Track) ([]byte, int) {
	trackByID := make(map[uint32]*Track)
	for _, t := range allTracks {
		trackByID[t.UniqueID] = t
	}

	type group struct {
		name   string
		tracks []uint32
	}
	groupOrder := []string{}
	groups := map[string]*group{}

	for _, t := range pl.Tracks {
		album := t.Album
		if album == "" {
			album = t.ShowName
		}
		g, ok := groups[album]
		if !ok {
			g = &group{name: album}
			groups[album] = g
			groupOrder = append(groupOrder, album)
		}
		g.tracks = append(g.tracks, t.UniqueID)
	}

	var result []byte
	mhipCount := 0
	nextID := uint32(1)

	for _, name := range groupOrder {
		g := groups[name]
		groupID := nextID
		nextID++

		groupHeader := writeMHIPPodcast(1, 256, groupID, 0, 0)
		titleMHOD := WriteMHODString(1, g.name)
		binary.LittleEndian.PutUint32(groupHeader[0x08:0x0C], uint32(len(groupHeader))+uint32(len(titleMHOD)))
		result = append(result, groupHeader...)
		result = append(result, titleMHOD...)
		mhipCount++

		for _, tid := range g.tracks {
			mhipID := nextID
			nextID++
			memberMHIP := writeMHIPPodcast(1, 0, mhipID, tid, groupID)
			posMHOD := WriteMHODPosition(int(mhipID))
			binary.LittleEndian.PutUint32(memberMHIP[0x08:0x0C], uint32(len(memberMHIP))+uint32(len(posMHOD)))
			result = append(result, memberMHIP...)
			result = append(result, posMHOD...)
			mhipCount++
		}
	}

	return result, mhipCount
}

func WriteMHODPosition(position int) []byte {
	totalLen := uint32(44)
	buf := make([]byte, totalLen)
	copy(buf[0:4], "mhod")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], 24)
	binary.LittleEndian.PutUint32(buf[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(buf[0x0C:0x10], 100)
	binary.LittleEndian.PutUint32(buf[24:28], uint32(position))
	return buf
}

func WriteMHODPlaylistPrefs() []byte {
	totalLen := uint32(0x288)
	buf := make([]byte, totalLen)
	copy(buf[0:4], "mhod")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], 24)
	binary.LittleEndian.PutUint32(buf[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(buf[0x0C:0x10], 100)

	binary.LittleEndian.PutUint32(buf[0x30:0x34], 0x010084)
	binary.LittleEndian.PutUint32(buf[0x34:0x38], 0x05)
	binary.LittleEndian.PutUint32(buf[0x38:0x3C], 0x09)
	binary.LittleEndian.PutUint32(buf[0x3C:0x40], 0x03)
	binary.LittleEndian.PutUint32(buf[0x40:0x44], 0x120001)
	binary.LittleEndian.PutUint32(buf[0x4C:0x50], 0x640014)
	binary.LittleEndian.PutUint32(buf[0x50:0x54], 0x01)
	binary.LittleEndian.PutUint32(buf[0x5C:0x60], 0x320014)
	binary.LittleEndian.PutUint32(buf[0x60:0x64], 0x01)
	binary.LittleEndian.PutUint32(buf[0x6C:0x70], 0x5a0014)
	binary.LittleEndian.PutUint32(buf[0x70:0x74], 0x01)
	binary.LittleEndian.PutUint32(buf[0x7C:0x80], 0x500014)
	binary.LittleEndian.PutUint32(buf[0x80:0x84], 0x01)
	binary.LittleEndian.PutUint32(buf[0x8C:0x90], 0x7d0015)
	binary.LittleEndian.PutUint32(buf[0x90:0x94], 0x01)

	return buf
}

func WriteMHLP(playlistChunks [][]byte) []byte {
	var body []byte
	for _, chunk := range playlistChunks {
		body = append(body, chunk...)
	}

	buf := make([]byte, mhlpHeaderSize)
	copy(buf[0:4], "mhlp")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], mhlpHeaderSize)
	binary.LittleEndian.PutUint32(buf[0x08:0x0C], uint32(len(playlistChunks)))

	return append(buf, body...)
}

func WriteMHOD50(prefs *SmartPlaylistPrefs) []byte {
	body := make([]byte, splPrefBodySize)

	if prefs.LiveUpdate {
		body[0] = 1
	}
	if prefs.CheckRules {
		body[1] = 1
	}
	if prefs.CheckLimits {
		body[2] = 1
	}
	body[3] = prefs.LimitType

	body[4] = uint8(prefs.LimitSort & 0xFF)

	binary.LittleEndian.PutUint32(body[8:12], prefs.LimitValue)

	if prefs.MatchCheckedOnly {
		body[12] = 1
	}
	if prefs.LimitSort&0x80000000 != 0 {
		body[13] = 1
	}

	headerLen := uint32(24)
	totalLen := headerLen + uint32(splPrefBodySize)

	header := make([]byte, headerLen)
	copy(header[0:4], "mhod")
	binary.LittleEndian.PutUint32(header[0x04:0x08], headerLen)
	binary.LittleEndian.PutUint32(header[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(header[0x0C:0x10], 50)

	return append(header, body...)
}

func writeSPLRule(rule *SmartPlaylistRule) []byte {
	ft := splFieldType(rule.FieldID)

	var dataSection []byte

	if ft == SPLFieldString && rule.StringValue != "" {
		u16 := utf16.Encode([]rune(rule.StringValue))
		strBytes := make([]byte, len(u16)*2)
		for i, c := range u16 {
			binary.BigEndian.PutUint16(strBytes[i*2:i*2+2], c)
		}
		dataSection = strBytes
	} else {
		dataSection = make([]byte, 0x44)
		binary.BigEndian.PutUint64(dataSection[0x00:0x08], rule.FromValue)
		putBigInt64(dataSection[0x08:0x10], rule.FromDate)
		binary.BigEndian.PutUint64(dataSection[0x10:0x18], rule.FromUnits)
		binary.BigEndian.PutUint64(dataSection[0x18:0x20], rule.ToValue)
		putBigInt64(dataSection[0x20:0x28], rule.ToDate)
		binary.BigEndian.PutUint64(dataSection[0x28:0x30], rule.ToUnits)
		binary.BigEndian.PutUint32(dataSection[0x30:0x34], rule.Unk052)
		binary.BigEndian.PutUint32(dataSection[0x34:0x38], rule.Unk056)
		binary.BigEndian.PutUint32(dataSection[0x38:0x3C], rule.Unk060)
		binary.BigEndian.PutUint32(dataSection[0x3C:0x40], rule.Unk064)
		binary.BigEndian.PutUint32(dataSection[0x40:0x44], rule.Unk068)
	}

	ruleHeader := make([]byte, 56)
	binary.BigEndian.PutUint32(ruleHeader[0x00:0x04], rule.FieldID)
	binary.BigEndian.PutUint32(ruleHeader[0x04:0x08], rule.ActionID)
	binary.BigEndian.PutUint32(ruleHeader[0x34:0x38], uint32(len(dataSection)))

	return append(ruleHeader, dataSection...)
}

func putBigInt64(b []byte, v int64) {
	binary.BigEndian.PutUint64(b, uint64(v))
}

func WriteMHOD51(rules *SmartPlaylistRules) []byte {
	slstHeader := make([]byte, slstHeaderSize)
	copy(slstHeader[0:4], "SLst")
	binary.BigEndian.PutUint32(slstHeader[8:12], uint32(len(rules.Rules)))

	conjVal := uint32(0)
	if rules.Conjunction == "OR" {
		conjVal = 1
	}
	binary.BigEndian.PutUint32(slstHeader[12:16], conjVal)

	var rulesBytes []byte
	for i := range rules.Rules {
		rulesBytes = append(rulesBytes, writeSPLRule(&rules.Rules[i])...)
	}

	slstBody := append(slstHeader, rulesBytes...)

	headerLen := uint32(24)
	totalLen := headerLen + uint32(len(slstBody))

	header := make([]byte, headerLen)
	copy(header[0:4], "mhod")
	binary.LittleEndian.PutUint32(header[0x04:0x08], headerLen)
	binary.LittleEndian.PutUint32(header[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(header[0x0C:0x10], 51)

	return append(header, slstBody...)
}

func WriteMHOD102(rawBody []byte) []byte {
	headerLen := uint32(24)
	totalLen := headerLen + uint32(len(rawBody))

	header := make([]byte, headerLen)
	copy(header[0:4], "mhod")
	binary.LittleEndian.PutUint32(header[0x04:0x08], headerLen)
	binary.LittleEndian.PutUint32(header[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(header[0x0C:0x10], 102)

	return append(header, rawBody...)
}
