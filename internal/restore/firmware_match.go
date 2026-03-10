package restore

import (
	"strings"
	"sync"
)

type firmwareRule struct {
	family     string
	generation string
	models     []string // if non-empty, only match these model prefixes
	indices    []int    // indices into the IPSW catalog
}

var (
	firmwareRules []firmwareRule
	firmwareOnce  sync.Once
)

func ensureFirmwareRules() {
	firmwareOnce.Do(func() {
		firmwareRules = buildFirmwareRules()
	})
}

func buildFirmwareRules() []firmwareRule {
	idx := make(map[string]int)
	for i, e := range catalog {
		idx[e.Filename] = i
	}
	i := func(filename string) int {
		if v, ok := idx[filename]; ok {
			return v
		}
		return -1
	}

	return []firmwareRule{
		// iPod 1st/2nd Gen
		{family: "iPod", generation: "1st Gen", indices: []int{i("iPod_1.1.5.ipsw")}},
		{family: "iPod", generation: "2nd Gen", indices: []int{i("iPod_1.1.5.ipsw")}},

		// iPod 3rd Gen (Dock Connector)
		{family: "iPod", generation: "3rd Gen", indices: []int{i("iPod_2.2.3.ipsw")}},

		// iPod 4th Gen (Click Wheel) — Initial: M9268, M9282, M9787
		{family: "iPod", generation: "4th Gen", models: []string{"M9268", "M9282", "M9787"},
			indices: []int{i("iPod_4.3.1.1.ipsw")}},
		{family: "iPod U2", generation: "4th Gen",
			indices: []int{i("iPod_4.3.1.1.ipsw")}},
		// iPod 4th Gen Rev A: ME436
		{family: "iPod", generation: "4th Gen", models: []string{"ME436"},
			indices: []int{i("iPod_10.3.1.1.ipsw")}},

		// iPod Photo / Color
		{family: "iPod Photo", generation: "4th Gen", models: []string{"M9585", "M9586"},
			indices: []int{i("iPod_5.1.2.1.ipsw")}},
		{family: "iPod Photo", generation: "4th Gen", models: []string{"M9829", "M9830", "MA079", "MS492", "MA215"},
			indices: []int{i("iPod_11.1.2.1.ipsw")}},

		// iPod Video 5th Gen — Initial
		{family: "iPod Video", generation: "5th Gen", models: []string{"MA002", "MA003", "MA146", "MA147", "MA452"},
			indices: []int{i("iPod_13.1.3.ipsw")}},
		// iPod Video 5th Gen — Rev A
		{family: "iPod Video", generation: "5th Gen", models: []string{"MA444", "MA446", "MA448", "MA450", "MA664"},
			indices: []int{i("iPod_25.1.3.ipsw")}},
		{family: "iPod Video", generation: "5.5th Gen",
			indices: []int{i("iPod_25.1.3.ipsw")}},
		{family: "iPod Video U2", generation: "5th Gen",
			indices: []int{i("iPod_13.1.3.ipsw")}},
		{family: "iPod Video U2", generation: "5.5th Gen",
			indices: []int{i("iPod_25.1.3.ipsw")}},

		// iPod Classic
		{family: "iPod Classic", generation: "1st Gen",
			indices: []int{i("iPod_24.1.1.2.ipsw")}},
		{family: "iPod Classic", generation: "2nd Gen",
			indices: []int{i("iPod_33.2.0.1.ipsw")}},
		{family: "iPod Classic", generation: "3rd Gen",
			indices: []int{i("iPod_35.2.0.4.ipsw"), i("iPod_38.2.0.5.ipsw")}},

		// iPod Mini
		{family: "iPod Mini", generation: "1st Gen", models: []string{"M9160", "M9434", "M9435", "M9436", "M9437"},
			indices: []int{i("iPod_3.1.4.1.ipsw")}},
		{family: "iPod Mini", generation: "2nd Gen",
			indices: []int{i("iPod_7.1.4.1.ipsw")}},

		// iPod Nano
		{family: "iPod Nano", generation: "1st Gen",
			indices: []int{i("iPod_14.1.3.1.ipsw"), i("iPod_17.1.3.1.ipsw")}},
		{family: "iPod Nano", generation: "2nd Gen",
			indices: []int{i("iPod_19.1.1.3.ipsw"), i("iPod_29.1.1.3.ipsw")}},
		{family: "iPod Nano", generation: "3rd Gen",
			indices: []int{i("iPod_26.1.1.3.ipsw")}},
		{family: "iPod Nano", generation: "4th Gen",
			indices: []int{i("iPod_31.1.0.4.ipsw")}},
		{family: "iPod Nano", generation: "5th Gen",
			indices: []int{i("iPod_1.0.2_34A20020.ipsw")}},
		{family: "iPod Nano", generation: "6th Gen",
			indices: []int{i("iPod_1.2_36B10147.ipsw")}},
		{family: "iPod Nano", generation: "7th Gen", models: []string{"MD475", "MD476", "MD477", "MD478", "MD479", "MD480", "MD481", "MD744", "ME971"},
			indices: []int{i("iPod_1.0.4_37A40005.ipsw")}},
		{family: "iPod Nano", generation: "7th Gen", models: []string{"MKMV2", "MKMX2", "MKN02", "MKN22", "MKN52", "MKN72"},
			indices: []int{i("iPod_1.1.2_39A10023.ipsw")}},

		// iPod Shuffle
		{family: "iPod Shuffle", generation: "1st Gen", models: []string{"M9724"},
			indices: []int{i("iPod_128.1.1.5.ipsw")}},
		{family: "iPod Shuffle", generation: "1st Gen", models: []string{"M9725"},
			indices: []int{i("iPod_129.1.1.5.ipsw")}},
		{family: "iPod Shuffle", generation: "2nd Gen",
			indices: []int{i("iPod_130.1.0.4.ipsw"), i("iPod_131.1.0.4.ipsw"), i("iPod_133.1.0.4.ipsw")}},
		{family: "iPod Shuffle", generation: "3rd Gen",
			indices: []int{i("iPod_132.1.1.ipsw")}},
		{family: "iPod Shuffle", generation: "4th Gen",
			indices: []int{i("iPod_134.1.0.1.ipsw"), i("iPod_135.1.0.2.ipsw"), i("iPod_136.1.0.3.ipsw")}},
	}
}

