package itunesdb

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"unicode/utf16"
)

func Parse(data []byte) (*Database, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("data too short for iTunesDB")
	}

	if string(data[0:4]) != "mhbd" {
		return nil, fmt.Errorf("invalid iTunesDB magic: %s", string(data[0:4]))
	}

	headerLen := le32(data, 4)
	totalLen := le32(data, 8)
	if uint32(len(data)) < totalLen {
		totalLen = uint32(len(data))
	}
	numDataSets := le32(data, 20)

	db := &Database{}
	pos := headerLen

	for i := uint32(0); i < numDataSets && pos+12 <= totalLen; i++ {
		if string(data[pos:pos+4]) != "mhsd" {
			break
		}

		dsTotalLen := le32(data, pos+8)
		dsType := le32(data, pos+12)
		dsHeaderLen := le32(data, pos+4)

		if pos+dsTotalLen > totalLen {
			dsTotalLen = totalLen - pos
		}

		dsEnd := pos + dsTotalLen
		childStart := pos + dsHeaderLen

		switch dsType {
		case 1:
			db.Tracks = parseTrackList(data, childStart, dsEnd)
		case 2:
			db.Playlists = append(db.Playlists,
				parsePlaylistList(data, childStart, dsEnd, db.Tracks)...)
		case 4:
			skipListChunk(data, childStart, dsEnd)
		case 8:
			skipListChunk(data, childStart, dsEnd)
		}

		pos += dsTotalLen
	}

	return db, nil
}

func parseTrackList(data []byte, start, end uint32) []*Track {
	if start+12 > end || start+4 > uint32(len(data)) {
		return nil
	}
	if string(data[start:start+4]) != "mhlt" {
		return nil
	}

	headerLen := le32(data, start+4)
	trackCount := le32(data, start+8)
	tracks := make([]*Track, 0, trackCount)
	pos := start + headerLen

	for i := uint32(0); i < trackCount && pos+12 <= end; i++ {
		if pos+4 > uint32(len(data)) || string(data[pos:pos+4]) != "mhit" {
			break
		}
		track, tLen := parseTrack(data, pos)
		if track != nil {
			tracks = append(tracks, track)
		}
		pos += tLen
	}

	return tracks
}

