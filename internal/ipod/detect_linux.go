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
