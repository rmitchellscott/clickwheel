package restore

import (
	"fmt"
	"os"
	"strconv"
)

func PartitionAndFormat(rawDiskPath string, firmwarePartSize int64, sectorSize int, volumeLabel string) error {
	if os.Getuid() == 0 {
		return partitionAndFormatDirect(rawDiskPath, firmwarePartSize, sectorSize, volumeLabel)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not find executable path: %w", err)
	}

	out, err := RunPrivileged(exe,
		"--restore-partition",
		rawDiskPath,
		strconv.FormatInt(firmwarePartSize, 10),
		strconv.Itoa(sectorSize),
		volumeLabel,
	)
	if err != nil {
		return fmt.Errorf("partitioning failed: %s", string(out))
	}
	return nil
}

func RunPartitionSubcommand(args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("usage: --restore-partition <disk> <fwSize> <sectorSize> <label>")
	}
	rawDiskPath := args[0]
	firmwarePartSize, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid firmware partition size: %w", err)
	}
	sectorSize, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("invalid sector size: %w", err)
	}
	volumeLabel := args[3]

	return partitionAndFormatDirect(rawDiskPath, firmwarePartSize, sectorSize, volumeLabel)
}


func sanitizeVolumeLabel(name string) string {
	if name == "" {
		return "IPOD"
	}
	var label []byte
	for _, c := range name {
		if c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			label = append(label, byte(c))
		} else if c >= 'a' && c <= 'z' {
			label = append(label, byte(c-32))
		}
		if len(label) >= 11 {
			break
		}
	}
	if len(label) == 0 {
		return "IPOD"
	}
	return string(label)
}
