package ipod

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type DeviceInfo struct {
	MountPoint      string `json:"mountPoint"`
	Name            string `json:"name"`
	FreeSpace       int64  `json:"freeSpace"`
	TotalSpace      int64  `json:"totalSpace"`
	Family          string `json:"family"`
	Generation      string `json:"generation"`
	Capacity        string `json:"capacity"`
	Color           string `json:"color"`
	Model           string `json:"model"`
	Icon            string `json:"icon"`
	DisplayCapacity string `json:"displayCapacity"`
	SerialNumber    string `json:"serialNumber,omitempty"`
}

type modelInfo struct {
	Family     string
	Generation string
	Capacity   string
	Color      string
}

// ipodModels maps model numbers to device metadata.
// Ported from iOpenPod (MIT licensed, github.com/therealsavi/iOpenPod).
var ipodModels = map[string]modelInfo{
	// iPod Classic (2007-2009)
	"MB029": {"iPod Classic", "1st Gen", "80GB", "Silver"},
	"MB147": {"iPod Classic", "1st Gen", "80GB", "Black"},
	"MB145": {"iPod Classic", "1st Gen", "160GB", "Silver"},
	"MB150": {"iPod Classic", "1st Gen", "160GB", "Black"},
	"MB562": {"iPod Classic", "2nd Gen", "120GB", "Silver"},
	"MB565": {"iPod Classic", "2nd Gen", "120GB", "Black"},
	"MC293": {"iPod Classic", "3rd Gen", "160GB", "Silver"},
	"MC297": {"iPod Classic", "3rd Gen", "160GB", "Black"},

	// iPod (Scroll Wheel) — 1st Gen (2001)
	"M8513": {"iPod", "1st Gen", "5GB", "White"},
	"M8541": {"iPod", "1st Gen", "5GB", "White"},
	"M8697": {"iPod", "1st Gen", "5GB", "White"},
	"M8709": {"iPod", "1st Gen", "10GB", "White"},

	// iPod (Touch Wheel) — 2nd Gen (2002)
	"M8737": {"iPod", "2nd Gen", "10GB", "White"},
	"M8740": {"iPod", "2nd Gen", "10GB", "White"},
	"M8738": {"iPod", "2nd Gen", "20GB", "White"},
	"M8741": {"iPod", "2nd Gen", "20GB", "White"},

	// iPod (Dock Connector) — 3rd Gen (2003)
	"M8976": {"iPod", "3rd Gen", "10GB", "White"},
	"M8946": {"iPod", "3rd Gen", "15GB", "White"},
	"M8948": {"iPod", "3rd Gen", "30GB", "White"},
	"M9244": {"iPod", "3rd Gen", "20GB", "White"},
	"M9245": {"iPod", "3rd Gen", "40GB", "White"},
	"M9460": {"iPod", "3rd Gen", "15GB", "White"},

	// iPod (Click Wheel) — 4th Gen (2004)
	"M9268": {"iPod", "4th Gen", "40GB", "White"},
	"M9282": {"iPod", "4th Gen", "20GB", "White"},
	"ME436": {"iPod", "4th Gen", "40GB", "White"},
	"M9787": {"iPod U2", "4th Gen", "20GB", "Black"},

	// iPod Photo / Color Display — 4th Gen (2004-2005)
	"M9585": {"iPod Photo", "4th Gen", "40GB", "White"},
	"M9586": {"iPod Photo", "4th Gen", "60GB", "White"},
	"M9829": {"iPod Photo", "4th Gen", "30GB", "White"},
	"M9830": {"iPod Photo", "4th Gen", "60GB", "White"},
	"MA079": {"iPod Photo", "4th Gen", "20GB", "White"},
	"MA127": {"iPod U2", "4th Gen", "20GB", "Black"},
	"MS492": {"iPod Photo", "4th Gen", "30GB", "White"},
	"MA215": {"iPod Photo", "4th Gen", "20GB", "White"},

	// iPod Video — 5th Gen (2005)
	"MA002": {"iPod Video", "5th Gen", "30GB", "White"},
	"MA003": {"iPod Video", "5th Gen", "60GB", "White"},
	"MA146": {"iPod Video", "5th Gen", "30GB", "Black"},
	"MA147": {"iPod Video", "5th Gen", "60GB", "Black"},
	"MA452": {"iPod Video U2", "5th Gen", "30GB", "Black"},

	// iPod Video — 5.5th Gen (Late 2006)
	"MA444": {"iPod Video", "5.5th Gen", "30GB", "White"},
	"MA446": {"iPod Video", "5.5th Gen", "30GB", "Black"},
	"MA448": {"iPod Video", "5.5th Gen", "80GB", "White"},
	"MA450": {"iPod Video", "5.5th Gen", "80GB", "Black"},
	"MA664": {"iPod Video U2", "5.5th Gen", "30GB", "Black"},

	// iPod Mini — 1st Gen (2004)
	"M9160": {"iPod Mini", "1st Gen", "4GB", "Silver"},
	"M9434": {"iPod Mini", "1st Gen", "4GB", "Green"},
	"M9435": {"iPod Mini", "1st Gen", "4GB", "Pink"},
	"M9436": {"iPod Mini", "1st Gen", "4GB", "Blue"},
	"M9437": {"iPod Mini", "1st Gen", "4GB", "Gold"},

	// iPod Mini — 2nd Gen (2005)
	"M9800": {"iPod Mini", "2nd Gen", "4GB", "Silver"},
	"M9801": {"iPod Mini", "2nd Gen", "6GB", "Silver"},
	"M9802": {"iPod Mini", "2nd Gen", "4GB", "Blue"},
	"M9803": {"iPod Mini", "2nd Gen", "6GB", "Blue"},
	"M9804": {"iPod Mini", "2nd Gen", "4GB", "Pink"},
	"M9805": {"iPod Mini", "2nd Gen", "6GB", "Pink"},
	"M9806": {"iPod Mini", "2nd Gen", "4GB", "Green"},
	"M9807": {"iPod Mini", "2nd Gen", "6GB", "Green"},

	// iPod Nano — 1st Gen (2005)
	"MA004": {"iPod Nano", "1st Gen", "2GB", "White"},
	"MA005": {"iPod Nano", "1st Gen", "4GB", "White"},
	"MA099": {"iPod Nano", "1st Gen", "2GB", "Black"},
	"MA107": {"iPod Nano", "1st Gen", "4GB", "Black"},
	"MA350": {"iPod Nano", "1st Gen", "1GB", "White"},
	"MA352": {"iPod Nano", "1st Gen", "1GB", "Black"},

	// iPod Nano — 2nd Gen (2006)
	"MA426": {"iPod Nano", "2nd Gen", "4GB", "Silver"},
	"MA428": {"iPod Nano", "2nd Gen", "4GB", "Blue"},
	"MA477": {"iPod Nano", "2nd Gen", "2GB", "Silver"},
	"MA487": {"iPod Nano", "2nd Gen", "4GB", "Green"},
	"MA489": {"iPod Nano", "2nd Gen", "4GB", "Pink"},
	"MA497": {"iPod Nano", "2nd Gen", "8GB", "Black"},
	"MA725": {"iPod Nano", "2nd Gen", "4GB", "Red"},
	"MA726": {"iPod Nano", "2nd Gen", "8GB", "Red"},
	"MA899": {"iPod Nano", "2nd Gen", "8GB", "Red"},

	// iPod Nano — 3rd Gen (2007)
	"MA978": {"iPod Nano", "3rd Gen", "4GB", "Silver"},
	"MA980": {"iPod Nano", "3rd Gen", "8GB", "Silver"},
	"MB249": {"iPod Nano", "3rd Gen", "8GB", "Blue"},
	"MB253": {"iPod Nano", "3rd Gen", "8GB", "Green"},
	"MB257": {"iPod Nano", "3rd Gen", "8GB", "Red"},
	"MB261": {"iPod Nano", "3rd Gen", "8GB", "Black"},
	"MB453": {"iPod Nano", "3rd Gen", "8GB", "Pink"},

	// iPod Nano — 4th Gen (2008)
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

	// iPod Nano — 5th Gen (2009)
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

	// iPod Nano — 6th Gen (2010)
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

	// iPod Nano — 7th Gen (2012)
	"MD475": {"iPod Nano", "7th Gen", "16GB", "Pink"},
	"MD476": {"iPod Nano", "7th Gen", "16GB", "Yellow"},
	"MD477": {"iPod Nano", "7th Gen", "16GB", "Blue"},
	"MD478": {"iPod Nano", "7th Gen", "16GB", "Green"},
	"MD479": {"iPod Nano", "7th Gen", "16GB", "Purple"},
	"MD480": {"iPod Nano", "7th Gen", "16GB", "Silver"},
	"MD481": {"iPod Nano", "7th Gen", "16GB", "Slate"},
	"MD744": {"iPod Nano", "7th Gen", "16GB", "Red"},
	"ME971": {"iPod Nano", "7th Gen", "16GB", "Space Gray"},
	// Mid 2015 refresh
	"MKMV2": {"iPod Nano", "7th Gen", "16GB", "Pink"},
	"MKMX2": {"iPod Nano", "7th Gen", "16GB", "Gold"},
	"MKN02": {"iPod Nano", "7th Gen", "16GB", "Blue"},
	"MKN22": {"iPod Nano", "7th Gen", "16GB", "Silver"},
	"MKN52": {"iPod Nano", "7th Gen", "16GB", "Space Gray"},
	"MKN72": {"iPod Nano", "7th Gen", "16GB", "Red"},

	// iPod Shuffle — 1st Gen (2005)
	"M9724": {"iPod Shuffle", "1st Gen", "512MB", "White"},
	"M9725": {"iPod Shuffle", "1st Gen", "1GB", "White"},

	// iPod Shuffle — 2nd Gen (2006-2008)
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

	// iPod Shuffle — 3rd Gen (2009)
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

	// iPod Shuffle — 4th Gen (2010-2015)
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

	// iPod Classic — additional (MC007 used in older code)
	"MC007": {"iPod Classic", "1st Gen", "80GB", "Silver"},
	"PB029": {"iPod Classic", "1st Gen", "80GB", "Silver"},
}

