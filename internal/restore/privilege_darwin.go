package restore

import (
	"fmt"
	"os/exec"
	"strings"
)

func CheckPrivilege() PrivilegeResult {
	cmd := exec.Command("osascript", "-e",
		`do shell script "echo ok" with administrator privileges`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return PrivilegeResult{Granted: false, Error: fmt.Errorf("privilege denied: %s", string(out))}
	}
	return PrivilegeResult{Granted: true}
}

func RunPrivileged(command string, args ...string) ([]byte, error) {
	parts := []string{shellQuote(command)}
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	script := strings.Join(parts, " ")
	escaped := strings.ReplaceAll(script, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	cmd := exec.Command("osascript", "-e",
		fmt.Sprintf(`do shell script "%s" with administrator privileges`, escaped))
	return cmd.CombinedOutput()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