func parseTrack(data []byte, offset uint32) (*Track, uint32) {
	d := data[offset:]
	if len(d) < 0x9C {
		return nil, 12
	}

	headerLen := le32(d, 4)
	totalLen := le32(d, 8)
	mhodCount := le32(d, 12)
	hl := int(headerLen)

	t := &Track{}

	t.UniqueID = le32(d, 0x10)

	if hl > 0x1C {
		t.FileType = le32(d, 0x18)
	}
	if hl > 0x1F {
		if d[0x1C] != 0 {
			t.VBR = true
		}
		if d[0x1E] != 0 {
			t.Compilation = true
		}
		t.Rating = d[0x1F]
	}
	if hl >= 0x24 {
		macTS := le32(d, 0x20)
		if macTS != 0 {
			t.LastModified = FromMacTimestamp(macTS)
		}
	}
	if hl >= 0x28 {
		t.Size = le32(d, 0x24)
	}
	if hl >= 0x2C {
		t.Duration = le32(d, 0x28)
	}
	if hl >= 0x30 {
		t.TrackNumber = uint16(le32(d, 0x2C))
	}
	if hl >= 0x34 {
		t.TotalTracks = le32(d, 0x30)
	}
	if hl >= 0x38 {
		t.Year = uint16(le32(d, 0x34))
	}
	if hl >= 0x3C {
		t.BitRate = le32(d, 0x38)
	}
	if hl >= 0x40 {
		t.SampleRate = le32(d, 0x3C) >> 16
	}
	if hl >= 0x44 {
		t.Volume = int32(le32(d, 0x40))
	}
	if hl >= 0x48 {
		t.StartTime = le32(d, 0x44)
	}
	if hl >= 0x4C {
		t.StopTime = le32(d, 0x48)
	}
	if hl >= 0x50 {
		t.SoundCheck = le32(d, 0x4C)
	}
	if hl >= 0x54 {
		t.PlayCount = le32(d, 0x50)
	}
	if hl >= 0x5C {
		t.LastPlayed = le32(d, 0x58)
	}
	if hl >= 0x60 {
		t.DiscNumber = le32(d, 0x5C)
	}
	if hl >= 0x64 {
		t.TotalDiscs = le32(d, 0x60)
	}
	if hl >= 0x68 {
		t.UserID = le32(d, 0x64)
	}
	if hl >= 0x6C {
		macTS := le32(d, 0x68)
		if macTS != 0 {
			t.DateAdded = FromMacTimestamp(macTS)
		}
	}
	if hl >= 0x70 {
		t.BookmarkTime = le32(d, 0x6C)
	}
	if hl >= 0x78 {
		t.DBID = le64(d, 0x70)
	}
	if hl >= 0x79 {
		t.Checked = d[0x78]
	}
	if hl >= 0x7A {
		t.AppRating = d[0x79]
	}
	if hl >= 0x7C {
		t.BPM = binary.LittleEndian.Uint16(d[0x7A:0x7C])
	}
	if hl >= 0x7E {
		t.ArtworkCount = binary.LittleEndian.Uint16(d[0x7C:0x7E])
	}
	if hl >= 0x84 {
		t.ArtworkSize = le32(d, 0x80)
	}
	if hl >= 0x8C {
		bits := le32(d, 0x88)
		sr2 := math.Float32frombits(bits)
		if sr2 > 0 && t.SampleRate == 0 {
			t.SampleRate = uint32(sr2)
		}
	}
	if hl >= 0x90 {
		macTS := le32(d, 0x8C)
		if macTS != 0 {
			t.DateReleased = FromMacTimestamp(macTS)
		}
	}
	if hl >= 0x92 {
		t.Unk144 = binary.LittleEndian.Uint16(d[0x90:0x92])
	}
	if hl >= 0x94 {
		t.ExplicitFlag = binary.LittleEndian.Uint16(d[0x92:0x94])
	}

	if hl >= 0xF4 {
		t.SkipCount = le32(d, 0x9C)

		macTS := le32(d, 0xA0)
		if macTS != 0 {
			t.LastSkipped = FromMacTimestamp(macTS)
		}

		if hl > 0xA5 {
			t.SkipWhenShuffling = d[0xA5]
		}
		if hl > 0xA6 {
			t.RememberPosition = d[0xA6]
		}
		if hl > 0xA7 {
			t.PodcastFlag = d[0xA7]
		}
		if hl > 0xB0 {
			t.HasLyrics = d[0xB0] != 0
		}
		if hl > 0xB1 {
			t.MovieFlag = d[0xB1]
		}
		if hl > 0xB2 {
			t.PlayedMark = int8(d[0xB2])
		}
		if hl >= 0xBC {
			t.Pregap = le32(d, 0xB8)
		}
		if hl >= 0xC4 {
			t.SampleCount = le64(d, 0xBC)
		}
		if hl >= 0xCC {
			t.Postgap = le32(d, 0xC8)
		}
		if hl >= 0xD0 {
			t.EncoderFlag = le32(d, 0xCC)
		}
		if hl >= 0xD4 {
			t.MediaType = le32(d, 0xD0)
		}
		if hl >= 0xD8 {
			t.SeasonNumber = le32(d, 0xD4)
		}
		if hl >= 0xDC {
			t.EpisodeNumber = le32(d, 0xD8)
		}
	} else {
		if hl > 0xA5 {
			t.SkipWhenShuffling = d[0xA5]
		}
		if hl > 0xA6 {
			t.RememberPosition = d[0xA6]
		}
		if hl >= 0xD4 {
			t.MediaType = le32(d, 0xD0)
		}
	}

	if hl >= 0x148 {
		t.GaplessData = le32(d, 0xF8)
		t.GaplessTrackFlag = binary.LittleEndian.Uint16(d[0x100:0x102])
		t.GaplessAlbumFlag = binary.LittleEndian.Uint16(d[0x102:0x104])
	}

	if hl >= 0x184 {
		t.AlbumID = le32(d, 0x120)
		t.MHIILink = le32(d, 0x160)
	}

	if hl >= 0x248 {
		t.ArtistID = le32(d, 0x1E0)
		t.ComposerID = le32(d, 0x1F4)
	}

	pos := headerLen
	for i := uint32(0); i < mhodCount && pos+12 <= totalLen; i++ {
		if pos+4 > uint32(len(d)) || string(d[pos:pos+4]) != "mhod" {
			break
		}

		mhodTotalLen := le32(d, pos+8)
		if mhodTotalLen < 12 {
			break
		}
		mhodType := le32(d, pos+12)

		assignTrackMHOD(t, mhodType, readMHODString(d, pos, mhodType))

		pos += mhodTotalLen
	}

	return t, totalLen
}