type colorKey struct {
	family     string
	generation string
	color      string
}

var colorMap = map[colorKey]string{
	// iPod (1G–4G)
	{"ipod", "1st gen", "white"}:       "iPod1.png",
	{"ipod", "2nd gen", "white"}:       "iPod1.png",
	{"ipod", "3rd gen", "white"}:       "iPod2.png",
	{"ipod", "4th gen", "white"}:       "iPod4-White.png",
	{"ipod u2", "4th gen", "black"}:    "iPod4-BlackRed.png",

	// iPod Photo
	{"ipod photo", "4th gen", "white"}:    "iPod5-White.png",
	{"ipod photo u2", "4th gen", "black"}: "iPod5-BlackRed.png",

	// iPod Video
	{"ipod video", "5th gen", "white"}:      "iPod6-White.png",
	{"ipod video", "5th gen", "black"}:      "iPod6-Black.png",
	{"ipod video", "5.5th gen", "white"}:    "iPod6-White.png",
	{"ipod video", "5.5th gen", "black"}:    "iPod6-Black.png",
	{"ipod video u2", "5th gen", "black"}:   "iPod6-BlackRed.png",
	{"ipod video u2", "5.5th gen", "black"}: "iPod6-BlackRed.png",

	// iPod Classic
	{"ipod classic", "1st gen", "silver"}: "iPod11-Silver.png",
	{"ipod classic", "1st gen", "black"}:  "iPod11-Black.png",
	{"ipod classic", "2nd gen", "silver"}: "iPod11-Silver.png",
	{"ipod classic", "2nd gen", "black"}:  "iPod11B-Black.png",
	{"ipod classic", "3rd gen", "silver"}: "iPod11-Silver.png",
	{"ipod classic", "3rd gen", "black"}:  "iPod11B-Black.png",

	// iPod Mini 1st Gen
	{"ipod mini", "1st gen", "silver"}: "iPod3-Silver.png",
	{"ipod mini", "1st gen", "blue"}:   "iPod3-Blue.png",
	{"ipod mini", "1st gen", "gold"}:   "iPod3-Gold.png",
	{"ipod mini", "1st gen", "green"}:  "iPod3-Green.png",
	{"ipod mini", "1st gen", "pink"}:   "iPod3-Pink.png",

	// iPod Mini 2nd Gen
	{"ipod mini", "2nd gen", "silver"}: "iPod3-Silver.png",
	{"ipod mini", "2nd gen", "blue"}:   "iPod3B-Blue.png",
	{"ipod mini", "2nd gen", "green"}:  "iPod3B-Green.png",
	{"ipod mini", "2nd gen", "pink"}:   "iPod3B-Pink.png",

	// iPod Nano 1st Gen
	{"ipod nano", "1st gen", "white"}: "iPod7-White.png",
	{"ipod nano", "1st gen", "black"}: "iPod7-Black.png",

	// iPod Nano 2nd Gen
	{"ipod nano", "2nd gen", "silver"}: "iPod9-Silver.png",
	{"ipod nano", "2nd gen", "black"}:  "iPod9-Black.png",
	{"ipod nano", "2nd gen", "blue"}:   "iPod9-Blue.png",
	{"ipod nano", "2nd gen", "green"}:  "iPod9-Green.png",
	{"ipod nano", "2nd gen", "pink"}:   "iPod9-Pink.png",
	{"ipod nano", "2nd gen", "red"}:    "iPod9-Red.png",

	// iPod Nano 3rd Gen
	{"ipod nano", "3rd gen", "silver"}: "iPod12-Silver.png",
	{"ipod nano", "3rd gen", "black"}:  "iPod12-Black.png",
	{"ipod nano", "3rd gen", "blue"}:   "iPod12-Blue.png",
	{"ipod nano", "3rd gen", "green"}:  "iPod12-Green.png",
	{"ipod nano", "3rd gen", "pink"}:   "iPod12-Pink.png",
	{"ipod nano", "3rd gen", "red"}:    "iPod12-Red.png",

	// iPod Nano 4th Gen
	{"ipod nano", "4th gen", "silver"}: "iPod15-Silver.png",
	{"ipod nano", "4th gen", "black"}:  "iPod15-Black.png",
	{"ipod nano", "4th gen", "blue"}:   "iPod15-Blue.png",
	{"ipod nano", "4th gen", "green"}:  "iPod15-Green.png",
	{"ipod nano", "4th gen", "orange"}: "iPod15-Orange.png",
	{"ipod nano", "4th gen", "pink"}:   "iPod15-Pink.png",
	{"ipod nano", "4th gen", "purple"}: "iPod15-Purple.png",
	{"ipod nano", "4th gen", "red"}:    "iPod15-Red.png",
	{"ipod nano", "4th gen", "yellow"}: "iPod15-Yellow.png",

	// iPod Nano 5th Gen
	{"ipod nano", "5th gen", "silver"}: "iPod16-Silver.png",
	{"ipod nano", "5th gen", "black"}:  "iPod16-Black.png",
	{"ipod nano", "5th gen", "blue"}:   "iPod16-Blue.png",
	{"ipod nano", "5th gen", "green"}:  "iPod16-Green.png",
	{"ipod nano", "5th gen", "orange"}: "iPod16-Orange.png",
	{"ipod nano", "5th gen", "pink"}:   "iPod16-Pink.png",
	{"ipod nano", "5th gen", "purple"}: "iPod16-Purple.png",
	{"ipod nano", "5th gen", "red"}:    "iPod16-Red.png",
	{"ipod nano", "5th gen", "yellow"}: "iPod16-Yellow.png",

	// iPod Nano 6th Gen
	{"ipod nano", "6th gen", "silver"}:   "iPod17-Silver.png",
	{"ipod nano", "6th gen", "graphite"}: "iPod17-DarkGray.png",
	{"ipod nano", "6th gen", "blue"}:     "iPod17-Blue.png",
	{"ipod nano", "6th gen", "green"}:    "iPod17-Green.png",
	{"ipod nano", "6th gen", "orange"}:   "iPod17-Orange.png",
	{"ipod nano", "6th gen", "pink"}:     "iPod17-Pink.png",
	{"ipod nano", "6th gen", "red"}:      "iPod17-Red.png",

	// iPod Nano 7th Gen
	{"ipod nano", "7th gen", "silver"}:     "iPod18A-Silver.png",
	{"ipod nano", "7th gen", "space gray"}: "iPod18A-SpaceGray.png",
	{"ipod nano", "7th gen", "blue"}:       "iPod18A-Blue.png",
	{"ipod nano", "7th gen", "pink"}:       "iPod18A-Pink.png",
	{"ipod nano", "7th gen", "red"}:        "iPod18A-Red.png",
	{"ipod nano", "7th gen", "gold"}:       "iPod18A-Gold.png",
	{"ipod nano", "7th gen", "slate"}:      "iPod18-DarkGray.png",
	{"ipod nano", "7th gen", "green"}:      "iPod18-Green.png",
	{"ipod nano", "7th gen", "purple"}:     "iPod18-Purple.png",
	{"ipod nano", "7th gen", "yellow"}:     "iPod18-Yellow.png",

	// iPod Shuffle 1st Gen
	{"ipod shuffle", "1st gen", "white"}: "iPod128.png",

	// iPod Shuffle 2nd Gen
	{"ipod shuffle", "2nd gen", "silver"}: "iPod130-Silver.png",
	{"ipod shuffle", "2nd gen", "blue"}:   "iPod130-Blue.png",
	{"ipod shuffle", "2nd gen", "green"}:  "iPod130-Green.png",
	{"ipod shuffle", "2nd gen", "pink"}:   "iPod130-Pink.png",
	{"ipod shuffle", "2nd gen", "orange"}: "iPod130-Orange.png",
	{"ipod shuffle", "2nd gen", "purple"}: "iPod130C-Purple.png",
	{"ipod shuffle", "2nd gen", "red"}:    "iPod130C-Red.png",
	{"ipod shuffle", "2nd gen", "gold"}:   "iPod130F-Gold.png",

	// iPod Shuffle 3rd Gen
	{"ipod shuffle", "3rd gen", "silver"}:          "iPod132-Silver.png",
	{"ipod shuffle", "3rd gen", "black"}:           "iPod132-DarkGray.png",
	{"ipod shuffle", "3rd gen", "blue"}:            "iPod132-Blue.png",
	{"ipod shuffle", "3rd gen", "green"}:           "iPod132-Green.png",
	{"ipod shuffle", "3rd gen", "pink"}:            "iPod132-Pink.png",
	{"ipod shuffle", "3rd gen", "stainless steel"}: "iPod132B-Silver.png",

	// iPod Shuffle 4th Gen
	{"ipod shuffle", "4th gen", "silver"}:     "iPod133D-Silver.png",
	{"ipod shuffle", "4th gen", "space gray"}: "iPod133D-SpaceGray.png",
	{"ipod shuffle", "4th gen", "blue"}:       "iPod133D-Blue.png",
	{"ipod shuffle", "4th gen", "pink"}:       "iPod133D-Pink.png",
	{"ipod shuffle", "4th gen", "red"}:        "iPod133D-Red.png",
	{"ipod shuffle", "4th gen", "gold"}:       "iPod133D-Gold.png",
	{"ipod shuffle", "4th gen", "slate"}:      "iPod133B-DarkGray.png",
	{"ipod shuffle", "4th gen", "green"}:      "iPod133B-Green.png",
	{"ipod shuffle", "4th gen", "purple"}:     "iPod133B-Purple.png",
	{"ipod shuffle", "4th gen", "yellow"}:     "iPod133B-Yellow.png",
	{"ipod shuffle", "4th gen", "orange"}:     "iPod133-Orange.png",
}

