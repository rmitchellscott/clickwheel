package itunesdb

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

type ChecksumType int

const (
	ChecksumNone        ChecksumType = 0
	ChecksumHash58      ChecksumType = 1
	ChecksumHash72      ChecksumType = 2
	ChecksumHashAB      ChecksumType = 3
	ChecksumUnsupported ChecksumType = 98
	ChecksumUnknown     ChecksumType = 99
)

var ChecksumMHBDScheme = map[ChecksumType]int{
	ChecksumNone:   0,
	ChecksumHash58: 1,
	ChecksumHash72: 2,
	ChecksumHashAB: 4,
}

var MHBDSchemeToChecksum = map[int]ChecksumType{
	0: ChecksumNone,
	1: ChecksumHash58,
	2: ChecksumHash72,
	4: ChecksumHashAB,
}

type DeviceCapabilities struct {
	Checksum              ChecksumType
	IsShuffle             bool
	ShadowDBVersion       int
	SupportsCompressedDB  bool
	SupportsVideo         bool
	SupportsPodcast       bool
	SupportsGapless       bool
	SupportsArtwork       bool
	SupportsPhoto         bool
	SupportsChapterImage  bool
	SupportsSparseArtwork bool
	MusicDirs             int
	UsesSQLiteDB          bool
	DBVersion             uint32
	ByteOrder             string
	HasScreen             bool
	FirewireID            string
	ModelName             string
	HashABCalc            HashABCalculator
	HashInfo              *HashInfo
}

type modelInfo struct {
	Family     string
	Generation string
	Capacity   string
	Color      string
}

type familyGen struct {
	Family     string
	Generation string
}

var modelNumberRegexp = regexp.MustCompile(`^(M[A-Z]?\d{3,4})`)