func readMHODString(data []byte, pos uint32, mhodType uint32) string {
	if pos+24 > uint32(len(data)) {
		return ""
	}

	mhodHeaderLen := le32(data, pos+4)
	mhodTotalLen := le32(data, pos+8)

	if mhodType == 15 || mhodType == 16 {
		bodyStart := pos + mhodHeaderLen
		bodyEnd := pos + mhodTotalLen
		if bodyEnd > uint32(len(data)) {
			bodyEnd = uint32(len(data))
		}
		if bodyStart >= bodyEnd {
			return ""
		}
		s := string(data[bodyStart:bodyEnd])
		for len(s) > 0 && s[len(s)-1] == 0 {
			s = s[:len(s)-1]
		}
		return s
	}

	if mhodType == 17 || mhodType == 32 || mhodType >= 50 {
		return ""
	}

	subStart := pos + mhodHeaderLen
	if subStart+16 > uint32(len(data)) {
		return ""
	}

	encoding := le32(data, subStart)
	strLen := le32(data, subStart+4)
	strStart := subStart + 16

	if strStart+strLen > uint32(len(data)) {
		strLen = uint32(len(data)) - strStart
	}
	if strLen == 0 {
		return ""
	}

	strData := data[strStart : strStart+strLen]

	if encoding == 2 {
		return string(strData)
	}

	return decodeUTF16LE(strData)
}

func assignTrackMHOD(t *Track, mhodType uint32, value string) {
	if value == "" {
		return
	}
	switch mhodType {
	case 1:
		t.Title = value
	case 2:
		t.Path = value
	case 3:
		t.Album = value
	case 4:
		t.Artist = value
	case 5:
		t.Genre = value
	case 6:
		t.FiletypeDesc = value
	case 7:
		t.EQSetting = value
	case 8:
		if strings.HasPrefix(value, sourceIDPrefix) {
			t.SourceID = strings.TrimPrefix(value, sourceIDPrefix)
		} else {
			t.Comment = value
		}
	case 9:
		t.Category = value
	case 10:
		t.Lyrics = value
	case 12:
		t.Composer = value
	case 13:
		t.Grouping = value
	case 14:
		t.Description = value
	case 15:
		t.PodcastEnclosureURL = value
	case 16:
		t.PodcastRSSURL = value
	case 18:
		t.Subtitle = value
	case 19:
		t.ShowName = value
	case 20:
		t.EpisodeID = value
	case 21:
		t.NetworkName = value
	case 22:
		t.AlbumArtist = value
	case 23:
		t.SortArtist = value
	case 24:
		t.Keywords = value
	case 25:
		t.ShowLocale = value
	case 27:
		t.SortName = value
	case 28:
		t.SortAlbum = value
	case 29:
		t.SortAlbumArtist = value
	case 30:
		t.SortComposer = value
	case 31:
		t.SortShow = value
	}
}

