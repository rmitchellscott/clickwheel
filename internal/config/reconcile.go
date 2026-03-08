package config

import (
	"crypto/rand"
	"fmt"
	"os"
)

func ResolveDeviceID(mountPoint, serial string) (string, error) {
	if serial != "" {
		return serial, nil
	}

	ipodPath := IPodConfigPath(mountPoint)
	if cfg, err := loadDeviceFrom(ipodPath); err == nil && cfg.DeviceID != "" {
		return cfg.DeviceID, nil
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func LoadDeviceConfig(mountPoint, deviceID string) (*DeviceConfig, error) {
	ipodPath := IPodConfigPath(mountPoint)
	backupPath, err := DeviceBackupPath(deviceID)
	if err != nil {
		return nil, err
	}

	ipodCfg, ipodErr := loadDeviceFrom(ipodPath)
	backupCfg, backupErr := loadDeviceFrom(backupPath)

	hasIPod := ipodErr == nil && ipodCfg != nil
	hasBackup := backupErr == nil && backupCfg != nil

	var winner *DeviceConfig
	switch {
	case hasIPod && hasBackup:
		if ipodCfg.LastModified >= backupCfg.LastModified {
			winner = ipodCfg
		} else {
			winner = backupCfg
		}
	case hasIPod:
		winner = ipodCfg
	case hasBackup:
		winner = backupCfg
	default:
		winner = DefaultDevice(deviceID)
	}

	winner.DeviceID = deviceID

	if err := os.MkdirAll(IPodConfigDir(mountPoint), 0755); err != nil {
		return nil, err
	}
	winner.path = ipodPath
	_ = winner.Save()
	if bp, err := DeviceBackupPath(deviceID); err == nil {
		winner.path = bp
		_ = winner.Save()
	}

	return winner, nil
}

func IPodConfigDir(mountPoint string) string {
	return fmt.Sprintf("%s/iPod_Control/Clickwheel", mountPoint)
}