var ipodModels = map[string]modelInfo{
	"MB029": {"iPod Classic", "1st Gen", "80GB", "Silver"},
	"MB147": {"iPod Classic", "1st Gen", "80GB", "Black"},
	"MB145": {"iPod Classic", "1st Gen", "160GB", "Silver"},
	"MB150": {"iPod Classic", "1st Gen", "160GB", "Black"},
	"MB562": {"iPod Classic", "2nd Gen", "120GB", "Silver"},
	"MB565": {"iPod Classic", "2nd Gen", "120GB", "Black"},
	"MC293": {"iPod Classic", "3rd Gen", "160GB", "Silver"},
	"MC297": {"iPod Classic", "3rd Gen", "160GB", "Black"},

	"M8513": {"iPod", "1st Gen", "5GB", "White"},
	"M8541": {"iPod", "1st Gen", "5GB", "White"},
	"M8697": {"iPod", "1st Gen", "5GB", "White"},
	"M8709": {"iPod", "1st Gen", "10GB", "White"},

	"M8737": {"iPod", "2nd Gen", "10GB", "White"},
	"M8740": {"iPod", "2nd Gen", "10GB", "White"},
	"M8738": {"iPod", "2nd Gen", "20GB", "White"},
	"M8741": {"iPod", "2nd Gen", "20GB", "White"},

	"M8976": {"iPod", "3rd Gen", "10GB", "White"},
	"M8946": {"iPod", "3rd Gen", "15GB", "White"},
	"M8948": {"iPod", "3rd Gen", "30GB", "White"},
	"M9244": {"iPod", "3rd Gen", "20GB", "White"},
	"M9245": {"iPod", "3rd Gen", "40GB", "White"},
	"M9460": {"iPod", "3rd Gen", "15GB", "White"},

	"M9268": {"iPod", "4th Gen", "40GB", "White"},
	"M9282": {"iPod", "4th Gen", "20GB", "White"},
	"ME436": {"iPod", "4th Gen", "40GB", "White"},
	"M9787": {"iPod U2", "4th Gen", "20GB", "Black"},

	"M9585": {"iPod Photo", "4th Gen", "40GB", "White"},
	"M9586": {"iPod Photo", "4th Gen", "60GB", "White"},
	"M9829": {"iPod Photo", "4th Gen", "30GB", "White"},
	"M9830": {"iPod Photo", "4th Gen", "60GB", "White"},
	"MA079": {"iPod Photo", "4th Gen", "20GB", "White"},
	"MA127": {"iPod U2", "4th Gen", "20GB", "Black"},
	"MS492": {"iPod Photo", "4th Gen", "30GB", "White"},
	"MA215": {"iPod Photo", "4th Gen", "20GB", "White"},

	"MA002": {"iPod Video", "5th Gen", "30GB", "White"},
	"MA003": {"iPod Video", "5th Gen", "60GB", "White"},
	"MA146": {"iPod Video", "5th Gen", "30GB", "Black"},
	"MA147": {"iPod Video", "5th Gen", "60GB", "Black"},
	"MA452": {"iPod Video U2", "5th Gen", "30GB", "Black"},

	"MA444": {"iPod Video", "5.5th Gen", "30GB", "White"},
	"MA446": {"iPod Video", "5.5th Gen", "30GB", "Black"},
	"MA448": {"iPod Video", "5.5th Gen", "80GB", "White"},
	"MA450": {"iPod Video", "5.5th Gen", "80GB", "Black"},
	"MA664": {"iPod Video U2", "5.5th Gen", "30GB", "Black"},

	"M9160": {"iPod Mini", "1st Gen", "4GB", "Silver"},
	"M9434": {"iPod Mini", "1st Gen", "4GB", "Green"},
	"M9435": {"iPod Mini", "1st Gen", "4GB", "Pink"},
	"M9436": {"iPod Mini", "1st Gen", "4GB", "Blue"},
	"M9437": {"iPod Mini", "1st Gen", "4GB", "Gold"},

	"M9800": {"iPod Mini", "2nd Gen", "4GB", "Silver"},
	"M9801": {"iPod Mini", "2nd Gen", "6GB", "Silver"},
	"M9802": {"iPod Mini", "2nd Gen", "4GB", "Blue"},
	"M9803": {"iPod Mini", "2nd Gen", "6GB", "Blue"},
	"M9804": {"iPod Mini", "2nd Gen", "4GB", "Pink"},
	"M9805": {"iPod Mini", "2nd Gen", "6GB", "Pink"},
	"M9806": {"iPod Mini", "2nd Gen", "4GB", "Green"},
	"M9807": {"iPod Mini", "2nd Gen", "6GB", "Green"},

	"MA004": {"iPod Nano", "1st Gen", "2GB", "White"},
	"MA005": {"iPod Nano", "1st Gen", "4GB", "White"},
	"MA099": {"iPod Nano", "1st Gen", "2GB", "Black"},
	"MA107": {"iPod Nano", "1st Gen", "4GB", "Black"},
	"MA350": {"iPod Nano", "1st Gen", "1GB", "White"},
	"MA352": {"iPod Nano", "1st Gen", "1GB", "Black"},

	"MA426": {"iPod Nano", "2nd Gen", "4GB", "Silver"},
	"MA428": {"iPod Nano", "2nd Gen", "4GB", "Blue"},
	"MA477": {"iPod Nano", "2nd Gen", "2GB", "Silver"},
	"MA487": {"iPod Nano", "2nd Gen", "4GB", "Green"},
	"MA489": {"iPod Nano", "2nd Gen", "4GB", "Pink"},
	"MA497": {"iPod Nano", "2nd Gen", "8GB", "Black"},
	"MA725": {"iPod Nano", "2nd Gen", "4GB", "Red"},
	"MA726": {"iPod Nano", "2nd Gen", "8GB", "Red"},
	"MA899": {"iPod Nano", "2nd Gen", "8GB", "Red"},

	"MA978": {"iPod Nano", "3rd Gen", "4GB", "Silver"},
	"MA980": {"iPod Nano", "3rd Gen", "8GB", "Silver"},
	"MB249": {"iPod Nano", "3rd Gen", "8GB", "Blue"},
	"MB253": {"iPod Nano", "3rd Gen", "8GB", "Green"},
	"MB257": {"iPod Nano", "3rd Gen", "8GB", "Red"},
	"MB261": {"iPod Nano", "3rd Gen", "8GB", "Black"},
	"MB453": {"iPod Nano", "3rd Gen", "8GB", "Pink"},

	"MB480": {"iPod Nano", "4th Gen", "4GB", "Silver"},
	"MB651": {"iPod Nano", "4th Gen", "4GB", "Blue"},
	"MB654": {"iPod Nano", "4th Gen", "4GB", "Pink"},
	"MB657": {"iPod Nano", "4th Gen", "4GB", "Purple"},
	"MB660": {"iPod Nano", "4th Gen", "4GB", "Orange"},
	"MB663": {"iPod Nano", "4th Gen", "4GB", "Green"},
	"MB666": {"iPod Nano", "4th Gen", "4GB", "Yellow"},
	"MB598": {"iPod Nano", "4th Gen", "8GB", "Silver"},
	"MB732": {"iPod Nano", "4th Gen", "8GB", "Blue"},
	"MB735": {"iPod Nano", "4th Gen", "8GB", "Pink"},
	"MB739": {"iPod Nano", "4th Gen", "8GB", "Purple"},
	"MB742": {"iPod Nano", "4th Gen", "8GB", "Orange"},
	"MB745": {"iPod Nano", "4th Gen", "8GB", "Green"},
	"MB748": {"iPod Nano", "4th Gen", "8GB", "Yellow"},
	"MB751": {"iPod Nano", "4th Gen", "8GB", "Red"},
	"MB754": {"iPod Nano", "4th Gen", "8GB", "Black"},
	"MB903": {"iPod Nano", "4th Gen", "16GB", "Silver"},
	"MB905": {"iPod Nano", "4th Gen", "16GB", "Blue"},
	"MB907": {"iPod Nano", "4th Gen", "16GB", "Pink"},
	"MB909": {"iPod Nano", "4th Gen", "16GB", "Purple"},
	"MB911": {"iPod Nano", "4th Gen", "16GB", "Orange"},
	"MB913": {"iPod Nano", "4th Gen", "16GB", "Green"},
	"MB915": {"iPod Nano", "4th Gen", "16GB", "Yellow"},
	"MB917": {"iPod Nano", "4th Gen", "16GB", "Red"},
	"MB918": {"iPod Nano", "4th Gen", "16GB", "Black"},

	"MC027": {"iPod Nano", "5th Gen", "8GB", "Silver"},
	"MC031": {"iPod Nano", "5th Gen", "8GB", "Black"},
	"MC034": {"iPod Nano", "5th Gen", "8GB", "Purple"},
	"MC037": {"iPod Nano", "5th Gen", "8GB", "Blue"},
	"MC040": {"iPod Nano", "5th Gen", "8GB", "Green"},
	"MC043": {"iPod Nano", "5th Gen", "8GB", "Yellow"},
	"MC046": {"iPod Nano", "5th Gen", "8GB", "Orange"},
	"MC049": {"iPod Nano", "5th Gen", "8GB", "Red"},
	"MC050": {"iPod Nano", "5th Gen", "8GB", "Pink"},
	"MC060": {"iPod Nano", "5th Gen", "16GB", "Silver"},
	"MC062": {"iPod Nano", "5th Gen", "16GB", "Black"},
	"MC064": {"iPod Nano", "5th Gen", "16GB", "Purple"},
	"MC066": {"iPod Nano", "5th Gen", "16GB", "Blue"},
	"MC068": {"iPod Nano", "5th Gen", "16GB", "Green"},
	"MC070": {"iPod Nano", "5th Gen", "16GB", "Yellow"},
	"MC072": {"iPod Nano", "5th Gen", "16GB", "Orange"},
	"MC074": {"iPod Nano", "5th Gen", "16GB", "Red"},
	"MC075": {"iPod Nano", "5th Gen", "16GB", "Pink"},

	"MC525": {"iPod Nano", "6th Gen", "8GB", "Silver"},
	"MC688": {"iPod Nano", "6th Gen", "8GB", "Graphite"},
	"MC689": {"iPod Nano", "6th Gen", "8GB", "Blue"},
	"MC690": {"iPod Nano", "6th Gen", "8GB", "Green"},
	"MC691": {"iPod Nano", "6th Gen", "8GB", "Orange"},
	"MC692": {"iPod Nano", "6th Gen", "8GB", "Pink"},
	"MC693": {"iPod Nano", "6th Gen", "8GB", "Red"},
	"MC526": {"iPod Nano", "6th Gen", "16GB", "Silver"},
	"MC694": {"iPod Nano", "6th Gen", "16GB", "Graphite"},
	"MC695": {"iPod Nano", "6th Gen", "16GB", "Blue"},
	"MC696": {"iPod Nano", "6th Gen", "16GB", "Green"},
	"MC697": {"iPod Nano", "6th Gen", "16GB", "Orange"},
	"MC698": {"iPod Nano", "6th Gen", "16GB", "Pink"},
	"MC699": {"iPod Nano", "6th Gen", "16GB", "Red"},

	"MD475": {"iPod Nano", "7th Gen", "16GB", "Pink"},
	"MD476": {"iPod Nano", "7th Gen", "16GB", "Yellow"},
	"MD477": {"iPod Nano", "7th Gen", "16GB", "Blue"},
	"MD478": {"iPod Nano", "7th Gen", "16GB", "Green"},
	"MD479": {"iPod Nano", "7th Gen", "16GB", "Purple"},
	"MD480": {"iPod Nano", "7th Gen", "16GB", "Silver"},
	"MD481": {"iPod Nano", "7th Gen", "16GB", "Slate"},
	"MD744": {"iPod Nano", "7th Gen", "16GB", "Red"},
	"ME971": {"iPod Nano", "7th Gen", "16GB", "Space Gray"},
	"MKMV2": {"iPod Nano", "7th Gen", "16GB", "Pink"},
	"MKMX2": {"iPod Nano", "7th Gen", "16GB", "Gold"},
	"MKN02": {"iPod Nano", "7th Gen", "16GB", "Blue"},
	"MKN22": {"iPod Nano", "7th Gen", "16GB", "Silver"},
	"MKN52": {"iPod Nano", "7th Gen", "16GB", "Space Gray"},
	"MKN72": {"iPod Nano", "7th Gen", "16GB", "Red"},

	"M9724": {"iPod Shuffle", "1st Gen", "512MB", "White"},
	"M9725": {"iPod Shuffle", "1st Gen", "1GB", "White"},

	"MA546": {"iPod Shuffle", "2nd Gen", "1GB", "Silver"},
	"MA564": {"iPod Shuffle", "2nd Gen", "1GB", "Silver"},
	"MA947": {"iPod Shuffle", "2nd Gen", "1GB", "Pink"},
	"MA949": {"iPod Shuffle", "2nd Gen", "1GB", "Blue"},
	"MA951": {"iPod Shuffle", "2nd Gen", "1GB", "Green"},
	"MA953": {"iPod Shuffle", "2nd Gen", "1GB", "Orange"},
	"MB225": {"iPod Shuffle", "2nd Gen", "1GB", "Silver"},
	"MB227": {"iPod Shuffle", "2nd Gen", "1GB", "Blue"},
	"MB228": {"iPod Shuffle", "2nd Gen", "1GB", "Blue"},
	"MB229": {"iPod Shuffle", "2nd Gen", "1GB", "Green"},
	"MB231": {"iPod Shuffle", "2nd Gen", "1GB", "Red"},
	"MB233": {"iPod Shuffle", "2nd Gen", "1GB", "Purple"},
	"MB518": {"iPod Shuffle", "2nd Gen", "2GB", "Silver"},
	"MB520": {"iPod Shuffle", "2nd Gen", "2GB", "Blue"},
	"MB522": {"iPod Shuffle", "2nd Gen", "2GB", "Green"},
	"MB524": {"iPod Shuffle", "2nd Gen", "2GB", "Red"},
	"MB526": {"iPod Shuffle", "2nd Gen", "2GB", "Purple"},
	"MB811": {"iPod Shuffle", "2nd Gen", "1GB", "Pink"},
	"MB813": {"iPod Shuffle", "2nd Gen", "1GB", "Blue"},
	"MB815": {"iPod Shuffle", "2nd Gen", "1GB", "Green"},
	"MB817": {"iPod Shuffle", "2nd Gen", "1GB", "Red"},
	"MB681": {"iPod Shuffle", "2nd Gen", "2GB", "Pink"},
	"MB683": {"iPod Shuffle", "2nd Gen", "2GB", "Blue"},
	"MB685": {"iPod Shuffle", "2nd Gen", "2GB", "Green"},
	"MB779": {"iPod Shuffle", "2nd Gen", "2GB", "Red"},
	"MC167": {"iPod Shuffle", "2nd Gen", "1GB", "Gold"},

	"MB867": {"iPod Shuffle", "3rd Gen", "4GB", "Silver"},
	"MC164": {"iPod Shuffle", "3rd Gen", "4GB", "Black"},
	"MC306": {"iPod Shuffle", "3rd Gen", "2GB", "Silver"},
	"MC323": {"iPod Shuffle", "3rd Gen", "2GB", "Black"},
	"MC381": {"iPod Shuffle", "3rd Gen", "2GB", "Green"},
	"MC384": {"iPod Shuffle", "3rd Gen", "2GB", "Blue"},
	"MC387": {"iPod Shuffle", "3rd Gen", "2GB", "Pink"},
	"MC303": {"iPod Shuffle", "3rd Gen", "4GB", "Stainless Steel"},
	"MC307": {"iPod Shuffle", "3rd Gen", "4GB", "Green"},
	"MC328": {"iPod Shuffle", "3rd Gen", "4GB", "Blue"},
	"MC331": {"iPod Shuffle", "3rd Gen", "4GB", "Pink"},

	"MC584": {"iPod Shuffle", "4th Gen", "2GB", "Silver"},
	"MC585": {"iPod Shuffle", "4th Gen", "2GB", "Pink"},
	"MC749": {"iPod Shuffle", "4th Gen", "2GB", "Orange"},
	"MC750": {"iPod Shuffle", "4th Gen", "2GB", "Green"},
	"MC751": {"iPod Shuffle", "4th Gen", "2GB", "Blue"},
	"MD773": {"iPod Shuffle", "4th Gen", "2GB", "Pink"},
	"MD774": {"iPod Shuffle", "4th Gen", "2GB", "Yellow"},
	"MD775": {"iPod Shuffle", "4th Gen", "2GB", "Blue"},
	"MD776": {"iPod Shuffle", "4th Gen", "2GB", "Green"},
	"MD777": {"iPod Shuffle", "4th Gen", "2GB", "Purple"},
	"MD778": {"iPod Shuffle", "4th Gen", "2GB", "Silver"},
	"MD779": {"iPod Shuffle", "4th Gen", "2GB", "Slate"},
	"MD780": {"iPod Shuffle", "4th Gen", "2GB", "Red"},
	"ME949": {"iPod Shuffle", "4th Gen", "2GB", "Space Gray"},
	"MKM72": {"iPod Shuffle", "4th Gen", "2GB", "Pink"},
	"MKM92": {"iPod Shuffle", "4th Gen", "2GB", "Gold"},
	"MKME2": {"iPod Shuffle", "4th Gen", "2GB", "Blue"},
	"MKMG2": {"iPod Shuffle", "4th Gen", "2GB", "Silver"},
	"MKMJ2": {"iPod Shuffle", "4th Gen", "2GB", "Space Gray"},
	"MKML2": {"iPod Shuffle", "4th Gen", "2GB", "Red"},
}

