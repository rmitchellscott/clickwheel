package itunesdb

// Track struct fields that need to be added to types.go for full iOpenPod parity:
//   VBR, Compilation, Rating, Volume, StartTime, StopTime, SoundCheck, DiscNumber,
//   TotalDiscs, TotalTracks, BPM, ArtworkCount, ArtworkSize, MHIILink, AlbumID,
//   SkipCount, LastSkipped, DateAdded, DateReleased, LastModified, ExplicitFlag,
//   PodcastFlag, HasLyrics, Lyrics, MovieFlag, PlayedMark, Pregap, Postgap,
//   SampleCount, EncoderFlag, GaplessData, GaplessTrackFlag, GaplessAlbumFlag,
//   SeasonNumber, EpisodeNumber, Composer, AlbumArtist, Comment, FiletypeDesc,
//   SortArtist, SortName, SortAlbum, SortAlbumArtist, SortComposer, Grouping,
//   Keywords, Description, Subtitle, ShowName, EpisodeID, NetworkName, SortShow,
//   ShowLocale, PodcastEnclosureURL, PodcastRSSURL, Category, EQSetting,
//   Checked, AppRating, UserID, ArtistID, ComposerID, Unk144, DBID

import (
	"encoding/binary"
	"math"
	"math/rand"
	"time"
	"unicode/utf16"
)

var filetypeCodes = map[string]uint32{
	"mp3":  0x4D503320,
	"m4a":  0x4D344120,
	"m4p":  0x4D345020,
	"m4b":  0x4D344220,
	"m4v":  0x4D345620,
	"mp4":  0x4D503420,
	"wav":  0x57415620,
	"aif":  0x41494646,
	"aiff": 0x41494646,
	"aac":  0x41414320,
}

const (
	mhitHeaderSize = 0x270
)

