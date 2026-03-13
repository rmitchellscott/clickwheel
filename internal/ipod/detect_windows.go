package ipod

import (
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

func DetectAll() ([]*DeviceInfo, error) {
	var devices []*DeviceInfo
	for letter := 'D'; letter <= 'Z'; letter++ {
		mount := fmt.Sprintf("%c:\\", letter)
		controlDir := filepath.Join(mount, "iPod_Control")
		if _, err := os.Stat(controlDir); err == nil {
			di, err := deviceInfoFromMount(mount, fmt.Sprintf("iPod (%c:)", letter))
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

	var free, total, available uint64
	path, _ := windows.UTF16PtrFromString(mount)
	err := windows.GetDiskFreeSpaceEx(path,
		(*uint64)(unsafe.Pointer(&available)),
		(*uint64)(unsafe.Pointer(&total)),
		(*uint64)(unsafe.Pointer(&free)),
	)
	if err == nil {
		di.FreeSpace = int64(free)
		di.TotalSpace = int64(total)
	}

	fillDeviceInfo(di, ReadSysInfo(mount))
	return di, nil
}