var familyGenCapabilities = map[familyGen]*DeviceCapabilities{
	{"iPod", "1st Gen"}: {
		SupportsPodcast: false,
		SupportsArtwork: false,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod", "2nd Gen"}: {
		SupportsPodcast: false,
		SupportsArtwork: false,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod", "3rd Gen"}: {
		SupportsPodcast: false,
		SupportsArtwork: false,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod", "4th Gen"}: {
		SupportsArtwork: false,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod U2", "4th Gen"}: {
		SupportsArtwork: false,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod Photo", "4th Gen"}: {
		SupportsArtwork: true,
		SupportsPhoto:   true,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod Video", "5th Gen"}: {
		SupportsVideo:   true,
		SupportsArtwork: true,
		SupportsPhoto:   true,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x19,
		ByteOrder:       "le",
	},
	{"iPod Video", "5.5th Gen"}: {
		SupportsVideo:   true,
		SupportsGapless: true,
		SupportsArtwork: true,
		SupportsPhoto:   true,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x19,
		ByteOrder:       "le",
	},
	{"iPod Video U2", "5th Gen"}: {
		SupportsVideo:   true,
		SupportsArtwork: true,
		SupportsPhoto:   true,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x19,
		ByteOrder:       "le",
	},
	{"iPod Video U2", "5.5th Gen"}: {
		SupportsVideo:   true,
		SupportsGapless: true,
		SupportsArtwork: true,
		SupportsPhoto:   true,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       20,
		DBVersion:       0x19,
		ByteOrder:       "le",
	},
	{"iPod Classic", "1st Gen"}: {
		Checksum:              ChecksumHash58,
		SupportsVideo:         true,
		SupportsGapless:       true,
		SupportsArtwork:       true,
		SupportsPhoto:         true,
		SupportsChapterImage:  true,
		SupportsSparseArtwork: true,
		SupportsPodcast:       true,
		HasScreen:             true,
		MusicDirs:             50,
		DBVersion:             0x30,
		ByteOrder:             "le",
	},
	{"iPod Classic", "2nd Gen"}: {
		Checksum:              ChecksumHash58,
		SupportsVideo:         true,
		SupportsGapless:       true,
		SupportsArtwork:       true,
		SupportsPhoto:         true,
		SupportsChapterImage:  true,
		SupportsSparseArtwork: true,
		SupportsPodcast:       true,
		HasScreen:             true,
		MusicDirs:             50,
		DBVersion:             0x30,
		ByteOrder:             "le",
	},
	{"iPod Classic", "3rd Gen"}: {
		Checksum:              ChecksumHash58,
		SupportsVideo:         true,
		SupportsGapless:       true,
		SupportsArtwork:       true,
		SupportsPhoto:         true,
		SupportsChapterImage:  true,
		SupportsSparseArtwork: true,
		SupportsPodcast:       true,
		HasScreen:             true,
		MusicDirs:             50,
		DBVersion:             0x30,
		ByteOrder:             "le",
	},
	{"iPod Mini", "1st Gen"}: {
		SupportsArtwork: false,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       6,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod Mini", "2nd Gen"}: {
		SupportsArtwork: false,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       6,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod Nano", "1st Gen"}: {
		SupportsArtwork: true,
		SupportsPhoto:   true,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       14,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod Nano", "2nd Gen"}: {
		SupportsArtwork: true,
		SupportsPhoto:   true,
		SupportsPodcast: true,
		HasScreen:       true,
		MusicDirs:       14,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod Nano", "3rd Gen"}: {
		Checksum:              ChecksumHash58,
		SupportsVideo:         true,
		SupportsGapless:       true,
		SupportsArtwork:       true,
		SupportsPhoto:         true,
		SupportsSparseArtwork: true,
		SupportsPodcast:       true,
		HasScreen:             true,
		MusicDirs:             20,
		DBVersion:             0x30,
		ByteOrder:             "le",
	},
	{"iPod Nano", "4th Gen"}: {
		Checksum:              ChecksumHash58,
		SupportsVideo:         true,
		SupportsGapless:       true,
		SupportsArtwork:       true,
		SupportsPhoto:         true,
		SupportsChapterImage:  true,
		SupportsSparseArtwork: true,
		SupportsPodcast:       true,
		HasScreen:             true,
		MusicDirs:             20,
		DBVersion:             0x30,
		ByteOrder:             "le",
	},
	{"iPod Nano", "5th Gen"}: {
		Checksum:              ChecksumHash72,
		SupportsVideo:         true,
		SupportsGapless:       true,
		SupportsArtwork:       true,
		SupportsPhoto:         true,
		SupportsSparseArtwork: true,
		SupportsCompressedDB:  true,
		SupportsPodcast:       true,
		HasScreen:             true,
		MusicDirs:             20,
		DBVersion:             0x30,
		ByteOrder:             "le",
	},
	{"iPod Nano", "6th Gen"}: {
		Checksum:              ChecksumHashAB,
		SupportsGapless:       true,
		SupportsArtwork:       true,
		SupportsSparseArtwork: true,
		SupportsCompressedDB:  true,
		UsesSQLiteDB:          true,
		SupportsPodcast:       true,
		HasScreen:             true,
		MusicDirs:             20,
		DBVersion:             0x30,
		ByteOrder:             "le",
	},
	{"iPod Nano", "7th Gen"}: {
		Checksum:              ChecksumHashAB,
		SupportsVideo:         true,
		SupportsGapless:       true,
		SupportsArtwork:       true,
		SupportsSparseArtwork: true,
		SupportsCompressedDB:  true,
		UsesSQLiteDB:          true,
		SupportsPodcast:       true,
		HasScreen:             true,
		MusicDirs:             20,
		DBVersion:             0x30,
		ByteOrder:             "le",
	},
	{"iPod Shuffle", "1st Gen"}: {
		IsShuffle:       true,
		ShadowDBVersion: 1,
		SupportsPodcast: true,
		SupportsArtwork: false,
		HasScreen:       false,
		MusicDirs:       3,
		DBVersion:       0x0c,
		ByteOrder:       "le",
	},
	{"iPod Shuffle", "2nd Gen"}: {
		IsShuffle:       true,
		ShadowDBVersion: 1,
		SupportsPodcast: true,
		SupportsArtwork: false,
		HasScreen:       false,
		MusicDirs:       3,
		DBVersion:       0x13,
		ByteOrder:       "le",
	},
	{"iPod Shuffle", "3rd Gen"}: {
		IsShuffle:       true,
		ShadowDBVersion: 2,
		SupportsPodcast: true,
		SupportsArtwork: false,
		HasScreen:       false,
		MusicDirs:       3,
		DBVersion:       0x19,
		ByteOrder:       "le",
	},
	{"iPod Shuffle", "4th Gen"}: {
		IsShuffle:       true,
		ShadowDBVersion: 2,
		SupportsPodcast: true,
		SupportsArtwork: false,
		HasScreen:       false,
		MusicDirs:       3,
		DBVersion:       0x19,
		ByteOrder:       "le",
	},
}

