package restore

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func RawDiskPath(mountPoint string) (string, error) {
	out, err := exec.Command("diskutil", "info", "-plist", mountPoint).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("diskutil info: %w", err)
	}

	outStr := string(out)
	idx := strings.Index(outStr, "<key>ParentWholeDisk</key>")
	if idx == -1 {
		return "", fmt.Errorf("could not find ParentWholeDisk in diskutil output")
	}
	rest := outStr[idx:]
	start := strings.Index(rest, "<string>")
	end := strings.Index(rest, "</string>")
	if start == -1 || end == -1 {
		return "", fmt.Errorf("could not parse ParentWholeDisk value")
	}
	diskID := rest[start+8 : end]

	return "/dev/" + diskID, nil
}

func RenameVolume(mountPoint, label string) error {
	sanitized := sanitizeVolumeLabel(label)
	out, err := exec.Command("diskutil", "rename", mountPoint, sanitized).CombinedOutput()
	if err != nil {
		return fmt.Errorf("diskutil rename: %s", string(out))
	}
	return nil
}

func UnmountDisk(mountPoint string) error {
	out, err := exec.Command("diskutil", "unmountDisk", "force", mountPoint).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmount: %s", string(out))
	}
	return nil
}

func rdiskPath(diskPath string) string {
	return strings.Replace(diskPath, "/dev/disk", "/dev/rdisk", 1)
}

func openDiskExclusive(diskPath string) (*os.File, error) {
	fd, err := syscall.Open(rdiskPath(diskPath), syscall.O_RDWR|syscall.O_EXLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", rdiskPath(diskPath), err)
	}
	f := os.NewFile(uintptr(fd), rdiskPath(diskPath))
	syscall.Syscall(syscall.SYS_FCNTL, f.Fd(), syscall.F_NOCACHE, 1)
	return f, nil
}

func alignToSector(data []byte, sectorSize int) []byte {
	remainder := len(data) % sectorSize
	if remainder == 0 {
		return data
	}
	padded := make([]byte, len(data)+sectorSize-remainder)
	copy(padded, data)
	return padded
}

func partitionAndFormatDirect(rawDiskPath string, firmwarePartSize int64, sectorSize int, volumeLabel string) error {
	exec.Command("diskutil", "unmountDisk", "force", rawDiskPath).Run()

	if err := openAndPartitionWithDiskFS(rawDiskPath, firmwarePartSize, sectorSize, volumeLabel); err != nil {
		return err
	}

	time.Sleep(2 * time.Second)
	exec.Command("diskutil", "unmountDisk", "force", rawDiskPath).Run()
	exec.Command("diskutil", "mount", rawDiskPath+"s2").Run()

	return nil
}

func MountDataPartition(rawDiskPath string) {
	exec.Command("diskutil", "mount", rawDiskPath+"s2").Run()
}

func CheckFullDiskAccess(_ string) bool {
	return true
}
