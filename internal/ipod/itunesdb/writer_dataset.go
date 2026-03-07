package itunesdb

import (
	"encoding/binary"
	"math/rand"
	"sort"
	"strings"
	"unicode"
	"unicode/utf16"
)

func WriteMHSD(dsType int, body []byte) []byte {
	headerLen := uint32(96)
	totalLen := headerLen + uint32(len(body))

	buf := make([]byte, headerLen)
	copy(buf[0:4], "mhsd")
	binary.LittleEndian.PutUint32(buf[4:8], headerLen)
	binary.LittleEndian.PutUint32(buf[8:12], totalLen)
	binary.LittleEndian.PutUint32(buf[12:16], uint32(dsType))

	return append(buf, body...)
}

func WriteMHSDEmptyStub(dsType int) []byte {
	mhlt := make([]byte, 92)
	copy(mhlt[0:4], "mhlt")
	binary.LittleEndian.PutUint32(mhlt[4:8], 92)

	return WriteMHSD(dsType, mhlt)
}

func WriteMHLT(trackChunks [][]byte) []byte {
	buf := make([]byte, 92)
	copy(buf[0:4], "mhlt")
	binary.LittleEndian.PutUint32(buf[4:8], 92)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(len(trackChunks)))

	for _, chunk := range trackChunks {
		buf = append(buf, chunk...)
	}
	return buf
}

func WriteMHLA(albums []*AlbumInfo) []byte {
	buf := make([]byte, 92)
	copy(buf[0:4], "mhla")
	binary.LittleEndian.PutUint32(buf[4:8], 92)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(len(albums)))

	for _, album := range albums {
		buf = append(buf, WriteMHIA(album)...)
	}
	return buf
}

func WriteMHIA(album *AlbumInfo) []byte {
	const headerSize = 88

	var children []byte
	childCount := 0

	if album.Name != "" {
		children = append(children, WriteMHODString(200, album.Name)...)
		childCount++
	}
	if album.SortName != "" {
		children = append(children, WriteMHODString(201, album.SortName)...)
		childCount++
	}

	totalLen := uint32(headerSize + len(children))
	buf := make([]byte, headerSize)
	copy(buf[0:4], "mhia")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], headerSize)
	binary.LittleEndian.PutUint32(buf[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(buf[0x0C:0x10], uint32(childCount))
	binary.LittleEndian.PutUint32(buf[0x10:0x14], album.AlbumID)
	binary.LittleEndian.PutUint64(buf[0x14:0x1C], rand.Uint64()|1)
	binary.LittleEndian.PutUint32(buf[0x1C:0x20], 2)

	return append(buf, children...)
}

func BuildAlbumList(tracks []*Track) []*AlbumInfo {
	type albumKey struct {
		name   string
		artist string
	}

	seen := make(map[albumKey]*AlbumInfo)
	var order []albumKey

	for _, t := range tracks {
		artist := t.AlbumArtist
		if artist == "" {
			artist = t.Artist
		}
		if t.Compilation {
			artist = ""
		}
		key := albumKey{name: t.Album, artist: artist}
		if info, ok := seen[key]; ok {
			info.TrackCount++
		} else {
			seen[key] = &AlbumInfo{
				Name:       t.Album,
				TrackCount: 1,
			}
			order = append(order, key)
		}
	}

	sort.Slice(order, func(i, j int) bool {
		ni, nj := strings.ToLower(order[i].name), strings.ToLower(order[j].name)
		if ni != nj {
			return ni < nj
		}
		return strings.ToLower(order[i].artist) < strings.ToLower(order[j].artist)
	})

	albums := make([]*AlbumInfo, 0, len(order))
	for i, key := range order {
		info := seen[key]
		info.AlbumID = uint32(i + 1)
		albums = append(albums, info)
	}
	return albums
}

func WriteMHLI(artists []*ArtistInfo) []byte {
	buf := make([]byte, 92)
	copy(buf[0:4], "mhli")
	binary.LittleEndian.PutUint32(buf[4:8], 92)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(len(artists)))

	for _, artist := range artists {
		buf = append(buf, WriteMHII(artist)...)
	}
	return buf
}

func WriteMHII(artist *ArtistInfo) []byte {
	const headerSize = 80

	var children []byte
	childCount := 0
	if artist.Name != "" {
		children = WriteMHODString(300, artist.Name)
		childCount = 1
	}

	totalLen := uint32(headerSize + len(children))
	buf := make([]byte, headerSize)
	copy(buf[0:4], "mhii")
	binary.LittleEndian.PutUint32(buf[0x04:0x08], headerSize)
	binary.LittleEndian.PutUint32(buf[0x08:0x0C], totalLen)
	binary.LittleEndian.PutUint32(buf[0x0C:0x10], uint32(childCount))
	binary.LittleEndian.PutUint32(buf[0x10:0x14], artist.ArtistID)
	binary.LittleEndian.PutUint64(buf[0x14:0x1C], rand.Uint64()|1)
	binary.LittleEndian.PutUint32(buf[0x1C:0x20], 2)

	return append(buf, children...)
}

