package vpd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	sgIO             = 0x2285
	sgDxferFromDev   = -3
	sgInterfaceIDVal = 'S'
)

type sgIOHdr struct {
	interfaceID    int32
	dxferDirection int32
	cmdLen         uint8
	mxSbLen        uint8
	ioxRefNr       uint16
	dxferLen       uint32
	dxferp         uintptr
	cmdp           uintptr
	sbp            uintptr
	timeout        uint32
	flags          uint32
	packID         int32
	usrPtr         uintptr
	status         uint8
	maskedStatus   uint8
	msgStatus      uint8
	sbLenWr        uint8
	hostStatus     uint16
	driverStatus   uint16
	resid          int32
	duration       uint32
	info           uint32
}

func resolveBlockDevice(mountPoint string) (string, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == mountPoint {
			dev := fields[0]
			for len(dev) > 0 && dev[len(dev)-1] >= '0' && dev[len(dev)-1] <= '9' {
				dev = dev[:len(dev)-1]
			}
			return dev, nil
		}
	}
	return "", fmt.Errorf("vpd: mount point %s not found in /proc/mounts", mountPoint)
}

func QueryVPD(mountPoint string) (*VPDInfo, error) {
	blockDev, err := resolveBlockDevice(mountPoint)
	if err != nil {
		return nil, err
	}

	fd, err := unix.Open(blockDev, unix.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("vpd: open %s: %w", blockDev, err)
	}
	defer unix.Close(fd)

	inquiry := func(cdb [6]byte, bufSize int) ([]byte, error) {
		buf := make([]byte, bufSize)
		senseBuf := make([]byte, 32)

		hdr := sgIOHdr{
			interfaceID:    sgInterfaceIDVal,
			dxferDirection: sgDxferFromDev,
			cmdLen:         6,
			mxSbLen:        uint8(len(senseBuf)),
			dxferLen:       uint32(bufSize),
			dxferp:         uintptr(unsafe.Pointer(&buf[0])),
			cmdp:           uintptr(unsafe.Pointer(&cdb[0])),
			sbp:            uintptr(unsafe.Pointer(&senseBuf[0])),
			timeout:        10000,
		}

		_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), sgIO, uintptr(unsafe.Pointer(&hdr)))
		if errno != 0 {
			return nil, fmt.Errorf("SG_IO ioctl failed: %v", errno)
		}
		if hdr.status != 0 {
			return nil, fmt.Errorf("SCSI status %d", hdr.status)
		}

		transferred := bufSize - int(hdr.resid)
		return buf[:transferred], nil
	}

	plistData, err := readVPDPages(inquiry)
	if err != nil {
		return nil, err
	}

	return parseVPDPlist(plistData)
}
