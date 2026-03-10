package restore

import "strings"

const appleVID = 0x05AC

type DeviceMode string

const (
	ModeDisk DeviceMode = "disk"
	ModeDFU  DeviceMode = "dfu"
	ModeWTF  DeviceMode = "wtf"
)

type IPodModel struct {
	Name             string
	DiskPID          uint16
	DFUPID           uint16
	WTFPID           uint16
	FirmwarePartSize int64
	SectorSize       int
	Family           string
	Generation       string
	Restorable       bool
}

var ipodModels = []IPodModel{
	// PortalPlayer — disk mode only, no DFU
	// 1G/2G/3G are FireWire-era and not supported for restore
	{
		Name: "iPod 1st/2nd Gen", DiskPID: 0x1201,
		FirmwarePartSize: 32 * 1024 * 1024, SectorSize: 512,
		Family: "iPod", Generation: "1st/2nd Gen",
	},
	// PID 0x1203 is shared by 3G and 4G but 3G (FireWire) is not supported,
	// so we map 0x1203 exclusively to 4G.
	{
		Name: "iPod 4th Gen", DiskPID: 0x1203,
		FirmwarePartSize: 40 * 1024 * 1024, SectorSize: 512,
		Family: "iPod", Generation: "4th Gen", Restorable: true,
	},
	{
		Name: "iPod Photo/Color", DiskPID: 0x1204,
		FirmwarePartSize: 40 * 1024 * 1024, SectorSize: 512,
		Family: "iPod Photo", Generation: "4th Gen", Restorable: true,
	},
	{
		Name: "iPod Mini", DiskPID: 0x1205,
		FirmwarePartSize: 40 * 1024 * 1024, SectorSize: 512,
		Family: "iPod Mini", Generation: "1st Gen", Restorable: true,
	},
	{
		Name: "iPod Video", DiskPID: 0x1209,
		FirmwarePartSize: 40 * 1024 * 1024, SectorSize: 2048,
		Family: "iPod Video", Generation: "5th Gen", Restorable: true,
	},
	{
		Name: "iPod Nano 1st Gen", DiskPID: 0x120A,
		FirmwarePartSize: 25 * 1024 * 1024, SectorSize: 512,
		Family: "iPod Nano", Generation: "1st Gen", Restorable: true,
	},

	// Samsung/S5L — not supported for restore (requires DFU flashing)
	{
		Name: "iPod Nano 2nd Gen", DiskPID: 0x1260, DFUPID: 0x1220, WTFPID: 0x1240,
		Family: "iPod Nano", Generation: "2nd Gen",
	},
	{
		Name: "iPod Classic", DiskPID: 0x1261,
		Family: "iPod Classic", Generation: "1st Gen",
	},
	{
		Name: "iPod Classic (DFU)", DiskPID: 0x1262, DFUPID: 0x1223, WTFPID: 0x1241,
		Family: "iPod Classic", Generation: "1st Gen",
	},
	{
		Name: "iPod Nano 3rd Gen", DiskPID: 0x1262, DFUPID: 0x1223, WTFPID: 0x1242,
		Family: "iPod Nano", Generation: "3rd Gen",
	},
	{
		Name: "iPod Nano 4th Gen", DiskPID: 0x1265, DFUPID: 0x1225, WTFPID: 0x1243,
		Family: "iPod Nano", Generation: "4th Gen",
	},
	{
		Name: "iPod Nano 5th Gen", DiskPID: 0x1267, DFUPID: 0x1231, WTFPID: 0x1246,
		Family: "iPod Nano", Generation: "5th Gen",
	},
}

type USBIPod struct {
	Model    *IPodModel `json:"model"`
	Mode     DeviceMode `json:"mode"`
	DiskPath string     `json:"diskPath,omitempty"`
}

func ModelByPID(pid uint16) (*IPodModel, DeviceMode) {
	for i := range ipodModels {
		if ipodModels[i].DiskPID == pid {
			return &ipodModels[i], ModeDisk
		}
		if ipodModels[i].DFUPID != 0 && ipodModels[i].DFUPID == pid {
			return &ipodModels[i], ModeDFU
		}
		if ipodModels[i].WTFPID != 0 && ipodModels[i].WTFPID == pid {
			return &ipodModels[i], ModeWTF
		}
	}
	return nil, ""
}

func ModelByFamilyGeneration(family, generation string) *IPodModel {
	family = strings.ToLower(strings.TrimSpace(family))
	generation = strings.ToLower(strings.TrimSpace(generation))

	for i := range ipodModels {
		m := &ipodModels[i]
		mf := strings.ToLower(m.Family)
		mg := strings.ToLower(m.Generation)

		if mf == family && mg == generation {
			return m
		}
		if family == mf+" u2" && mg == generation {
			return m
		}
		if mg == "1st/2nd gen" && (generation == "1st gen" || generation == "2nd gen") && mf == family {
			return m
		}
		if mf == "ipod video" && family == "ipod video" && generation == "5.5th gen" {
			return m
		}
	}
	return nil
}