func WriteMHIT(track *Track, trackID uint32, id0x24 uint64, caps *DeviceCapabilities) []byte {
	if track.DBID == 0 {
		track.DBID = rand.Uint64()
	}
	if track.DateAdded.IsZero() {
		track.DateAdded = time.Now()
	}

	filetypeCode := filetypeLookup(track.FiletypeKey)
	mhodData, mhodCount := writeTrackMHODs(track)
	totalLength := uint32(mhitHeaderSize) + uint32(len(mhodData))

	buf := make([]byte, mhitHeaderSize)

	copy(buf[0x00:0x04], "mhit")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], mhitHeaderSize)
	binary.LittleEndian.PutUint32(buf[0x08:0x0C], totalLength)
	binary.LittleEndian.PutUint32(buf[0x0C:0x10], uint32(mhodCount))

	binary.LittleEndian.PutUint32(buf[0x10:0x14], trackID)
	binary.LittleEndian.PutUint32(buf[0x14:0x18], 1)
	binary.LittleEndian.PutUint32(buf[0x18:0x1C], filetypeCode)

	if track.VBR {
		buf[0x1C] = 1
	}
	if track.FiletypeKey == "mp3" {
		buf[0x1D] = 1
	}
	if track.Compilation {
		buf[0x1E] = 1
	}
	r := track.Rating
	if r > 100 {
		r = 100
	}
	buf[0x1F] = r

	timeMod := track.LastModified
	if timeMod.IsZero() {
		timeMod = track.DateAdded
	}
	binary.LittleEndian.PutUint32(buf[0x20:0x24], MacTimestamp(timeMod))
	binary.LittleEndian.PutUint32(buf[0x24:0x28], track.Size)
	binary.LittleEndian.PutUint32(buf[0x28:0x2C], track.Duration)
	binary.LittleEndian.PutUint32(buf[0x2C:0x30], uint32(track.TrackNumber))

	binary.LittleEndian.PutUint32(buf[0x30:0x34], uint32(track.TotalTracks))
	binary.LittleEndian.PutUint32(buf[0x34:0x38], uint32(track.Year))
	binary.LittleEndian.PutUint32(buf[0x38:0x3C], track.BitRate)
	binary.LittleEndian.PutUint32(buf[0x3C:0x40], track.SampleRate<<16)

	binary.LittleEndian.PutUint32(buf[0x40:0x44], uint32(int32(track.Volume)))
	binary.LittleEndian.PutUint32(buf[0x44:0x48], track.StartTime)
	binary.LittleEndian.PutUint32(buf[0x48:0x4C], track.StopTime)
	binary.LittleEndian.PutUint32(buf[0x4C:0x50], track.SoundCheck)

	binary.LittleEndian.PutUint32(buf[0x50:0x54], track.PlayCount)
	binary.LittleEndian.PutUint32(buf[0x54:0x58], 0)
	binary.LittleEndian.PutUint32(buf[0x58:0x5C], track.LastPlayed)
	binary.LittleEndian.PutUint32(buf[0x5C:0x60], uint32(track.DiscNumber))

	binary.LittleEndian.PutUint32(buf[0x60:0x64], uint32(track.TotalDiscs))
	binary.LittleEndian.PutUint32(buf[0x64:0x68], track.UserID)
	binary.LittleEndian.PutUint32(buf[0x68:0x6C], MacTimestamp(track.DateAdded))
	binary.LittleEndian.PutUint32(buf[0x6C:0x70], track.BookmarkTime)

	binary.LittleEndian.PutUint64(buf[0x70:0x78], track.DBID)

	buf[0x78] = track.Checked
	buf[0x79] = track.AppRating
	binary.LittleEndian.PutUint16(buf[0x7A:0x7C], track.BPM)
	binary.LittleEndian.PutUint16(buf[0x7C:0x7E], track.ArtworkCount)
	binary.LittleEndian.PutUint16(buf[0x7E:0x80], codecHint(track.FiletypeKey))

	binary.LittleEndian.PutUint32(buf[0x80:0x84], track.ArtworkSize)
	binary.LittleEndian.PutUint32(buf[0x88:0x8C], math.Float32bits(float32(track.SampleRate)))
	binary.LittleEndian.PutUint32(buf[0x8C:0x90], macTimestampOrZero(track.DateReleased))

	binary.LittleEndian.PutUint16(buf[0x90:0x92], track.Unk144)
	binary.LittleEndian.PutUint16(buf[0x92:0x94], track.ExplicitFlag)

	binary.LittleEndian.PutUint32(buf[0x9C:0xA0], track.SkipCount)
	binary.LittleEndian.PutUint32(buf[0xA0:0xA4], macTimestampOrZero(track.LastSkipped))

	if track.ArtworkCount > 0 {
		buf[0xA4] = 1
	} else {
		buf[0xA4] = 2
	}
	buf[0xA5] = track.SkipWhenShuffling
	buf[0xA6] = track.RememberPosition
	buf[0xA7] = track.PodcastFlag

	binary.LittleEndian.PutUint64(buf[0xA8:0xB0], track.DBID)

	hasLyrics := track.HasLyrics || track.Lyrics != ""
	if hasLyrics {
		buf[0xB0] = 1
	}

	movieFlag := track.MovieFlag
	if movieFlag == 0 {
		switch track.MediaType {
		case MediaTypeVideo, MediaTypeMusicVideo, MediaTypeTVShow, MediaTypeVideoPodcast:
			movieFlag = 1
		}
	}
	buf[0xB1] = movieFlag

	if track.PlayedMark >= 0 {
		buf[0xB2] = uint8(track.PlayedMark)
	} else if track.PlayCount > 0 {
		buf[0xB2] = 0x01
	} else {
		buf[0xB2] = 0x02
	}

	gapless := caps == nil || caps.SupportsGapless
	if gapless {
		binary.LittleEndian.PutUint32(buf[0xB8:0xBC], track.Pregap)
		binary.LittleEndian.PutUint64(buf[0xBC:0xC4], track.SampleCount)
		binary.LittleEndian.PutUint32(buf[0xC8:0xCC], track.Postgap)
		binary.LittleEndian.PutUint32(buf[0xF8:0xFC], track.GaplessData)
		binary.LittleEndian.PutUint16(buf[0x100:0x102], track.GaplessTrackFlag)
		binary.LittleEndian.PutUint16(buf[0x102:0x104], track.GaplessAlbumFlag)
	}

	binary.LittleEndian.PutUint32(buf[0xCC:0xD0], track.EncoderFlag)

	mediaType := track.MediaType
	if caps != nil {
		if !caps.SupportsVideo {
			switch mediaType {
			case MediaTypeVideo, MediaTypeMusicVideo, MediaTypeTVShow:
				mediaType = MediaTypeMusic
			case MediaTypeVideoPodcast:
				mediaType = MediaTypePodcast
			}
		}
		if !caps.SupportsPodcast {
			switch mediaType {
			case MediaTypePodcast, MediaTypeVideoPodcast:
				mediaType = MediaTypeMusic
			}
		}
	}
	binary.LittleEndian.PutUint32(buf[0xD0:0xD4], mediaType)

	binary.LittleEndian.PutUint32(buf[0xD4:0xD8], track.SeasonNumber)
	binary.LittleEndian.PutUint32(buf[0xD8:0xDC], track.EpisodeNumber)

	binary.LittleEndian.PutUint32(buf[0x120:0x124], track.AlbumID)
	binary.LittleEndian.PutUint64(buf[0x124:0x12C], id0x24)
	binary.LittleEndian.PutUint32(buf[0x12C:0x130], track.Size)

	binary.LittleEndian.PutUint64(buf[0x134:0x13C], 0x808080808080)

	binary.LittleEndian.PutUint32(buf[0x160:0x164], track.MHIILink)

	binary.LittleEndian.PutUint32(buf[0x168:0x16C], 1)

	binary.LittleEndian.PutUint32(buf[0x1E0:0x1E4], track.ArtistID)
	binary.LittleEndian.PutUint32(buf[0x1F4:0x1F8], track.ComposerID)

	return append(buf, mhodData...)
}

