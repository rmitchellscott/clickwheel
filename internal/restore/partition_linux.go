package restore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RawDiskPath(mountPoint string) (string, error) {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("read /proc/mounts: %w", err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == mountPoint {
			dev := fields[0]
			base := filepath.Base(dev)
			for i := len(base) - 1; i >= 0; i-- {
				if base[i] < '0' || base[i] > '9' {
					return filepath.Join("/dev", base[:i+1]), nil
				}
			}
			return dev, nil
		}
	}

	return "", fmt.Errorf("mount point %s not found in /proc/mounts", mountPoint)
}

func RenameVolume(mountPoint, label string) error {
	sanitized := sanitizeVolumeLabel(label)
	rawDisk, err := RawDiskPath(mountPoint)
	if err != nil {
		return err
	}
	partDev := rawDisk + "2"
	out, err := exec.Command("fatlabel", partDev, sanitized).CombinedOutput()
	if err != nil {
		return fmt.Errorf("fatlabel: %s", string(out))
	}
	return nil
}

func UnmountDisk(mountPoint string) error {
	out, err := exec.Command("umount", mountPoint).CombinedOutput()
	if err != nil {
		return fmt.Errorf("umount: %s", string(out))
	}
	return nil
}

func partitionAndFormatDirect(rawDiskPath string, firmwarePartSize int64, sectorSize int, volumeLabel string) error {
	return openAndPartitionWithDiskFS(rawDiskPath, firmwarePartSize, sectorSize, volumeLabel)
}

func MountDataPartition(_ string) error { return nil }

func FindMountPoint(rawDiskPath string) (string, error) {
	partDev := rawDiskPath + "2"
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return "", fmt.Errorf("read /proc/mounts: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == partDev {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("partition %s not found in /proc/mounts", partDev)
}
