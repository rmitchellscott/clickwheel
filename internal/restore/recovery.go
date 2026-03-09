package restore

import (
	"fmt"
	"os"
	"strconv"
)

func WriteFirmwarePartition(rawDiskPath string, _ []byte, fwCachePath string, sectorSize int) error {
	if os.Getuid() == 0 {
		return writeFirmwareDirect(rawDiskPath, fwCachePath, sectorSize)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not find executable path: %w", err)
	}
	out, err := RunPrivileged(exe, "--restore-write-fw", rawDiskPath, fwCachePath, strconv.Itoa(sectorSize))
	if err != nil {
		return fmt.Errorf("firmware write failed: %s", string(out))
	}
	return nil
}

func RunWriteFirmwareSubcommand(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: --restore-write-fw <disk> <fwFile> <sectorSize>")
	}
	sectorSize, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("invalid sector size: %w", err)
	}
	return writeFirmwareDirect(args[0], args[1], sectorSize)
}
