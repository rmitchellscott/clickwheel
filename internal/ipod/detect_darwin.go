package ipod

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func DetectAll() ([]*DeviceInfo, error) {
	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return nil, err
	}

	var devices []*DeviceInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mount := filepath.Join("/Volumes", entry.Name())
		controlDir := filepath.Join(mount, "iPod_Control")
		if _, err := os.Stat(controlDir); err == nil {
			di, err := deviceInfoFromMount(mount, entry.Name())
			if err != nil {
				continue
			}
			devices = append(devices, di)
		}
	}

	return devices, nil
}

func deviceInfoFromMount(mount, name string) (*DeviceInfo, error) {
	di := &DeviceInfo{MountPoint: mount, Name: name}

	var stat unix.Statfs_t
	if err := unix.Statfs(mount, &stat); err == nil {
		di.FreeSpace = int64(stat.Bfree) * int64(stat.Bsize)
		di.TotalSpace = int64(stat.Blocks) * int64(stat.Bsize)
	}

	fillDeviceInfo(di, ReadSysInfo(mount))
	return di, nil
}