var modelImage = map[string]string{
	// iPod Nano 7th Gen (2012 → iPod18)
	"MD475": "iPod18-Pink.png",
	"MD477": "iPod18-Blue.png",
	"MD480": "iPod18-Silver.png",
	"MD744": "iPod18-Red.png",
	"ME971": "iPod18-SpaceGray.png",

	// iPod Shuffle 2nd Gen — Sept 2007 (iPod130C)
	"MB227": "iPod130C-Blue.png",
	"MB228": "iPod130C-Blue.png",
	"MB229": "iPod130C-Green.png",
	"MB520": "iPod130C-Blue.png",
	"MB522": "iPod130C-Green.png",

	// iPod Shuffle 2nd Gen — 2008 (iPod130F)
	"MB811": "iPod130F-Pink.png",
	"MB813": "iPod130F-Blue.png",
	"MB815": "iPod130F-Green.png",
	"MB817": "iPod130F-Red.png",
	"MB681": "iPod130F-Pink.png",
	"MB683": "iPod130F-Blue.png",
	"MB685": "iPod130F-Green.png",
	"MB779": "iPod130F-Red.png",

	// iPod Shuffle 4th Gen — 2010 (iPod133)
	"MC584": "iPod133-Silver.png",
	"MC585": "iPod133-Pink.png",
	"MC750": "iPod133-Green.png",
	"MC751": "iPod133-Blue.png",

	// iPod Shuffle 4th Gen — Late 2012 (iPod133B)
	"MD773": "iPod133B-Pink.png",
	"MD775": "iPod133B-Blue.png",
	"MD778": "iPod133B-Silver.png",
	"MD780": "iPod133B-Red.png",
	"ME949": "iPod133B-SpaceGray.png",
}

