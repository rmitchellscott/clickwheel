package restore

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func DetectUSBIPods() ([]USBIPod, error) {
	// Use PowerShell to find USB devices with Apple VID
	script := fmt.Sprintf(`Get-PnpDevice -Class USB | Where-Object { $_.HardwareID -match 'VID_%04X' } | ForEach-Object { $_.HardwareID[0] }`, appleVID)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).CombinedOutput()
	if err != nil {
		return nil, err
	}

	var results []USBIPod
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		pidIdx := strings.Index(strings.ToUpper(line), "PID_")
		if pidIdx == -1 {
			continue
		}
		pidStr := line[pidIdx+4:]
		if len(pidStr) > 4 {
			pidStr = pidStr[:4]
		}
		pid, _ := strconv.ParseUint(pidStr, 16, 16)

		model, mode := ModelByPID(uint16(pid))
		if model == nil {
			continue
		}

		results = append(results, USBIPod{Model: model, Mode: mode})
	}

	return results, nil
}
