package restore

import (
	"fmt"
	"os/exec"
	"strings"
)

func RawDiskPath(mountPoint string) (string, error) {
	driveLetter := strings.TrimRight(mountPoint, "\\")
	if len(driveLetter) < 2 {
		return "", fmt.Errorf("invalid mount point: %s", mountPoint)
	}

	script := fmt.Sprintf(
		`Get-Partition -DriveLetter '%c' | Get-Disk | Select-Object -ExpandProperty Number`,
		driveLetter[0],
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("powershell get disk: %s", string(out))
	}

	diskNum := strings.TrimSpace(string(out))
	return fmt.Sprintf(`\\.\PhysicalDrive%s`, diskNum), nil
}

func RenameVolume(mountPoint, label string) error {
	sanitized := sanitizeVolumeLabel(label)
	driveLetter := strings.TrimRight(mountPoint, "\\")
	out, err := exec.Command("label", driveLetter, sanitized).CombinedOutput()
	if err != nil {
		return fmt.Errorf("label: %s", string(out))
	}
	return nil
}

func UnmountDisk(mountPoint string) error {
	driveLetter := strings.TrimRight(mountPoint, "\\")
	out, err := exec.Command("mountvol", driveLetter+"\\", "/P").CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmount: %s", string(out))
	}
	return nil
}

func partitionAndFormatDirect(rawDiskPath string, firmwarePartSize int64, sectorSize int, volumeLabel string) error {
	return openAndPartitionWithDiskFS(rawDiskPath, firmwarePartSize, sectorSize, volumeLabel)
}

func MountDataPartition(_ string) error { return nil }

func CheckFullDiskAccess(_ string) bool { return true }
