package vpd

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation

#include <IOKit/IOKitLib.h>
#include <IOKit/scsi/SCSITaskLib.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>
#include <Availability.h>

#if !defined(__MAC_12_0) || __MAC_OS_X_VERSION_MIN_REQUIRED < __MAC_12_0
#define kIOMainPortDefault kIOMasterPortDefault
#endif

typedef struct {
	IOCFPlugInInterface **plugIn;
	SCSITaskDeviceInterface **task;
	SCSITaskInterface **cmd;
} scsi_session_t;

static io_service_t find_ipod_service() {
	CFMutableDictionaryRef match = IOServiceMatching("com_apple_driver_iPodSBCNub");
	if (!match) return 0;
	io_service_t svc = IOServiceGetMatchingService(kIOMainPortDefault, match);
	return svc;
}

static int get_usb_serial(io_service_t service, char *buf, int bufLen) {
	io_service_t current = service;
	IOObjectRetain(current);
	CFStringRef key = CFStringCreateWithCString(kCFAllocatorDefault, "USB Serial Number", kCFStringEncodingUTF8);

	for (int depth = 0; depth < 15; depth++) {
		CFTypeRef val = IORegistryEntryCreateCFProperty(current, key, kCFAllocatorDefault, 0);
		if (val) {
			if (CFGetTypeID(val) == CFStringGetTypeID()) {
				CFStringGetCString((CFStringRef)val, buf, bufLen, kCFStringEncodingUTF8);
				CFRelease(val);
				IOObjectRelease(current);
				CFRelease(key);
				return 0;
			}
			CFRelease(val);
		}
		io_service_t next = 0;
		kern_return_t kr = IORegistryEntryGetParentEntry(current, kIOServicePlane, &next);
		IOObjectRelease(current);
		if (kr != KERN_SUCCESS) break;
		current = next;
	}
	CFRelease(key);
	return -1;
}

static int get_product_id(io_service_t service) {
	io_service_t current = service;
	IOObjectRetain(current);
	CFStringRef key = CFStringCreateWithCString(kCFAllocatorDefault, "idProduct", kCFStringEncodingUTF8);

	for (int depth = 0; depth < 15; depth++) {
		CFTypeRef val = IORegistryEntryCreateCFProperty(current, key, kCFAllocatorDefault, 0);
		if (val) {
			if (CFGetTypeID(val) == CFNumberGetTypeID()) {
				int result = 0;
				CFNumberGetValue((CFNumberRef)val, kCFNumberIntType, &result);
				CFRelease(val);
				IOObjectRelease(current);
				CFRelease(key);
				return result;
			}
			CFRelease(val);
		}
		io_service_t next = 0;
		kern_return_t kr = IORegistryEntryGetParentEntry(current, kIOServicePlane, &next);
		IOObjectRelease(current);
		if (kr != KERN_SUCCESS) break;
		current = next;
	}
	CFRelease(key);
	return -1;
}

static int open_scsi_session(io_service_t service, scsi_session_t *sess) {
	SInt32 score = 0;
	kern_return_t kr;
	HRESULT hr;

	memset(sess, 0, sizeof(*sess));

	kr = IOCreatePlugInInterfaceForService(service, kIOSCSITaskDeviceUserClientTypeID, kIOCFPlugInInterfaceID, &sess->plugIn, &score);
	if (kr != KERN_SUCCESS || !sess->plugIn) return -1;

	hr = (*sess->plugIn)->QueryInterface(sess->plugIn, CFUUIDGetUUIDBytes(kIOSCSITaskDeviceInterfaceID), (LPVOID *)&sess->task);
	if (hr != S_OK || !sess->task) return -2;

	kr = (*sess->task)->ObtainExclusiveAccess(sess->task);
	if (kr != kIOReturnSuccess) return -3;

	sess->cmd = (*sess->task)->CreateSCSITask(sess->task);
	if (!sess->cmd) return -4;

	return 0;
}

