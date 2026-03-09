package restore

import (
	"fmt"
	"os"
	"path/filepath"
)

func helperPath() (string, error) {
	if p := os.Getenv("CLICKWHEEL_HELPER"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "clickwheel-helper")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	p := filepath.Join("build", "bin", "clickwheel-helper")
	if _, err := os.Stat(p); err == nil {
		return p, nil
	}

	return "", fmt.Errorf("clickwheel-helper not found; run 'make helper'")
}
