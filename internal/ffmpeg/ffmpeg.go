package ffmpeg

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/tetratelabs/wazero"
)

var (
	mu          sync.Mutex
	resolvedBin string
	usingWasm   bool
	initialized bool
)

func Init(configPath string) {
	mu.Lock()
	defer mu.Unlock()

	resolvedBin = ""
	usingWasm = false
	initialized = true

	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			resolvedBin = configPath
			log.Printf("[ffmpeg] using configured path: %s", resolvedBin)
			return
		}
		log.Printf("[ffmpeg] configured path not found: %s", configPath)
	}

	if p, err := exec.LookPath("ffmpeg"); err == nil {
		resolvedBin = p
		log.Printf("[ffmpeg] found on PATH: %s", resolvedBin)
		return
	}

	log.Printf("[ffmpeg] not found, will use WASM fallback")
	usingWasm = true
	ffmpreg.Initialize()
}

type Info struct {
	Path    string `json:"path"`
	Source  string `json:"source"`
	Version string `json:"version"`
}

func GetInfo() Info {
	mu.Lock()
	defer mu.Unlock()

	if !initialized {
		return Info{Source: "not initialized"}
	}
	if usingWasm {
		return Info{Source: "wasm"}
	}
	info := Info{Path: resolvedBin, Source: "native"}
	if v, err := probeVersion(resolvedBin); err == nil {
		info.Version = v
	}
	return info
}

func probeVersion(bin string) (string, error) {
	out, err := exec.Command(bin, "-version").Output()
	if err != nil {
		return "", err
	}
	line := string(out)
	for i, c := range line {
		if c == '\n' {
			line = line[:i]
			break
		}
	}
	return line, nil
}

func Run(ctx context.Context, args []string) error {
	mu.Lock()
	native := resolvedBin
	wasm := usingWasm
	mu.Unlock()

	if native != "" {
		return runNative(ctx, native, args)
	}
	if wasm {
		return runWasm(ctx, args)
	}
	return fmt.Errorf("ffmpeg not initialized — call ffmpeg.Init first")
}

func runNative(ctx context.Context, bin string, args []string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}
	return nil
}

func runWasm(ctx context.Context, args []string) error {
	var dirs []string
	for i, a := range args {
		if i > 0 && (args[i-1] == "-i" || args[i-1] == "-y") {
			if d := filepath.Dir(a); d != "" && d != "." {
				dirs = append(dirs, d)
			}
		}
		if i == len(args)-1 {
			if d := filepath.Dir(a); d != "" && d != "." {
				dirs = append(dirs, d)
			}
		}
	}

	rc, err := ffmpreg.Ffmpeg(ctx, wasm.Args{
		Stderr: os.Stderr,
		Args:   args,
		Config: mountDirs(dirs),
	})
	if err != nil {
		return err
	}
	if rc != 0 {
		return fmt.Errorf("ffmpeg (wasm) exited with code %d", rc)
	}
	return nil
}

func mountDirs(dirs []string) func(wazero.ModuleConfig) wazero.ModuleConfig {
	return func(cfg wazero.ModuleConfig) wazero.ModuleConfig {
		fs := wazero.NewFSConfig()
		seen := make(map[string]bool)
		for _, d := range dirs {
			if !seen[d] {
				fs = fs.WithDirMount(d, d)
				seen[d] = true
			}
		}
		return cfg.WithFSConfig(fs)
	}
}
