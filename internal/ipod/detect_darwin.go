package ipod

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func Detect() (*DeviceInfo, error) {
	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mount := filepath.Join("/Volumes", entry.Name())
		controlDir := filepath.Join(mount, "iPod_Control")
		if _, err := os.Stat(controlDir); err == nil {
			return deviceInfoFromMount(mount, entry.Name())
		}
	}

	return nil, nil
}

func deviceInfoFromMount(mount, name string) (*DeviceInfo, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(mount, &stat); err != nil {
		return &DeviceInfo{MountPoint: mount, Name: name}, nil
	}

	return &DeviceInfo{
		MountPoint: mount,
		Name:       name,
		FreeSpace:  int64(stat.Bfree) * int64(stat.Bsize),
		TotalSpace: int64(stat.Blocks) * int64(stat.Bsize),
	}, nil
}
