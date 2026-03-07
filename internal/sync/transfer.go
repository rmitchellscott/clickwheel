package sync

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
	"clickwheel/internal/navidrome"
)

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
		MediaType:   itunesdb.MediaTypeMusic,
		SourceID:    item.SourceID,
	}

	dev.DB.AddTrack(track)
	return nil
}

func TransferBook(ctx context.Context, abs *audiobookshelf.Client, dev *ipod.Device, item BookItem) error {
	tmpDir, err := os.MkdirTemp("", "clickwheel-book-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	downloadPath := filepath.Join(tmpDir, "book.download")
	downloadFile, err := os.Create(downloadPath)
	if err != nil {
		return err
	}

	if err := abs.DownloadFile(item.SourceID, downloadFile); err != nil {
		downloadFile.Close()
		return err
	}
	downloadFile.Close()

	outputPath := filepath.Join(tmpDir, "book.m4b")
	if err := audiobookshelf.MergeToM4B(ctx, []string{downloadPath}, item.Chapters, outputPath); err != nil {
		return fmt.Errorf("converting %s: %w", item.Title, err)
	}

	destPath := ipod.AllocateFilePath(dev.Info.MountPoint, ".m4b")
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return err
	}

	var bookmarkTime uint32
	progress, err := abs.GetProgress(item.SourceID)
	if err == nil && progress != nil {
		bookmarkTime = uint32(progress.CurrentTime * 1000)
	}

	track := &itunesdb.Track{
		UniqueID:          rand.Uint32(),
		Title:             item.Title,
		Artist:            item.Author,
		Album:             item.Title,
		Path:              ipod.ToiPodPath(dev.Info.MountPoint, destPath),
		Duration:          uint32(item.Duration * 1000),
		Size:              uint32(len(data)),
		MediaType:         itunesdb.MediaTypeAudiobook,
		RememberPosition:  1,
		SkipWhenShuffling: 1,
		BookmarkTime:      bookmarkTime,
		SourceID:          item.SourceID,
	}

	dev.DB.AddTrack(track)
	return nil
}