static int scsi_inquiry(scsi_session_t *sess, const unsigned char *cdb, int cdbLen, unsigned char *buf, int bufLen) {
	memset(buf, 0, bufLen);

	SCSITaskSGElement sg;
	sg.address = (uintptr_t)buf;
	sg.length = bufLen;

	SCSICommandDescriptorBlock taskCDB;
	memset(&taskCDB, 0, sizeof(taskCDB));
	memcpy(taskCDB, cdb, cdbLen < (int)sizeof(taskCDB) ? cdbLen : (int)sizeof(taskCDB));

	kern_return_t kr;
	kr = (*sess->cmd)->SetCommandDescriptorBlock(sess->cmd, taskCDB, cdbLen);
	if (kr != kIOReturnSuccess) return -1;

	kr = (*sess->cmd)->SetScatterGatherEntries(sess->cmd, &sg, 1, bufLen, kSCSIDataTransfer_FromTargetToInitiator);
	if (kr != kIOReturnSuccess) return -2;

	kr = (*sess->cmd)->SetTimeoutDuration(sess->cmd, 10000);
	if (kr != kIOReturnSuccess) return -3;

	SCSITaskStatus taskStatus;
	UInt64 bytesTransferred = 0;
	SCSI_Sense_Data senseData;
	memset(&senseData, 0, sizeof(senseData));

	kr = (*sess->cmd)->ExecuteTaskSync(sess->cmd, &senseData, &taskStatus, &bytesTransferred);
	if (kr != kIOReturnSuccess) return -4;

	(*sess->cmd)->ResetForNewTask(sess->cmd);
	return (int)bytesTransferred;
}

static void close_scsi_session(scsi_session_t *sess) {
	if (sess->cmd) {
		(*sess->cmd)->Release(sess->cmd);
		sess->cmd = NULL;
	}
	if (sess->task) {
		(*sess->task)->ReleaseExclusiveAccess(sess->task);
		(*sess->task)->Release(sess->task);
		sess->task = NULL;
	}
	if (sess->plugIn) {
		IODestroyPlugInInterface(sess->plugIn);
		sess->plugIn = NULL;
	}
}
*/
import "C"

import (
	"fmt"
	"log"
	"strings"
	"unsafe"
)

var usbPIDToFamilyID = map[int]int{
	0x1200: 1,  // iPod 1st Gen
	0x1201: 1,  // iPod 1st/2nd Gen
	0x1203: 4,  // iPod 4th Gen (Click Wheel)
	0x1204: 5,  // iPod Photo/Color
	0x1205: 3,  // iPod Mini
	0x1209: 6,  // iPod Video 5th Gen
	0x120A: 7,  // iPod Nano 1st Gen
	0x1260: 9,  // iPod Nano 2nd Gen
	0x1261: 11, // iPod Classic
	0x1265: 15, // iPod Nano 4th Gen
	0x1267: 16, // iPod Nano 5th Gen
}

func QueryVPD(mountPoint string) (*VPDInfo, error) {
	svc := C.find_ipod_service()
	if svc == 0 {
		return nil, fmt.Errorf("vpd: no iPod IOKit service found")
	}
	defer C.IOObjectRelease(svc)

	var usbSerial string
	var serialBuf [256]C.char
	if C.get_usb_serial(svc, &serialBuf[0], 256) == 0 {
		usbSerial = strings.TrimSpace(C.GoString(&serialBuf[0]))
	}

	productID := int(C.get_product_id(svc))

	info, err := queryViaSCSI(svc)
	if err != nil {
		log.Printf("[vpd] SCSI query failed: %v, using USB properties", err)
		info = buildFromUSBProperties(usbSerial, productID)
	}
	if info == nil {
		return nil, fmt.Errorf("vpd: no device info available")
	}

	if usbSerial != "" {
		info.USBSerial = usbSerial
	}

	if info.FamilyID == 0 {
		if fid, ok := usbPIDToFamilyID[productID]; ok {
			info.FamilyID = fid
		}
	}

	return info, nil
}

func queryViaSCSI(svc C.io_service_t) (*VPDInfo, error) {
	var sess C.scsi_session_t
	rc := C.open_scsi_session(svc, &sess)
	if rc != 0 {
		return nil, fmt.Errorf("open session failed (rc=%d)", rc)
	}
	defer C.close_scsi_session(&sess)

	inquiry := func(cdb [6]byte, bufSize int) ([]byte, error) {
		buf := make([]byte, bufSize)
		n := C.scsi_inquiry(&sess, (*C.uchar)(unsafe.Pointer(&cdb[0])), 6, (*C.uchar)(unsafe.Pointer(&buf[0])), C.int(bufSize))
		if n < 0 {
			return nil, fmt.Errorf("INQUIRY failed (rc=%d)", n)
		}
		return buf[:n], nil
	}

	plistData, err := readVPDPages(inquiry)
	if err != nil {
		return nil, err
	}

	return parseVPDPlist(plistData)
}

func buildFromUSBProperties(usbSerial string, productID int) *VPDInfo {
	if usbSerial == "" && productID < 0 {
		return nil
	}

	info := &VPDInfo{}

	if len(usbSerial) > 16 {
		info.FireWireGUID = usbSerial[:16]
		info.SerialNumber = usbSerial[16:]
	} else if usbSerial != "" {
		info.FireWireGUID = usbSerial
	}

	if fid, ok := usbPIDToFamilyID[productID]; ok {
		info.FamilyID = fid
	}

	return info
}
