package itunesdb

import "time"

const MacEpochDelta = 2082844800

func MacTimestamp(t time.Time) uint32 {
	if t.IsZero() {
		return 0
	}
	return uint32(t.Unix() + MacEpochDelta)
}

func FromMacTimestamp(ts uint32) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	return time.Unix(int64(ts)-MacEpochDelta, 0)
}

const (
	MediaTypeMusic        = 0x00000001
	MediaTypeVideo        = 0x00000002
	MediaTypePodcast      = 0x00000004
	MediaTypeVideoPodcast = 0x00000006
	MediaTypeAudiobook    = 0x00000008
	MediaTypeMusicVideo   = 0x00000020
	MediaTypeTVShow       = 0x00000040
	MediaTypeRingtone     = 0x00004000
)

type Database struct {
	Tracks    []*Track
	Playlists []*Playlist
	TZOffset  int32 // seconds east of UTC, from mhbd header 0x6C
}

type Track struct {
	UniqueID  uint32
	FileType  uint32
	MediaType uint32
	Size      uint32
	Duration  uint32
	BitRate   uint32
	SampleRate uint32
	PlayCount uint32
	BookmarkTime uint32
	LastPlayed   uint32

	TrackNumber uint16
	Year        uint16

	RememberPosition  uint8
	SkipWhenShuffling uint8

	Title  string
	Artist string
	Album  string
	Genre  string
	Path   string

	SourceID string

	DBID             uint64
	FiletypeKey      string
	VBR              bool
	Compilation      bool
	Rating           uint8
	Volume           int32
	StartTime        uint32
	StopTime         uint32
	SoundCheck       uint32
	DiscNumber       uint32
	TotalDiscs       uint32
	TotalTracks      uint32
	BPM              uint16
	ArtworkCount     uint16
	ArtworkSize      uint32
	MHIILink         uint32
	AlbumID          uint32
	ArtistID         uint32
	ComposerID       uint32
	SkipCount        uint32
	LastSkipped      time.Time
	DateAdded        time.Time
	DateReleased     time.Time
	LastModified     time.Time
	ExplicitFlag     uint16
	PodcastFlag      uint8
	HasLyrics        bool
	MovieFlag        uint8
	PlayedMark       int8
	Pregap           uint32
	Postgap          uint32
	SampleCount      uint64
	EncoderFlag      uint32
	GaplessData      uint32
	GaplessTrackFlag uint16
	GaplessAlbumFlag uint16
	SeasonNumber     uint32
	EpisodeNumber    uint32
	Checked          uint8
	AppRating        uint8
	UserID           uint32
	Unk144           uint16

	Composer         string
	AlbumArtist      string
	Comment          string
	FiletypeDesc     string
	SortArtist       string
	SortName         string
	SortAlbum        string
	SortAlbumArtist  string
	SortComposer     string
	Grouping         string
	Keywords         string
	Description      string
	Subtitle         string
	ShowName         string
	EpisodeID        string
	NetworkName      string
	SortShow         string
	ShowLocale       string
	PodcastEnclosureURL string
	PodcastRSSURL    string
	Category         string
	EQSetting        string
	Lyrics           string
}

type Playlist struct {
	Name     string
	IsMaster bool
	Tracks   []*Track

	PlaylistID  uint64
	SortOrder   uint32
	PodcastFlag uint8
	GroupFlag   uint8
	IsSmart     bool
	SmartPrefs  *SmartPlaylistPrefs
	SmartRules  *SmartPlaylistRules
	Mhsd5Type   uint16
}

type SmartPlaylistPrefs struct {
	LiveUpdate       bool
	CheckRules       bool
	CheckLimits      bool
	LimitType        uint8
	LimitSort        uint32
	LimitValue       uint32
	MatchCheckedOnly bool
}

type SmartPlaylistRule struct {
	FieldID  uint32
	ActionID uint32

	StringValue string

	FromValue uint64
	FromDate  int64
	FromUnits uint64
	ToValue   uint64
	ToDate    int64
	ToUnits   uint64

	Unk052 uint32
	Unk056 uint32
	Unk060 uint32
	Unk064 uint32
	Unk068 uint32
}

type SmartPlaylistRules struct {
	Conjunction string
	MatchAll    bool
	Rules       []SmartPlaylistRule
}

type AlbumInfo struct {
	Name       string
	AlbumID    uint32
	ArtistID   uint32
	TrackCount int
	SortName   string
}

type ArtistInfo struct {
	Name       string
	ArtistID   uint32
	TrackCount int
	SortName   string
}

func NewDatabase(name ...string) *Database {
	plName := "clickwheel"
	if len(name) > 0 && name[0] != "" {
		plName = name[0]
	}
	return &Database{
		Playlists: []*Playlist{
			{Name: plName, IsMaster: true},
		},
	}
}

func (db *Database) AddTrack(t *Track) {
	db.Tracks = append(db.Tracks, t)
	if t.MediaType == MediaTypePodcast {
		return
	}
	for _, pl := range db.Playlists {
		if pl.IsMaster {
			pl.Tracks = append(pl.Tracks, t)
			break
		}
	}
}

func (db *Database) RemoveTrack(sourceID string) *Track {
	for i, t := range db.Tracks {
		if t.SourceID == sourceID {
			db.Tracks = append(db.Tracks[:i], db.Tracks[i+1:]...)
			for _, pl := range db.Playlists {
				for j, pt := range pl.Tracks {
					if pt.SourceID == sourceID {
						pl.Tracks = append(pl.Tracks[:j], pl.Tracks[j+1:]...)
						break
					}
				}
			}
			return t
		}
	}
	return nil
}

func (db *Database) FindTrackBySourceID(sourceID string) *Track {
	for _, t := range db.Tracks {
		if t.SourceID == sourceID {
			return t
		}
	}
	return nil
}
