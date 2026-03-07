package ipod

import (
	"os"
	"path/filepath"

	"clickwheel/internal/ipod/itunesdb"
)

type Device struct {
	Info *DeviceInfo
	DB   *itunesdb.Database
}

func OpenDevice(info *DeviceInfo) (*Device, error) {
	dbPath := filepath.Join(info.MountPoint, "iPod_Control", "iTunes", "iTunesDB")

	data, err := os.ReadFile(dbPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Device{
				Info: info,
				DB:   itunesdb.NewDatabase(),
			}, nil
		}
		return nil, err
	}

	db, err := itunesdb.Parse(data)
	if err != nil {
		return nil, err
	}

	return &Device{Info: info, DB: db}, nil
}

func (d *Device) Save() error {
	itunesDir := filepath.Join(d.Info.MountPoint, "iPod_Control", "iTunes")
	if err := os.MkdirAll(itunesDir, 0755); err != nil {
		return err
	}

	data := d.DB.Serialize()
	return os.WriteFile(filepath.Join(itunesDir, "iTunesDB"), data, 0644)
}