func BuildArtistList(tracks []*Track) []*ArtistInfo {
	seen := make(map[string]*ArtistInfo)
	var order []string

	for _, t := range tracks {
		if t.Artist == "" {
			continue
		}
		key := strings.ToLower(t.Artist)
		if info, ok := seen[key]; ok {
			info.TrackCount++
		} else {
			seen[key] = &ArtistInfo{
				Name:       t.Artist,
				TrackCount: 1,
			}
			order = append(order, key)
		}
	}

	sort.Strings(order)

	artists := make([]*ArtistInfo, 0, len(order))
	for i, key := range order {
		info := seen[key]
		info.ArtistID = uint32(i + 1)
		artists = append(artists, info)
	}
	return artists
}

const (
	sortTitle       = 0x03
	sortAlbum       = 0x04
	sortArtist      = 0x05
	sortGenre       = 0x07
	sortComposer    = 0x12
	sortShow        = 0x1D
	sortSeason      = 0x1E
	sortEpisode     = 0x1F
	sortAlbumArtist = 0x23
)

var baseSortTypes = []int{sortTitle, sortAlbum, sortArtist, sortGenre, sortComposer}
var videoSortTypes = []int{sortShow, sortSeason, sortEpisode}

func WriteLibraryIndices(tracks []*Track, caps *DeviceCapabilities) ([]byte, int) {
	if len(tracks) == 0 {
		return nil, 0
	}

	sortTypes := make([]int, len(baseSortTypes))
	copy(sortTypes, baseSortTypes)
	if caps != nil {
		if caps.SupportsVideo {
			sortTypes = append(sortTypes, videoSortTypes...)
		}
		sortTypes = append(sortTypes, sortAlbumArtist)
	}

	var result []byte
	mhodCount := 0

	for _, st := range sortTypes {
		data52, jumpEntries := writeMHODType52(tracks, st)
		result = append(result, data52...)
		mhodCount++

		data53 := writeMHODType53(st, jumpEntries)
		result = append(result, data53...)
		mhodCount++
	}

	return result, mhodCount
}

type jumpEntry struct {
	letter uint16
	start  uint32
	count  uint32
}

func writeMHODType52(tracks []*Track, sortType int) ([]byte, []jumpEntry) {
	numTracks := len(tracks)

	type indexedTrack struct {
		keys  []string
		nums  []int
		index int
		track *Track
	}

	indexed := make([]indexedTrack, numTracks)
	for i, t := range tracks {
		keys, nums := dsGetSortFields(t, sortType)
		indexed[i] = indexedTrack{keys: keys, nums: nums, index: i, track: t}
	}

	sort.SliceStable(indexed, func(a, b int) bool {
		ia, ib := indexed[a], indexed[b]
		for k := 0; k < len(ia.keys) && k < len(ib.keys); k++ {
			if ia.keys[k] != ib.keys[k] {
				return ia.keys[k] < ib.keys[k]
			}
		}
		for k := 0; k < len(ia.nums) && k < len(ib.nums); k++ {
			if ia.nums[k] != ib.nums[k] {
				return ia.nums[k] < ib.nums[k]
			}
		}
		return false
	})

	totalLen := uint32(4*numTracks + 72)
	header := make([]byte, 24)
	copy(header[0:4], "mhod")
	binary.LittleEndian.PutUint32(header[4:8], 24)
	binary.LittleEndian.PutUint32(header[8:12], totalLen)
	binary.LittleEndian.PutUint32(header[12:16], 52)

	bodyHeader := make([]byte, 48)
	binary.LittleEndian.PutUint32(bodyHeader[0:4], uint32(sortType))
	binary.LittleEndian.PutUint32(bodyHeader[4:8], uint32(numTracks))

	indicesData := make([]byte, 4*numTracks)
	for i, it := range indexed {
		binary.LittleEndian.PutUint32(indicesData[i*4:i*4+4], uint32(it.index))
	}

	var entries []jumpEntry
	lastLetter := uint16(0xFFFF)
	for pos, it := range indexed {
		letter := dsGetJumpLetter(it.track, sortType)
		if letter != lastLetter {
			entries = append(entries, jumpEntry{letter: letter, start: uint32(pos), count: 1})
			lastLetter = letter
		} else if len(entries) > 0 {
			entries[len(entries)-1].count++
		}
	}

	result := append(header, bodyHeader...)
	return append(result, indicesData...), entries
}

