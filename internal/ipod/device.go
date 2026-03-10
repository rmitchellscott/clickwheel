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

	pcPath := filepath.Join(info.MountPoint, "iPod_Control", "iTunes", "Play Counts")
	if pcData, err := os.ReadFile(pcPath); err == nil {
		if entries, err := itunesdb.ParsePlayCounts(pcData); err == nil {
			itunesdb.MergePlayCounts(db, entries)
		}
	}

	return &Device{Info: info, DB: db}, nil
}

func (d *Device) Capabilities() *itunesdb.DeviceCapabilities {
	caps := itunesdb.CapabilitiesForFamilyGen(d.Info.Family, d.Info.Generation)
	if caps == nil {
		caps = itunesdb.DefaultCapabilities()
	}
	copied := *caps
	if copied.FirewireID == "" {
		if d.Info.FirewireGUID != "" {
			copied.FirewireID = d.Info.FirewireGUID
		} else if d.Info.SerialNumber != "" {
			copied.FirewireID = d.Info.SerialNumber
		}
	}
	return &copied
}

func (d *Device) Save() error {
	itunesDir := filepath.Join(d.Info.MountPoint, "iPod_Control", "iTunes")
	if err := os.MkdirAll(itunesDir, 0755); err != nil {
		return err
	}

	caps := d.Capabilities()
	data := itunesdb.SerializeDatabase(d.DB, caps)
	if err := os.WriteFile(filepath.Join(itunesDir, "iTunesDB"), data, 0644); err != nil {
		return err
	}

	os.Remove(filepath.Join(itunesDir, "Play Counts"))
	return nil
}