func DefaultCapabilities() *DeviceCapabilities {
	return &DeviceCapabilities{
		Checksum:    ChecksumNone,
		MusicDirs:   20,
		DBVersion:   0x13,
		ByteOrder:   "le",
		HasScreen:   true,
		SupportsPodcast: true,
		SupportsArtwork: false,
	}
}

func ExtractModelNumber(modelStr string) string {
	if modelStr == "" {
		return ""
	}
	if strings.HasPrefix(modelStr, "x") {
		modelStr = "M" + modelStr[1:]
	}
	upper := strings.ToUpper(modelStr)
	m := modelNumberRegexp.FindString(upper)
	if m != "" {
		return m
	}
	if len(upper) >= 5 {
		return upper[:5]
	}
	return upper
}

func LookupModel(modelNumber string) (family, generation, capacity, color string, ok bool) {
	info, found := ipodModels[modelNumber]
	if !found {
		return "", "", "", "", false
	}
	return info.Family, info.Generation, info.Capacity, info.Color, true
}

func CapabilitiesForModel(modelNumber string) *DeviceCapabilities {
	info, found := ipodModels[modelNumber]
	if !found {
		return nil
	}
	caps, found := familyGenCapabilities[familyGen{info.Family, info.Generation}]
	if !found {
		return nil
	}
	return caps
}

func DetectCapabilities(sysInfoPath string) *DeviceCapabilities {
	f, err := os.Open(sysInfoPath)
	if err != nil {
		return DefaultCapabilities()
	}
	defer f.Close()

	var modelStr string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		if strings.TrimSpace(key) == "ModelNumStr" {
			modelStr = strings.TrimSpace(value)
			break
		}
	}

	if modelStr == "" {
		return DefaultCapabilities()
	}

	modelNumber := ExtractModelNumber(modelStr)
	caps := CapabilitiesForModel(modelNumber)
	if caps == nil {
		return DefaultCapabilities()
	}
	return caps
}