var familyFallback = map[string]string{
	"ipod":           "iPod4-White.png",
	"ipod u2":        "iPod4-BlackRed.png",
	"ipod photo":     "iPod5-White.png",
	"ipod photo u2":  "iPod5-BlackRed.png",
	"ipod video":     "iPod6-White.png",
	"ipod video u2":  "iPod6-BlackRed.png",
	"ipod classic":   "iPod11-Silver.png",
	"ipod mini":      "iPod3-Silver.png",
	"ipod nano":      "iPod15-Silver.png",
	"ipod shuffle":   "iPod133D-Silver.png",
}

var extractModelRe = regexp.MustCompile(`^(M[A-Z]?\d{3,4})`)

func extractModelNumber(raw string) string {
	if raw == "" {
		return ""
	}
	if raw[0] == 'x' {
		raw = "M" + raw[1:]
	}
	raw = strings.ToUpper(raw)
	if m := extractModelRe.FindString(raw); m != "" {
		return m
	}
	if len(raw) >= 5 {
		return raw[:5]
	}
	return raw
}

func getModelInfo(modelNum string) *modelInfo {
	if info, ok := ipodModels[modelNum]; ok {
		return &info
	}
	if len(modelNum) >= 4 {
		prefix := modelNum[:4]
		for k, info := range ipodModels {
			if strings.HasPrefix(k, prefix) {
				return &info
			}
		}
	}
	return nil
}

