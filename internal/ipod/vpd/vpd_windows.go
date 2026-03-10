package vpd

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ioctlScsiPassThroughDirect  = 0x4D014
	ioctlVolumeGetDiskExtents   = 0x560000
	scsiIoctlDataIn             = 1
)

type scsiPassThroughDirect struct {
	Length             uint16
	ScsiStatus         uint8
	PathId             uint8
	TargetId           uint8
	Lun                uint8
	CdbLength          uint8
	SenseInfoLength    uint8
	DataIn             uint8
	_pad1              [3]uint8
	DataTransferLength uint32
	TimeOutValue       uint32
	DataBuffer         uintptr
	SenseInfoOffset    uint32
	Cdb                [16]uint8
}

type scsiPassThroughDirectWithSense struct {
	sptd  scsiPassThroughDirect
	sense [32]uint8
}

type diskExtent struct {
	DiskNumber     uint32
	StartingOffset int64
	ExtentLength   int64
}

func resolvePhysicalDrive(driveLetter byte) (string, error) {
	volumePath, err := windows.UTF16PtrFromString(fmt.Sprintf(`\\.\%c:`, driveLetter))
	if err != nil {
		return "", err
	}

	h, err := windows.CreateFile(volumePath, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return "", fmt.Errorf("vpd: open volume %c: %w", driveLetter, err)
	}
	defer windows.CloseHandle(h)

	buf := make([]byte, 256)
	var returned uint32
	err = windows.DeviceIoControl(h, ioctlVolumeGetDiskExtents, nil, 0, &buf[0], uint32(len(buf)), &returned, nil)
	if err != nil {
		return "", fmt.Errorf("vpd: get disk extents: %w", err)
	}

	count := binary.LittleEndian.Uint32(buf[0:4])
	if count == 0 {
		return "", fmt.Errorf("vpd: no disk extents")
	}

	diskNum := binary.LittleEndian.Uint32(buf[8:12])
	return fmt.Sprintf(`\\.\PhysicalDrive%d`, diskNum), nil
}

func QueryVPD(mountPoint string) (*VPDInfo, error) {
	if len(mountPoint) == 0 {
		return nil, fmt.Errorf("vpd: empty mount point")
	}
	driveLetter := mountPoint[0]

	physDrive, err := resolvePhysicalDrive(driveLetter)
	if err != nil {
		return nil, err
	}

	drivePath, err := windows.UTF16PtrFromString(physDrive)
	if err != nil {
		return nil, err
	}

	h, err := windows.CreateFile(drivePath, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("vpd: open %s: %w", physDrive, err)
	}
	defer windows.CloseHandle(h)

	inquiry := func(cdb [6]byte, bufSize int) ([]byte, error) {
		buf := make([]byte, bufSize)
		var sptwb scsiPassThroughDirectWithSense
		sptwb.sptd.Length = uint16(unsafe.Sizeof(sptwb.sptd))
		sptwb.sptd.CdbLength = 6
		sptwb.sptd.SenseInfoLength = uint8(len(sptwb.sense))
		sptwb.sptd.DataIn = scsiIoctlDataIn
		sptwb.sptd.DataTransferLength = uint32(bufSize)
		sptwb.sptd.TimeOutValue = 10
		sptwb.sptd.DataBuffer = uintptr(unsafe.Pointer(&buf[0]))
		sptwb.sptd.SenseInfoOffset = uint32(unsafe.Offsetof(sptwb.sense))
		copy(sptwb.sptd.Cdb[:], cdb[:])

		var returned uint32
		inBuf := (*byte)(unsafe.Pointer(&sptwb))
		inSize := uint32(unsafe.Sizeof(sptwb))
		err := windows.DeviceIoControl(h, ioctlScsiPassThroughDirect, inBuf, inSize, inBuf, inSize, &returned, nil)
		if err != nil {
			return nil, fmt.Errorf("SCSI pass-through failed: %w", err)
		}
		if sptwb.sptd.ScsiStatus != 0 {
			return nil, fmt.Errorf("SCSI status %d", sptwb.sptd.ScsiStatus)
		}

		return buf[:sptwb.sptd.DataTransferLength], nil
	}

	plistData, err := readVPDPages(inquiry)
	if err != nil {
		return nil, err
	}

	return parseVPDPlist(plistData)
}
