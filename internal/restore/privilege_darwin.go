package restore

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var (
	sudoPassword string
	passwordMu   sync.Mutex
)

func SetPassword(pw string) {
	passwordMu.Lock()
	sudoPassword = pw
	passwordMu.Unlock()
}

func ClearPassword() {
	SetPassword("")
}

func CheckPrivilege() PrivilegeResult {
	return PrivilegeResult{Granted: true}
}

func RunPrivileged(_ string, args ...string) ([]byte, error) {
	helper, err := helperPath()
	if err != nil {
		return nil, fmt.Errorf("signed helper not found: %w", err)
	}

	passwordMu.Lock()
	pw := sudoPassword
	passwordMu.Unlock()

	if pw == "" {
		return nil, fmt.Errorf("no admin password provided")
	}

	sudoArgs := append([]string{"-S", helper}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	cmd.Stdin = strings.NewReader(pw + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil && strings.Contains(string(out), "Sorry, try again") {
		return out, fmt.Errorf("incorrect password")
	}
	return out, err
}
