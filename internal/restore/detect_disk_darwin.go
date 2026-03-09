package restore

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var (
	reVendorID  = regexp.MustCompile(`"idVendor"\s*=\s*(\d+)`)
	reProductID = regexp.MustCompile(`"idProduct"\s*=\s*(\d+)`)
	reBSDName   = regexp.MustCompile(`"BSD Name"\s*=\s*"(disk\d+)"`)
)

func DetectUSBIPods() ([]USBIPod, error) {
	out, err := exec.Command("ioreg", "-r", "-c", "IOUSBHostDevice", "-l").CombinedOutput()
	if err != nil {
		return nil, err
	}

	blocks := splitAtIOUSBHostDevice(string(out))

	seen := make(map[uint16]bool)
	var results []USBIPod

	for _, block := range blocks {
		vidMatch := reVendorID.FindStringSubmatch(block)
		if vidMatch == nil {
			continue
		}
		vid, _ := strconv.ParseUint(vidMatch[1], 10, 16)
		if uint16(vid) != appleVID {
			continue
		}

		pidMatch := reProductID.FindStringSubmatch(block)
		if pidMatch == nil {
			continue
		}
		pid, _ := strconv.ParseUint(pidMatch[1], 10, 16)

		if seen[uint16(pid)] {
			continue
		}

		model, mode := ModelByPID(uint16(pid))
		if model == nil {
			continue
		}

		seen[uint16(pid)] = true
		ipod := USBIPod{Model: model, Mode: mode}

		if mode == ModeDisk {
			bsdMatch := reBSDName.FindStringSubmatch(block)
			if bsdMatch != nil {
				ipod.DiskPath = "/dev/" + bsdMatch[1]
			}
		}

		results = append(results, ipod)
	}

	return results, nil
}

func splitAtIOUSBHostDevice(output string) []string {
	marker := "<class IOUSBHostDevice,"
	lines := strings.Split(output, "\n")

	var blocks []string
	var current []string

	for _, line := range lines {
		if strings.Contains(line, marker) {
			if len(current) > 0 {
				blocks = append(blocks, strings.Join(current, "\n"))
			}
			current = []string{line}
		} else if len(current) > 0 {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		blocks = append(blocks, strings.Join(current, "\n"))
	}
	return blocks
}
