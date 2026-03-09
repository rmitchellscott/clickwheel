package restore

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func DetectUSBIPods() ([]USBIPod, error) {
	// Walk /sys/bus/usb/devices/ for Apple VID devices
	entries, err := filepath.Glob("/sys/bus/usb/devices/*/idVendor")
	if err != nil {
		return nil, err
	}

	var results []USBIPod
	for _, vendorFile := range entries {
		vidBytes, _ := os.ReadFile(vendorFile)
		vid, _ := strconv.ParseUint(strings.TrimSpace(string(vidBytes)), 16, 16)
		if uint16(vid) != appleVID {
			continue
		}

		dir := filepath.Dir(vendorFile)
		pidBytes, _ := os.ReadFile(filepath.Join(dir, "idProduct"))
		pid, _ := strconv.ParseUint(strings.TrimSpace(string(pidBytes)), 16, 16)

		model, mode := ModelByPID(uint16(pid))
		if model == nil {
			continue
		}

		ipod := USBIPod{Model: model, Mode: mode}

		if mode == ModeDisk {
			// Find the block device under this USB device
			blocks, _ := filepath.Glob(filepath.Join(dir, "*/*/block/*"))
			if len(blocks) > 0 {
				ipod.DiskPath = "/dev/" + filepath.Base(blocks[0])
			}
		}

		results = append(results, ipod)
	}

	return results, nil
}
