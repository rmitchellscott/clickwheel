package restore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const askpassScript = `#!/bin/bash
osascript -e 'display dialog "clickwheel needs administrator privileges to restore your iPod." with title "clickwheel" default answer "" with hidden answer buttons {"Cancel","OK"} default button "OK"' -e 'text returned of result' 2>/dev/null
`

func CheckPrivilege() PrivilegeResult {
	askpass, err := writeAskpass()
	if err != nil {
		return PrivilegeResult{Granted: false, Error: err}
	}
	defer os.Remove(askpass)

	cmd := exec.Command("sudo", "--askpass", "echo", "ok")
	cmd.Env = append(os.Environ(), "SUDO_ASKPASS="+askpass)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return PrivilegeResult{Granted: false, Error: fmt.Errorf("privilege denied: %s", string(out))}
	}
	return PrivilegeResult{Granted: true}
}

func RunPrivileged(_ string, args ...string) ([]byte, error) {
	helper, err := helperPath()
	if err != nil {
		return nil, fmt.Errorf("signed helper not found: %w", err)
	}

	askpass, err := writeAskpass()
	if err != nil {
		return nil, err
	}
	defer os.Remove(askpass)

	sudoArgs := append([]string{"--askpass", helper}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	cmd.Env = append(os.Environ(), "SUDO_ASKPASS="+askpass)
	return cmd.CombinedOutput()
}

func writeAskpass() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("cache dir: %w", err)
	}
	p := filepath.Join(dir, "clickwheel", "askpass.sh")
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, []byte(askpassScript), 0700); err != nil {
		return "", err
	}
	return p, nil
}
