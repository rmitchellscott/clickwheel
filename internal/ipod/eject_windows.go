package ipod

import (
	"fmt"
	"os/exec"
)

func Eject(mountPoint string) error {
	drive := mountPoint[:2]
	out, err := exec.Command("mountvol", drive, "/P").CombinedOutput()
	if err != nil {
		return fmt.Errorf("eject failed: %s", string(out))
	}
	return nil
}