func writeMHODType53(sortType int, entries []jumpEntry) []byte {
	numEntries := len(entries)
	totalLen := uint32(12*numEntries + 40)

	header := make([]byte, 24)
	copy(header[0:4], "mhod")
	binary.LittleEndian.PutUint32(header[4:8], 24)
	binary.LittleEndian.PutUint32(header[8:12], totalLen)
	binary.LittleEndian.PutUint32(header[12:16], 53)

	bodyHeader := make([]byte, 16)
	binary.LittleEndian.PutUint32(bodyHeader[0:4], uint32(sortType))
	binary.LittleEndian.PutUint32(bodyHeader[4:8], uint32(numEntries))

	entriesData := make([]byte, 12*numEntries)
	for i, e := range entries {
		off := i * 12
		binary.LittleEndian.PutUint16(entriesData[off:off+2], e.letter)
		binary.LittleEndian.PutUint16(entriesData[off+2:off+4], 0)
		binary.LittleEndian.PutUint32(entriesData[off+4:off+8], e.start)
		binary.LittleEndian.PutUint32(entriesData[off+8:off+12], e.count)
	}

	result := append(header, bodyHeader...)
	return append(result, entriesData...)
}

func dsSortKey(s string) string {
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "the ") {
		s = s[4:]
	}
	return strings.ToLower(s)
}

func dsJumpLetter(s string) uint16 {
	if s == "" {
		return uint16('0')
	}
	for _, ch := range s {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			if unicode.IsDigit(ch) {
				return uint16('0')
			}
			encoded := utf16.Encode([]rune{unicode.ToUpper(ch)})
			if len(encoded) > 0 {
				return encoded[0]
			}
			return uint16('0')
		}
	}
	return uint16('0')
}

func dsOrStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func dsGetSortFields(t *Track, sortType int) ([]string, []int) {
	title := dsSortKey(dsOrStr(t.SortName, t.Title))
	album := dsSortKey(dsOrStr(t.SortAlbum, t.Album))
	artist := dsSortKey(dsOrStr(t.SortArtist, t.Artist))
	genre := dsSortKey(t.Genre)
	composer := dsSortKey(dsOrStr(t.SortComposer, t.Composer))
	trackNr := int(t.TrackNumber)
	discNr := int(t.DiscNumber)

	switch sortType {
	case sortTitle:
		return []string{title}, nil
	case sortAlbum:
		return []string{album}, []int{discNr, trackNr}
	case sortArtist:
		return []string{artist, album}, []int{discNr, trackNr}
	case sortGenre:
		return []string{genre, artist, album}, []int{discNr, trackNr}
	case sortComposer:
		return []string{composer, album}, []int{discNr, trackNr}
	case sortShow:
		show := dsSortKey(dsOrStr(t.SortShow, t.ShowName))
		return []string{show}, []int{int(t.SeasonNumber), int(t.EpisodeNumber)}
	case sortSeason:
		show := dsSortKey(dsOrStr(t.SortShow, t.ShowName))
		return []string{show}, []int{int(t.SeasonNumber), int(t.EpisodeNumber)}
	case sortEpisode:
		show := dsSortKey(dsOrStr(t.SortShow, t.ShowName))
		return []string{show}, []int{int(t.EpisodeNumber), int(t.SeasonNumber)}
	case sortAlbumArtist:
		aa := dsSortKey(dsOrStr(t.SortAlbumArtist, dsOrStr(t.AlbumArtist, dsOrStr(t.SortArtist, t.Artist))))
		return []string{aa, album}, []int{discNr, trackNr}
	default:
		return []string{title}, nil
	}
}

func dsGetJumpLetter(t *Track, sortType int) uint16 {
	switch sortType {
	case sortTitle:
		return dsJumpLetter(dsOrStr(t.SortName, t.Title))
	case sortAlbum:
		return dsJumpLetter(dsOrStr(t.SortAlbum, t.Album))
	case sortArtist:
		return dsJumpLetter(dsOrStr(t.SortArtist, t.Artist))
	case sortGenre:
		return dsJumpLetter(t.Genre)
	case sortComposer:
		return dsJumpLetter(dsOrStr(t.SortComposer, t.Composer))
	case sortShow:
		return dsJumpLetter(dsOrStr(t.SortShow, t.ShowName))
	case sortSeason:
		if t.SeasonNumber > 0 {
			return dsJumpLetter("0")
		}
		return uint16('0')
	case sortEpisode:
		if t.EpisodeNumber > 0 {
			return dsJumpLetter("0")
		}
		return uint16('0')
	case sortAlbumArtist:
		return dsJumpLetter(dsOrStr(t.SortAlbumArtist, dsOrStr(t.AlbumArtist, dsOrStr(t.SortArtist, t.Artist))))
	default:
		return dsJumpLetter(dsOrStr(t.SortName, t.Title))
	}
}
