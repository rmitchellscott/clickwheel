package transcode

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
	"github.com/tetratelabs/wazero"
)

var compatibleFormats = map[string]bool{
	".mp3":  true,
	".m4a":  true,
	".aac":  true,
	".wav":  true,
	".aiff": true,
	".aif":  true,
	".m4b":  true,
}

var nativeFFmpeg string

func Init() {
	nativeFFmpeg, _ = exec.LookPath("ffmpeg")
	if nativeFFmpeg != "" {
		log.Printf("[transcode] using native ffmpeg: %s", nativeFFmpeg)
	} else {
		log.Printf("[transcode] native ffmpeg not found, using WASM")
		ffmpreg.Initialize()
	}
}

func NeedsTranscode(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return !compatibleFormats[ext]
}

func OutputExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if compatibleFormats[ext] {
		return ext
	}
	return ".mp3"
}

func runFFmpeg(ctx context.Context, args []string) error {
	if nativeFFmpeg != "" {
		return runNative(ctx, args)
	}
	return runWasm(ctx, args)
}

func runNative(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, nativeFFmpeg, args...)
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

func ToMP3(ctx context.Context, input, output string) error {
	return runFFmpeg(ctx, []string{
		"-i", input,
		"-c:a", "libmp3lame",
		"-b:a", "320k",
		"-y", output,
	})
}

func ToAAC(ctx context.Context, input, output string) error {
	return runFFmpeg(ctx, []string{
		"-i", input,
		"-vn",
		"-c:a", "aac",
		"-b:a", "256k",
		"-ar", "44100",
		"-ac", "2",
		"-aac_pns", "0",
		"-profile:a", "aac_low",
		"-movflags", "+faststart",
		"-f", "ipod",
		"-y", output,
	})
}

func RemuxToM4A(ctx context.Context, input, output string) error {
	return runFFmpeg(ctx, []string{
		"-i", input,
		"-c:a", "copy",
		"-movflags", "+faststart",
		"-f", "ipod",
		"-y", output,
	})
}

func Transcode(ctx context.Context, input, output, format string, bitRate int) error {
	br := fmt.Sprintf("%dk", bitRate)
	switch format {
	case "aac":
		return runFFmpeg(ctx, []string{
			"-i", input,
			"-vn",
			"-c:a", "aac",
			"-b:a", br,
			"-ar", "44100",
			"-ac", "2",
			"-aac_pns", "0",
			"-profile:a", "aac_low",
			"-movflags", "+faststart",
			"-f", "ipod",
			"-y", output,
		})
	case "mp3":
		return runFFmpeg(ctx, []string{
			"-i", input,
			"-c:a", "libmp3lame",
			"-b:a", br,
			"-ar", "44100",
			"-ac", "2",
			"-y", output,
		})
	case "alac":
		return runFFmpeg(ctx, []string{
			"-i", input,
			"-vn",
			"-c:a", "alac",
			"-ar", "44100",
			"-ac", "2",
			"-movflags", "+faststart",
			"-f", "ipod",
			"-y", output,
		})
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}
