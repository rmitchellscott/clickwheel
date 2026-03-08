package ipod

import (
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func Detect() (*DeviceInfo, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}

	searchDirs := []string{
		filepath.Join("/media", u.Username),
		filepath.Join("/run/media", u.Username),
	}

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			mount := filepath.Join(dir, entry.Name())
			controlDir := filepath.Join(mount, "iPod_Control")
			if _, err := os.Stat(controlDir); err == nil {
				return deviceInfoFromMount(mount, entry.Name())
			}
		}
	}

	return nil, nil
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