func parsePlaylistList(data []byte, start, end uint32, tracks []*Track) []*Playlist {
	if start+12 > end || start+4 > uint32(len(data)) {
		return nil
	}
	if string(data[start:start+4]) != "mhlp" {
		return nil
	}

	headerLen := le32(data, start+4)
	playlistCount := le32(data, start+8)

	trackByUID := make(map[uint32]*Track, len(tracks))
	for _, t := range tracks {
		trackByUID[t.UniqueID] = t
	}

	playlists := make([]*Playlist, 0, playlistCount)
	pos := start + headerLen

	for i := uint32(0); i < playlistCount && pos+12 <= end; i++ {
		if pos+4 > uint32(len(data)) || string(data[pos:pos+4]) != "mhyp" {
			break
		}
		pl, tLen := parsePlaylist(data, pos, trackByUID)
		if pl != nil {
			playlists = append(playlists, pl)
		}
		pos += tLen
	}

	return playlists
}

func parsePlaylist(data []byte, offset uint32, trackByUID map[uint32]*Track) (*Playlist, uint32) {
	d := data[offset:]
	if len(d) < 48 {
		return nil, 12
	}

	headerLen := le32(d, 4)
	totalLen := le32(d, 8)
	mhodCount := le32(d, 12)
	mhipCount := le32(d, 16)

	pl := &Playlist{
		IsMaster: d[20] == 1,
		Tracks:   make([]*Track, 0, mhipCount),
	}

	if headerLen >= 0x24 {
		pl.PlaylistID = le64(d, 0x1C)
	}
	if headerLen >= 0x2C {
		pl.PodcastFlag = d[0x2A]
		pl.GroupFlag = d[0x2B]
	}
	if headerLen >= 0x30 {
		pl.SortOrder = le32(d, 0x2C)
	}

	pos := headerLen
	for i := uint32(0); i < mhodCount && pos+12 <= totalLen; i++ {
		if pos+4 > uint32(len(d)) || string(d[pos:pos+4]) != "mhod" {
			break
		}

		mhodTotalLen := le32(d, pos+8)
		if mhodTotalLen < 12 {
			break
		}
		mhodType := le32(d, pos+12)

		switch mhodType {
		case 1:
			pl.Name = readMHODString(d, pos, mhodType)
		case 50:
			pl.IsSmart = true
			pl.SmartPrefs = parseMHOD50(d, pos)
		case 51:
			pl.SmartRules = parseMHOD51(d, pos)
		}

		pos += mhodTotalLen
	}

	for i := uint32(0); i < mhipCount && pos+12 <= totalLen; i++ {
		if pos+4 > uint32(len(d)) || string(d[pos:pos+4]) != "mhip" {
			break
		}

		mhipTotalLen := le32(d, pos+8)
		if mhipTotalLen < 12 {
			break
		}

		mhipHeaderLen := le32(d, pos+4)
		if mhipHeaderLen >= 28 {
			trackUID := le32(d, pos+24)
			if t, ok := trackByUID[trackUID]; ok {
				pl.Tracks = append(pl.Tracks, t)
			}
		}

		pos += mhipTotalLen
	}

	return pl, totalLen
}

func parseMHOD50(data []byte, pos uint32) *SmartPlaylistPrefs {
	mhodHeaderLen := le32(data, pos+4)
	mhodTotalLen := le32(data, pos+8)

	bodyStart := pos + mhodHeaderLen
	bodyLen := mhodTotalLen - mhodHeaderLen
	if bodyStart+12 > uint32(len(data)) || bodyLen < 12 {
		return nil
	}

	body := data[bodyStart:]
	prefs := &SmartPlaylistPrefs{
		LiveUpdate:  body[0] != 0,
		CheckRules:  body[1] != 0,
		CheckLimits: body[2] != 0,
		LimitType:   body[3],
	}

	limitSortRaw := body[4]
	prefs.LimitValue = le32(body, 8)

	if bodyLen >= 13 {
		prefs.MatchCheckedOnly = body[12] != 0
	}

	limitSort := uint32(limitSortRaw)
	if bodyLen >= 14 && body[13] != 0 {
		limitSort |= 0x80000000
	}
	prefs.LimitSort = limitSort

	return prefs
}

