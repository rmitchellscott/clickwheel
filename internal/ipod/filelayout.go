package ipod

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultMusicDirCount = 20
	filenameLen          = 4
)

func EnsureMusicDirs(mountPoint string, dirCount int) error {
	if dirCount <= 0 {
		dirCount = defaultMusicDirCount
	}
	musicDir := filepath.Join(mountPoint, "iPod_Control", "Music")
	for i := 0; i < dirCount; i++ {
		dir := filepath.Join(musicDir, subDirName(i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}

func AllocateFilePath(mountPoint, ext string, dirCount int) string {
	if dirCount <= 0 {
		dirCount = defaultMusicDirCount
	}
	dirIndex := rand.Intn(dirCount)
	name := randomFilename() + ext
	return filepath.Join(mountPoint, "iPod_Control", "Music", subDirName(dirIndex), name)
}

func ToiPodPath(mountPoint, absPath string) string {
	rel, _ := filepath.Rel(mountPoint, absPath)
	return ":" + strings.ReplaceAll(rel, string(filepath.Separator), ":")
}

func FromiPodPath(mountPoint, iPodPath string) string {
	rel := strings.ReplaceAll(strings.TrimPrefix(iPodPath, ":"), ":", string(filepath.Separator))
	return filepath.Join(mountPoint, rel)
}

func CleanOrphans(mountPoint string, knownPaths map[string]bool) (int, error) {
	musicDir := filepath.Join(mountPoint, "iPod_Control", "Music")
	removed := 0
	entries, err := os.ReadDir(musicDir)
	if err != nil {
		return 0, err
	}
	for _, dir := range entries {
		if !dir.IsDir() {
			continue
		}
		subDir := filepath.Join(musicDir, dir.Name())
		files, err := os.ReadDir(subDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			path := filepath.Join(subDir, f.Name())
			iPodPath := ToiPodPath(mountPoint, path)
			if !knownPaths[iPodPath] {
				os.Remove(path)
				removed++
			}
		}
	}
	return removed, nil
}

func subDirName(index int) string {
	return fmt.Sprintf("F%02d", index)
}

func randomFilename() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, filenameLen)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
