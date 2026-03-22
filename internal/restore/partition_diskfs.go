package restore

import (
	"fmt"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

func writeMBR(d *disk.Disk, firmwarePartSize int64, sectorSize int) error {
	ss := int64(sectorSize)
	if ss == 0 {
		ss = 512
	}

	diskSize := d.Size
	firmwareSectors := firmwarePartSize / ss
	totalSectors := diskSize / ss
	fat32Start := firmwareSectors
	fat32Sectors := totalSectors - firmwareSectors - 1

	table := &mbr.Table{
		LogicalSectorSize:  int(ss),
		PhysicalSectorSize: int(ss),
		Partitions: []*mbr.Partition{
			{
				Bootable: false,
				Type:     mbr.Empty,
				Start:    1,
				Size:     uint32(firmwareSectors - 1),
			},
			{
				Bootable: false,
				Type:     mbr.Fat32LBA,
				Start:    uint32(fat32Start),
				Size:     uint32(fat32Sectors),
			},
		},
	}

	rwBackend, err := d.Backend.Writable()
	if err != nil {
		return fmt.Errorf("get writable backend: %w", err)
	}
	if err := table.Write(rwBackend, d.Size); err != nil {
		return fmt.Errorf("write partition table: %w", err)
	}
	d.Table = table
	return nil
}

func partitionWithDiskFS(d *disk.Disk, firmwarePartSize int64, sectorSize int, volumeLabel string) error {
	if err := writeMBR(d, firmwarePartSize, sectorSize); err != nil {
		return err
	}

	label := sanitizeVolumeLabel(volumeLabel)
	_, err := d.CreateFilesystem(disk.FilesystemSpec{
		Partition:   2,
		FSType:      filesystem.TypeFat32,
		VolumeLabel: label,
	})
	if err != nil {
		return fmt.Errorf("format FAT32: %w", err)
	}

	return nil
}

func openAndPartitionWithDiskFS(rawDiskPath string, firmwarePartSize int64, sectorSize int, volumeLabel string) error {
	ss := int64(sectorSize)
	if ss == 0 {
		ss = 512
	}

	d, err := diskfs.Open(rawDiskPath, diskfs.WithSectorSize(diskfs.SectorSize(ss)))
	if err != nil {
		return fmt.Errorf("open disk for partitioning: %w", err)
	}
	defer d.Close()

	return partitionWithDiskFS(d, firmwarePartSize, sectorSize, volumeLabel)
}

func openAndWriteMBR(rawDiskPath string, firmwarePartSize int64, sectorSize int) error {
	ss := int64(sectorSize)
	if ss == 0 {
		ss = 512
	}

	d, err := diskfs.Open(rawDiskPath, diskfs.WithSectorSize(diskfs.SectorSize(ss)))
	if err != nil {
		return fmt.Errorf("open disk for partitioning: %w", err)
	}
	defer d.Close()

	return writeMBR(d, firmwarePartSize, sectorSize)
}
