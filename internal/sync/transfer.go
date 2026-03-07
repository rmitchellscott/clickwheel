package sync

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
	"clickwheel/internal/navidrome"
)

var audioExts = map[string]bool{
	".mp3": true, ".m4a": true, ".m4b": true, ".mp4": true,
	".ogg": true, ".opus": true, ".flac": true, ".wav": true,
	".aac": true, ".wma": true,
}

func TransferTrack(ctx context.Context, nav *navidrome.Client, dev *ipod.Device, item TrackItem) error {
	tmpDir, err := os.MkdirTemp("", "clickwheel-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "source.mp3")
	srcFile, err := os.Create(srcPath)
	if err != nil {
		return err
	}

	if err := nav.Stream(item.SourceID, srcFile); err != nil {
		srcFile.Close()
		return err
	}
	srcFile.Close()

	destPath := ipod.AllocateFilePath(dev.Info.MountPoint, ".mp3")
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return err
	}

	track := &itunesdb.Track{
		UniqueID:    rand.Uint32(),
		Title:       item.Title,
		Artist:      item.Artist,
		Album:       item.Album,
		Genre:       item.Genre,
		Path:        ipod.ToiPodPath(dev.Info.MountPoint, destPath),
		TrackNumber: uint16(item.Track),
		Year:        uint16(item.Year),
		Duration:    uint32(item.Duration * 1000),
		Size:        uint32(len(data)),
		BitRate:     320,
		SampleRate:  44100,
		FiletypeKey: "mp3",
		MediaType:   itunesdb.MediaTypeMusic,
		SourceID:    item.SourceID,
	}

	dev.DB.AddTrack(track)
	return nil
}

func bookCacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "clickwheel", "books")
	return p, os.MkdirAll(p, 0755)
}

func TransferBook(ctx context.Context, abs *audiobookshelf.Client, dev *ipod.Device, item BookItem, onStep func(string)) error {
	cacheDir, _ := bookCacheDir()
	cachedPath := ""
	if cacheDir != "" {
		cachedPath = filepath.Join(cacheDir, item.SourceID+".m4b")
	}

	var finalPath string
	if cachedPath != "" {
		if _, err := os.Stat(cachedPath); err == nil {
			onStep("Using cached")
			finalPath = cachedPath
		}
	}

	if finalPath == "" {
		tmpDir, err := os.MkdirTemp("", "clickwheel-book-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		zipPath := filepath.Join(tmpDir, "book.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			return err
		}

		if err := abs.DownloadFile(item.SourceID, zipFile); err != nil {
			zipFile.Close()
			return fmt.Errorf("downloading %s: %w", item.Title, err)
		}
		zipFile.Close()

		onStep("Extracting")
		audioFiles, err := extractAudioFromZip(zipPath, tmpDir)
		if err != nil {
			return fmt.Errorf("extracting %s: %w", item.Title, err)
		}
		if len(audioFiles) == 0 {
			return fmt.Errorf("no audio files found in download for %s", item.Title)
		}

		onStep("Transcoding")
		transcodedPath := filepath.Join(tmpDir, "book.m4b")
		if err := audiobookshelf.MergeToM4B(ctx, audioFiles, item.Chapters, transcodedPath); err != nil {
			return fmt.Errorf("converting %s: %w", item.Title, err)
		}
		finalPath = transcodedPath

		if cachedPath != "" {
			data, err := os.ReadFile(finalPath)
			if err == nil {
				_ = os.WriteFile(cachedPath, data, 0644)
			}
		}
	}

	onStep("Copying to iPod")
	destPath := ipod.AllocateFilePath(dev.Info.MountPoint, ".m4b")
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(finalPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return err
	}

	durationMs := uint32(item.Duration * 1000)

	var bookmarkTime uint32
	progress, err := abs.GetProgress(item.SourceID)
	if err == nil && progress != nil {
		bm := uint32(progress.CurrentTime * 1000)
		if bm < durationMs {
			bookmarkTime = bm
		}
	}

	track := &itunesdb.Track{
		UniqueID:          rand.Uint32(),
		Title:             item.Title,
		Artist:            item.Author,
		Album:             item.Title,
		Path:              ipod.ToiPodPath(dev.Info.MountPoint, destPath),
		Duration:          durationMs,
		Size:              uint32(len(data)),
		FiletypeKey:       "m4b",
		BitRate:           64,
		SampleRate:        44100,
		MediaType:         itunesdb.MediaTypeAudiobook,
		RememberPosition:  1,
		SkipWhenShuffling: 1,
		BookmarkTime:      bookmarkTime,
		SourceID:          item.SourceID,
	}

	dev.DB.AddTrack(track)
	return nil
}

func extractAudioFromZip(zipPath, destDir string) ([]string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var audioFiles []string
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Name))
		if !audioExts[ext] {
			continue
		}

		outPath := filepath.Join(destDir, filepath.Base(f.Name))
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		out, err := os.Create(outPath)
		if err != nil {
			rc.Close()
			return nil, err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return nil, err
		}
		audioFiles = append(audioFiles, outPath)
	}

	sort.Strings(audioFiles)
	return audioFiles, nil
}

