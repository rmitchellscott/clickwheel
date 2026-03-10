package restore

import (
	"fmt"
	"os/exec"
)

func CheckPrivilege() PrivilegeResult {
	cmd := exec.Command("pkexec", "--help")
	if err := cmd.Run(); err != nil {
		cmd2 := exec.Command("sudo", "-n", "true")
		if err := cmd2.Run(); err != nil {
			return PrivilegeResult{Granted: false, Error: fmt.Errorf("no privilege escalation available")}
		}
	}
	return PrivilegeResult{Granted: true}
}

func SetPassword(_ string) {}
func ClearPassword()       {}

func RunPrivileged(command string, args ...string) ([]byte, error) {
	allArgs := append([]string{command}, args...)
	cmd := exec.Command("pkexec", allArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		allArgs = append([]string{command}, args...)
		cmd = exec.Command("sudo", allArgs...)
		return cmd.CombinedOutput()
	}
	return out, nil
}