func parseMHOD51(data []byte, pos uint32) *SmartPlaylistRules {
	mhodHeaderLen := le32(data, pos+4)
	mhodTotalLen := le32(data, pos+8)

	bodyStart := pos + mhodHeaderLen
	bodyLen := mhodTotalLen - mhodHeaderLen
	if bodyStart+136 > uint32(len(data)) || bodyLen < 136 {
		return nil
	}

	body := data[bodyStart:]
	if string(body[0:4]) != "SLst" {
		return nil
	}

	ruleCount := binary.BigEndian.Uint32(body[8:12])
	conjunction := binary.BigEndian.Uint32(body[12:16])

	rules := &SmartPlaylistRules{
		Conjunction: "AND",
	}
	if conjunction != 0 {
		rules.Conjunction = "OR"
	}

	ruleOffset := uint32(136)
	for i := uint32(0); i < ruleCount; i++ {
		if ruleOffset+56 > bodyLen {
			break
		}

		fieldID := binary.BigEndian.Uint32(body[ruleOffset : ruleOffset+4])
		actionID := binary.BigEndian.Uint32(body[ruleOffset+4 : ruleOffset+8])
		dataLen := binary.BigEndian.Uint32(body[ruleOffset+0x34 : ruleOffset+0x38])

		rule := SmartPlaylistRule{
			FieldID:  fieldID,
			ActionID: actionID,
		}

		dataOffset := ruleOffset + 56
		ft := splFieldType(fieldID)

		if ft == SPLFieldString {
			if dataOffset+dataLen <= bodyLen && dataLen > 0 {
				strData := body[dataOffset : dataOffset+dataLen]
				u16 := make([]uint16, dataLen/2)
				for j := range u16 {
					u16[j] = binary.BigEndian.Uint16(strData[j*2 : j*2+2])
				}
				rule.StringValue = string(utf16.Decode(u16))
			}
		} else {
			if dataOffset+0x44 <= bodyLen {
				rd := body[dataOffset:]
				rule.FromValue = binary.BigEndian.Uint64(rd[0:8])
				rule.FromDate = int64(binary.BigEndian.Uint64(rd[8:16]))
				rule.FromUnits = binary.BigEndian.Uint64(rd[16:24])
				rule.ToValue = binary.BigEndian.Uint64(rd[24:32])
				rule.ToDate = int64(binary.BigEndian.Uint64(rd[32:40]))
				rule.ToUnits = binary.BigEndian.Uint64(rd[40:48])
				rule.Unk052 = binary.BigEndian.Uint32(rd[48:52])
				rule.Unk056 = binary.BigEndian.Uint32(rd[52:56])
				rule.Unk060 = binary.BigEndian.Uint32(rd[56:60])
				rule.Unk064 = binary.BigEndian.Uint32(rd[60:64])
				rule.Unk068 = binary.BigEndian.Uint32(rd[64:68])
			}
		}

		rules.Rules = append(rules.Rules, rule)
		ruleOffset += 56 + dataLen
	}

	return rules
}

func skipListChunk(data []byte, start, end uint32) {
	if start+12 > end || start+4 > uint32(len(data)) {
		return
	}
}

func decodeUTF16LE(data []byte) string {
	if len(data) < 2 {
		return ""
	}
	n := len(data) / 2
	u16 := make([]uint16, n)
	for i := 0; i < n; i++ {
		u16[i] = binary.LittleEndian.Uint16(data[i*2 : i*2+2])
	}
	return string(utf16.Decode(u16))
}

func le32(data []byte, offset uint32) uint32 {
	return binary.LittleEndian.Uint32(data[offset : offset+4])
}

func le64(data []byte, offset uint32) uint64 {
	return binary.LittleEndian.Uint64(data[offset : offset+8])
}
