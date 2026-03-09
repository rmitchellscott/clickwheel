//go:build !darwin

package restore

import (
	"fmt"
	"os"
)

func writeFirmwareDirect(rawDiskPath, fwFilePath string, sectorSize int) error {
	data, err := os.ReadFile(fwFilePath)
	if err != nil {
		return fmt.Errorf("read firmware: %w", err)
	}

	f, err := os.OpenFile(rawDiskPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open disk: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteAt(data, int64(sectorSize)); err != nil {
		return fmt.Errorf("write firmware: %w", err)
	}

	return f.Sync()
}