func resolveImageFilename(family, generation, color string) string {
	fam := strings.ToLower(family)
	gen := strings.ToLower(generation)
	col := strings.ToLower(strings.TrimSpace(color))

	if col != "" {
		if fn, ok := colorMap[colorKey{fam, gen, col}]; ok {
			return fn
		}
	}
	for _, defaultCol := range []string{"silver", "white"} {
		if fn, ok := colorMap[colorKey{fam, gen, defaultCol}]; ok {
			return fn
		}
	}
	if fn, ok := familyFallback[fam]; ok {
		return fn
	}
	return "iPodGeneric.png"
}

func imageForModel(modelNum string) string {
	if fn, ok := modelImage[modelNum]; ok {
		return fn
	}
	if info, ok := ipodModels[modelNum]; ok {
		return resolveImageFilename(info.Family, info.Generation, info.Color)
	}
	return "iPodGeneric.png"
}

func parseCapacityGB(cap string) int {
	cap = strings.ToUpper(strings.TrimSpace(cap))
	if strings.HasSuffix(cap, "MB") {
		return 0
	}
	cap = strings.TrimSuffix(cap, "GB")
	var gb int
	fmt.Sscanf(cap, "%d", &gb)
	return gb
}

func displayCapacity(modelCapacityGB int, actualBytes int64) string {
	actualGB := float64(actualBytes) / (1 << 30)
	if modelCapacityGB > 0 && math.Abs(actualGB-float64(modelCapacityGB))/float64(modelCapacityGB) < 0.2 {
		return fmt.Sprintf("%d GB", modelCapacityGB)
	}
	sizes := []int{8, 16, 32, 64, 128, 256, 512, 1024, 2048}
	for _, s := range sizes {
		if int(actualGB) <= s {
			if s >= 1024 {
				return fmt.Sprintf("%d TB", s/1024)
			}
			return fmt.Sprintf("%d GB", s)
		}
	}
	return fmt.Sprintf("%d GB", int(actualGB))
}

