package vpd

import (
	"fmt"
	"strings"

	"howett.net/plist"
)

type VPDInfo struct {
	SerialNumber    string
	FireWireGUID    string
	FamilyID        int
	UpdaterFamilyID int
	BuildID         string
	VisibleBuildID  string
	USBSerial       string
	ModelNumStr     string
	Raw             map[string]interface{}
}

type familyEntry struct {
	Family     string
	Generation string
}

var familyIDMap = map[int]familyEntry{
	1:  {"iPod", "1st Gen"},
	2:  {"iPod", "3rd Gen"},
	3:  {"iPod Mini", "1st Gen"},
	4:  {"iPod", "4th Gen"},
	5:  {"iPod Photo", "4th Gen"},
	6:  {"iPod Video", "5th Gen"},
	7:  {"iPod Nano", "1st Gen"},
	9:  {"iPod Nano", "2nd Gen"},
	11: {"iPod Classic", "1st Gen"},
	12: {"iPod Nano", "3rd Gen"},
	15: {"iPod Nano", "4th Gen"},
	16: {"iPod Nano", "5th Gen"},
	17: {"iPod Nano", "6th Gen"},
	18: {"iPod Nano", "7th Gen"},
}

func (v *VPDInfo) FamilyGeneration() (family, generation string) {
	if e, ok := familyIDMap[v.FamilyID]; ok {
		return e.Family, e.Generation
	}
	return "", ""
}

func (v *VPDInfo) ToSysInfo() string {
	family, _ := v.FamilyGeneration()

	var b strings.Builder
	if v.ModelNumStr != "" {
		fmt.Fprintf(&b, "ModelNumStr: %s\n", v.ModelNumStr)
	}
	if v.SerialNumber != "" {
		fmt.Fprintf(&b, "pszSerialNumber: %s\n", v.SerialNumber)
	}
	if v.FireWireGUID != "" {
		fmt.Fprintf(&b, "FirewireGuid: %s\n", v.FireWireGUID)
	}
	if v.VisibleBuildID != "" {
		fmt.Fprintf(&b, "visibleBuildID: %s\n", v.VisibleBuildID)
	}
	if family != "" {
		fmt.Fprintf(&b, "BoardHwName: %s\n", family)
	}
	if v.FamilyID != 0 {
		fmt.Fprintf(&b, "iPodFamily: %d\n", v.FamilyID)
	}
	if v.UpdaterFamilyID != 0 {
		fmt.Fprintf(&b, "updaterFamily: %d\n", v.UpdaterFamilyID)
	}
	return b.String()
}

func parseVPDPlist(data []byte) (*VPDInfo, error) {
	var raw map[string]interface{}
	_, err := plist.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("vpd: plist decode: %w", err)
	}

	info := &VPDInfo{Raw: raw}

	if v, ok := raw["SerialNumber"].(string); ok {
		info.SerialNumber = v
	}
	if v, ok := raw["FireWireGUID"].(string); ok {
		info.FireWireGUID = v
	} else if v, ok := raw["FirewireGuid"].(string); ok {
		info.FireWireGUID = v
	}
	if v, ok := raw["FamilyID"]; ok {
		info.FamilyID = toInt(v)
	}
	if v, ok := raw["UpdaterFamilyID"]; ok {
		info.UpdaterFamilyID = toInt(v)
	}
	if v, ok := raw["BuildID"].(string); ok {
		info.BuildID = v
	}
	if v, ok := raw["VisibleBuildID"].(string); ok {
		info.VisibleBuildID = v
	}

	return info, nil
}

func toInt(v interface{}) int {
	switch n := v.(type) {
	case uint64:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	}
	return 0
}

func buildInquiryCDB(evpd bool, page, allocLen byte) [6]byte {
	var cdb [6]byte
	cdb[0] = 0x12 // INQUIRY
	if evpd {
		cdb[1] = 0x01
	}
	cdb[2] = page
	cdb[4] = allocLen
	return cdb
}

func readVPDPages(inquiry func([6]byte, int) ([]byte, error)) ([]byte, error) {
	cdb := buildInquiryCDB(true, 0xC0, 255)
	resp, err := inquiry(cdb, 255)
	if err != nil {
		return nil, fmt.Errorf("vpd: read page 0xC0: %w", err)
	}

	if len(resp) < 4 {
		return nil, fmt.Errorf("vpd: page 0xC0 response too short (%d bytes)", len(resp))
	}

	pageLen := int(resp[3])
	pages := resp[4 : 4+pageLen]

	var result []byte
	for _, pg := range pages {
		if pg < 0xC2 {
			continue
		}
		cdb = buildInquiryCDB(true, pg, 255)
		data, err := inquiry(cdb, 255)
		if err != nil {
			continue
		}
		if len(data) < 4 {
			continue
		}
		payloadLen := int(data[3])
		if len(data) < 4+payloadLen {
			payloadLen = len(data) - 4
		}
		result = append(result, data[4:4+payloadLen]...)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("vpd: no VPD payload data found")
	}

	return result, nil
}
