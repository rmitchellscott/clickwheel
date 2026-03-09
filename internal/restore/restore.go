package restore

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"clickwheel/internal/ipod"
)

type RestoreState string

const (
	StateIdle                RestoreState = "idle"
	StateDetecting           RestoreState = "detecting"
	StateDownloadingFirmware RestoreState = "downloading_firmware"
	StatePartitioning        RestoreState = "partitioning"
	StateFlashingFirmware    RestoreState = "flashing_firmware"
	StateInitializing        RestoreState = "initializing"
	StateComplete            RestoreState = "complete"
	StateError               RestoreState = "error"
)

type RestoreProgress struct {
	State    RestoreState `json:"state"`
	Message  string       `json:"message"`
	Percent  float64      `json:"percent"`
	CanRetry bool         `json:"canRetry"`
	Error    string       `json:"error,omitempty"`
}

type RestoreEngine struct {
	ipswEntry  IPSWEntry
	model      *IPodModel
	deviceName string
	rawDisk    string
	onProgress func(RestoreProgress)
}

func NewRestoreEngine(entry IPSWEntry, model *IPodModel, deviceName string, onProgress func(RestoreProgress)) *RestoreEngine {
	return &RestoreEngine{
		ipswEntry:  entry,
		model:      model,
		deviceName: deviceName,
		onProgress: onProgress,
	}
}

func (e *RestoreEngine) SetRawDisk(path string) {
	e.rawDisk = path
}

func (e *RestoreEngine) emit(state RestoreState, message string, percent float64) {
	if e.onProgress != nil {
		e.onProgress(RestoreProgress{
			State:   state,
			Message: message,
			Percent: percent,
		})
	}
}

func (e *RestoreEngine) emitError(message string, canRetry bool) {
	if e.onProgress != nil {
		e.onProgress(RestoreProgress{
			State:    StateError,
			Message:  message,
			Error:    message,
			CanRetry: canRetry,
		})
	}
}

func (e *RestoreEngine) Run(ctx context.Context) error {
	// Step 1: Download firmware FIRST (non-destructive)
	e.emit(StateDownloadingFirmware, "Downloading firmware...", 5)
	ipswPath, err := DownloadIPSW(e.ipswEntry, func(downloaded, total int64) {
		if total > 0 {
			pct := 5.0 + (float64(downloaded)/float64(total))*25.0
			e.emit(StateDownloadingFirmware,
				fmt.Sprintf("Downloading firmware... %d%%", int(float64(downloaded)/float64(total)*100)),
				pct)
		}
	})
	if err != nil {
		e.emitError(fmt.Sprintf("Failed to download firmware: %v", err), true)
		return err
	}

	// Prepare firmware cache file (non-destructive)
	fwCachePath, err := writeFirmwareToCache(ipswPath)
	if err != nil {
		e.emitError(fmt.Sprintf("Failed to prepare firmware: %v", err), true)
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 2: Detect iPod and resolve raw disk
	rawDisk := e.rawDisk
	if rawDisk == "" {
		e.emit(StateDetecting, "Detecting iPod...", 30)
		info, err := ipod.Detect()
		if err == nil && info != nil {
			rawDisk, err = RawDiskPath(info.MountPoint)
			if err != nil {
				e.emitError(fmt.Sprintf("Could not determine raw disk path: %v", err), true)
				return err
			}
			e.emit(StatePartitioning, "Unmounting iPod...", 35)
			if err := UnmountDisk(info.MountPoint); err != nil {
				e.emitError(fmt.Sprintf("Failed to unmount: %v", err), true)
				return err
			}
		} else {
			usbIPods, usbErr := DetectUSBIPods()
			if usbErr != nil || len(usbIPods) == 0 || usbIPods[0].DiskPath == "" {
				e.emitError("iPod not detected. Please connect your iPod and enter disk mode.", true)
				return fmt.Errorf("iPod not detected")
			}
			rawDisk = usbIPods[0].DiskPath
		}
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// Step 3: Partition + write firmware (destructive — do both without pause)
	e.emit(StatePartitioning, "Partitioning disk...", 40)
	if err := PartitionAndFormat(rawDisk, e.model.FirmwarePartSize, e.model.SectorSize, e.deviceName); err != nil {
		e.emitError(fmt.Sprintf("Failed to partition: %v", err), true)
		return err
	}

	e.emit(StateFlashingFirmware, "Writing firmware...", 55)
	if err := WriteFirmwarePartition(rawDisk, nil, fwCachePath, e.model.SectorSize); err != nil {
		e.emitError(fmt.Sprintf("Failed to write firmware: %v", err), true)
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	e.emit(StateComplete, "Restore complete! Disconnect and plug into a wall charger to initialize.", 100)
	return nil
}

func writeFirmwareToCache(ipswPath string) (string, error) {
	cacheDir, err := ipswCacheDir()
	if err != nil {
		return "", err
	}
	fwPath := filepath.Join(cacheDir, filepath.Base(ipswPath)+".fw")

	r, err := zip.OpenReader(ipswPath)
	if err != nil {
		return "", fmt.Errorf("open IPSW: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "Firmware") {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("open firmware in IPSW: %w", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("read firmware from IPSW: %w", err)
			}
			if err := os.WriteFile(fwPath, data, 0644); err != nil {
				return "", fmt.Errorf("write firmware cache: %w", err)
			}
			return fwPath, nil
		}
	}

	return "", fmt.Errorf("no firmware file found in IPSW")
}