func ReadSysInfo(mountPoint string) *DeviceInfo {
	f, err := os.Open(filepath.Join(mountPoint, "iPod_Control", "Device", "SysInfo"))
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var modelNumRaw, serial, fwGuid string
	for scanner.Scan() {
		line := scanner.Text()
		if k, v, ok := strings.Cut(line, ":"); ok {
			k, v = strings.TrimSpace(k), strings.TrimSpace(v)
			switch k {
			case "ModelNumStr":
				modelNumRaw = v
			case "pszSerialNumber":
				serial = v
			case "FirewireGuid":
				fwGuid = v
			}
		}
	}

	if modelNumRaw == "" {
		return nil
	}

	serialNumber := serial
	if serialNumber == "" {
		serialNumber = fwGuid
	}

	modelNum := extractModelNumber(modelNumRaw)
	info := getModelInfo(modelNum)
	if info == nil {
		return &DeviceInfo{
			Model:        modelNum,
			Icon:         "iPodGeneric.png",
			SerialNumber: serialNumber,
		}
	}

	return &DeviceInfo{
		Family:       info.Family,
		Generation:   info.Generation,
		Capacity:     info.Capacity,
		Color:        info.Color,
		Model:        modelNum,
		Icon:         imageForModel(modelNum),
		SerialNumber: serialNumber,
	}
}

func fillDeviceInfo(di *DeviceInfo, sysInfo *DeviceInfo) {
	if sysInfo == nil {
		di.Icon = "iPodGeneric.png"
		return
	}
	di.Family = sysInfo.Family
	di.Generation = sysInfo.Generation
	di.Capacity = sysInfo.Capacity
	di.Color = sysInfo.Color
	di.Model = sysInfo.Model
	di.Icon = sysInfo.Icon
	di.SerialNumber = sysInfo.SerialNumber
	if di.TotalSpace > 0 {
		di.DisplayCapacity = displayCapacity(parseCapacityGB(sysInfo.Capacity), di.TotalSpace)
	} else if sysInfo.Capacity != "" {
		di.DisplayCapacity = strings.Replace(sysInfo.Capacity, "GB", " GB", 1)
	}
}
