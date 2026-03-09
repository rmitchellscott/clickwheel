package restore

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func writeFirmwareDirect(rawDiskPath, fwFilePath string, sectorSize int) error {
	exec.Command("diskutil", "unmountDisk", "force", rawDiskPath).Run()

	fw, err := os.ReadFile(fwFilePath)
	if err != nil {
		return fmt.Errorf("read firmware: %w", err)
	}

	fw = alignToSector(fw, sectorSize)

	f, err := openDiskExclusive(rawDiskPath)
	if err != nil {
		return fmt.Errorf("open raw disk: %w", err)
	}

	if _, err := f.WriteAt(fw, int64(sectorSize)); err != nil {
		f.Close()
		return fmt.Errorf("write firmware: %w", err)
	}
	f.Close()

	time.Sleep(2 * time.Second)
	exec.Command("diskutil", "unmountDisk", "force", rawDiskPath).Run()

	return nil
}