type FirmwareMatch struct {
	Entry IPSWEntry `json:"entry"`
	Index int       `json:"index"`
}

func ModelForFirmwareIndex(index int) *IPodModel {
	ensureFirmwareRules()
	if index < 0 || index >= len(catalog) {
		return nil
	}
	for _, rule := range firmwareRules {
		for _, idx := range rule.indices {
			if idx == index {
				return ModelByFamilyGeneration(rule.family, rule.generation)
			}
		}
	}
	return nil
}

func MatchFirmware(family, generation, modelNum string) []FirmwareMatch {
	ensureFirmwareRules()
	family = strings.TrimSpace(family)
	generation = strings.TrimSpace(generation)

	var matches []FirmwareMatch

	for _, rule := range firmwareRules {
		if !strings.EqualFold(rule.family, family) || !strings.EqualFold(rule.generation, generation) {
			continue
		}

		if len(rule.models) > 0 && modelNum != "" {
			found := false
			for _, m := range rule.models {
				if strings.EqualFold(m, modelNum) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		for _, idx := range rule.indices {
			if idx >= 0 && idx < len(catalog) {
				matches = append(matches, FirmwareMatch{Entry: catalog[idx], Index: idx})
			}
		}
	}

	if len(matches) == 0 {
		seen := make(map[int]bool)
		for _, rule := range firmwareRules {
			if !strings.EqualFold(rule.family, family) || !strings.EqualFold(rule.generation, generation) {
				continue
			}
			for _, idx := range rule.indices {
				if idx >= 0 && idx < len(catalog) && !seen[idx] {
					seen[idx] = true
					matches = append(matches, FirmwareMatch{Entry: catalog[idx], Index: idx})
				}
			}
		}
	}

	return matches
}
