package ipod

import (
	"fmt"
	"os/exec"
)

func Eject(mountPoint string) error {
	out, err := exec.Command("umount", mountPoint).CombinedOutput()
	if err != nil {
		return fmt.Errorf("eject failed: %s", string(out))
	}
	return nil
}
