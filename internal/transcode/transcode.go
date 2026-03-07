package transcode

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/gruf/go-ffmpreg/ffmpreg"
	"codeberg.org/gruf/go-ffmpreg/wasm"
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

func Init() {
	ffmpreg.Initialize()
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
	rc, err := ffmpreg.Ffmpeg(ctx, wasm.Args{
		Stderr: os.Stderr,
		Args:   args,
	})
	if err != nil {
		return err
	}
	if rc != 0 {
		return fmt.Errorf("ffmpeg exited with code %d", rc)
	}
	return nil
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
		"-c:a", "aac",
		"-b:a", "256k",
		"-movflags", "+faststart",
		"-y", output,
	})
}
