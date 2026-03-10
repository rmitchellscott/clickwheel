package restore

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

func CheckPrivilege() PrivilegeResult {
	cmd := exec.Command("net", "session")
	if err := cmd.Run(); err != nil {
		return PrivilegeResult{Granted: false, Error: fmt.Errorf("not running as administrator")}
	}
	return PrivilegeResult{Granted: true}
}

func SetPassword(_ string) {}
func ClearPassword()       {}

func RunPrivileged(command string, args ...string) ([]byte, error) {
	verb := "runas"
	argStr := strings.Join(args, " ")

	shellExecute := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	cmdPtr, _ := syscall.UTF16PtrFromString(command)
	argsPtr, _ := syscall.UTF16PtrFromString(argStr)

	ret, _, err := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(cmdPtr)),
		uintptr(unsafe.Pointer(argsPtr)),
		0,
		1,
	)

	if ret <= 32 {
		return nil, fmt.Errorf("ShellExecute failed: %v", err)
	}

	return nil, nil
}
