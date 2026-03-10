package restore

import (
	"os"
	"path/filepath"
	"strings"
)

const deviceNameFile = ".clickwheel-name"

func WriteDeviceName(mountPoint, name string) {
	_ = os.WriteFile(filepath.Join(mountPoint, deviceNameFile), []byte(name), 0644)
}

func ReadPendingDeviceName(mountPoint string) string {
	data, err := os.ReadFile(filepath.Join(mountPoint, deviceNameFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func ClearPendingDeviceName(mountPoint string) {
	os.Remove(filepath.Join(mountPoint, deviceNameFile))
}
