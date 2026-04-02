package audiobookshelf

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

var nativeFFmpeg string

func init() {
	nativeFFmpeg, _ = exec.LookPath("ffmpeg")
	if nativeFFmpeg != "" {
		log.Printf("[ffmpeg] using native: %s", nativeFFmpeg)
	} else {
		log.Printf("[ffmpeg] native not found, using WASM (slower)")
	}
}

func MergeToM4B(ctx context.Context, inputFiles []string, chapters []Chapter, outputPath string) error {
	if len(inputFiles) == 0 {
		return fmt.Errorf("no input files provided")
	}

	if len(inputFiles) == 1 {
		return transcodeSingle(ctx, inputFiles[0], outputPath)
	}

	dir := filepath.Dir(outputPath)
	listFile := filepath.Join(dir, "concat_list.txt")

	var b strings.Builder
	for _, f := range inputFiles {
		fmt.Fprintf(&b, "file '%s'\n", f)
	}
	if err := os.WriteFile(listFile, []byte(b.String()), 0644); err != nil {
		return err
	}
	defer os.Remove(listFile)

	chapterFile := ""
	if len(chapters) > 0 {
		chapterFile = filepath.Join(dir, "chapters.txt")
		if err := writeFFMetadata(chapters, chapterFile); err != nil {
			return err
		}
		defer os.Remove(chapterFile)
	}

	args := []string{
		"-f", "concat", "-safe", "0", "-i", listFile,
	}
	if chapterFile != "" {
		args = append(args, "-i", chapterFile, "-map_metadata", "1")
	}
	args = append(args,
		"-vn",
		"-c:a", "aac", "-b:a", "64k", "-ar", "22050", "-profile:a", "aac_low", "-aac_pns", "0", "-ac", "1",
		"-movflags", "+faststart",
		"-y", outputPath,
	)

	return RunFFmpeg(ctx, args, dir)
}

func transcodeSingle(ctx context.Context, input, output string) error {
	dir := filepath.Dir(input)
	args := []string{
		"-i", input,
		"-vn",
		"-c:a", "aac", "-b:a", "64k", "-ar", "22050", "-profile:a", "aac_low", "-aac_pns", "0", "-ac", "1",
		"-movflags", "+faststart",
		"-y", output,
	}
	return RunFFmpeg(ctx, args, dir)
}

func RunFFmpeg(ctx context.Context, args []string, workDir string) error {
	if nativeFFmpeg != "" {
		return runNativeFFmpeg(ctx, args)
	}
	return runWasmFFmpeg(ctx, args, workDir)
}

func runNativeFFmpeg(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, nativeFFmpeg, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}
	return nil
}

func runWasmFFmpeg(ctx context.Context, args []string, workDir string) error {
	rc, err := ffmpreg.Ffmpeg(ctx, wasm.Args{
		Stderr: os.Stderr,
		Args:   args,
		Config: mountDir(workDir),
	})
	if err != nil {
		return err
	}
	if rc != 0 {
		return fmt.Errorf("ffmpeg (wasm) exited with code %d", rc)
	}
	return nil
}

func mountDir(dir string) func(wazero.ModuleConfig) wazero.ModuleConfig {
	return func(cfg wazero.ModuleConfig) wazero.ModuleConfig {
		return cfg.WithFSConfig(wazero.NewFSConfig().WithDirMount(dir, dir))
	}
}

func writeFFMetadata(chapters []Chapter, path string) error {
	var b strings.Builder
	b.WriteString(";FFMETADATA1\n")
	for _, ch := range chapters {
		fmt.Fprintf(&b, "\n[CHAPTER]\nTIMEBASE=1/1000\nSTART=%d\nEND=%d\ntitle=%s\n",
			int64(ch.Start*1000), int64(ch.End*1000), ch.Title)
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}