func WriteMHODString(mhodType int, value string) []byte {
	if value == "" {
		return nil
	}

	u16 := utf16.Encode([]rune(value))
	strBytes := make([]byte, len(u16)*2)
	for i, c := range u16 {
		binary.LittleEndian.PutUint16(strBytes[i*2:i*2+2], c)
	}

	headerLen := uint32(24)
	typeHeaderLen := uint32(16)
	totalLen := headerLen + typeHeaderLen + uint32(len(strBytes))

	buf := make([]byte, headerLen+typeHeaderLen)
	copy(buf[0:4], "mhod")
	binary.LittleEndian.PutUint32(buf[4:8], headerLen)
	binary.LittleEndian.PutUint32(buf[8:12], totalLen)
	binary.LittleEndian.PutUint32(buf[12:16], uint32(mhodType))

	binary.LittleEndian.PutUint32(buf[headerLen:headerLen+4], 1)
	binary.LittleEndian.PutUint32(buf[headerLen+4:headerLen+8], uint32(len(strBytes)))
	binary.LittleEndian.PutUint32(buf[headerLen+8:headerLen+12], 1)
	binary.LittleEndian.PutUint32(buf[headerLen+12:headerLen+16], 0)

	return append(buf, strBytes...)
}

func WriteMHODPodcastURL(mhodType int, url string) []byte {
	if url == "" {
		return nil
	}

	strBytes := []byte(url)
	headerLen := uint32(24)
	totalLen := headerLen + uint32(len(strBytes))

	buf := make([]byte, headerLen)
	copy(buf[0:4], "mhod")
	binary.LittleEndian.PutUint32(buf[4:8], headerLen)
	binary.LittleEndian.PutUint32(buf[8:12], totalLen)
	binary.LittleEndian.PutUint32(buf[12:16], uint32(mhodType))

	return append(buf, strBytes...)
}

func writeTrackMHODs(track *Track) ([]byte, int) {
	type mhodEntry struct {
		typ   int
		value string
		pod   bool
	}

	entries := []mhodEntry{
		{1, track.Title, false},
		{2, track.Path, false},
		{4, track.Artist, false},
		{3, track.Album, false},
		{5, track.Genre, false},
		{22, track.AlbumArtist, false},
		{12, track.Composer, false},
		{8, trackComment(track), false},
		{6, track.FiletypeDesc, false},
		{9, track.Category, false},
		{14, track.Description, false},
		{18, track.Subtitle, false},
		{19, track.ShowName, false},
		{20, track.EpisodeID, false},
		{21, track.NetworkName, false},
		{24, track.Keywords, false},
		{23, track.SortArtist, false},
		{27, track.SortName, false},
		{28, track.SortAlbum, false},
		{29, track.SortAlbumArtist, false},
		{30, track.SortComposer, false},
		{31, track.SortShow, false},
		{25, track.ShowLocale, false},
		{13, track.Grouping, false},
		{15, track.PodcastEnclosureURL, true},
		{16, track.PodcastRSSURL, true},
		{10, track.Lyrics, false},
		{7, track.EQSetting, false},
	}

	var data []byte
	count := 0

	for _, e := range entries {
		if e.value == "" {
			continue
		}
		var chunk []byte
		if e.pod {
			chunk = WriteMHODPodcastURL(e.typ, e.value)
		} else {
			chunk = WriteMHODString(e.typ, e.value)
		}
		if chunk != nil {
			data = append(data, chunk...)
			count++
		}
	}

	return data, count
}

const sourceIDPrefix = "clickwheel:"

func trackComment(t *Track) string {
	if t.SourceID != "" {
		return sourceIDPrefix + t.SourceID
	}
	return t.Comment
}

func filetypeLookup(key string) uint32 {
	if code, ok := filetypeCodes[key]; ok {
		return code
	}
	return filetypeCodes["mp3"]
}

func codecHint(filetype string) uint16 {
	switch filetype {
	case "wav", "aif", "aiff":
		return 0x0000
	case "m4b":
		return 0x0001
	default:
		return 0xFFFF
	}
}

func macTimestampOrZero(t time.Time) uint32 {
	if t.IsZero() {
		return 0
	}
	return MacTimestamp(t)
}
