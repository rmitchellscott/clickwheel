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
	"clickwheel/internal/subsonic"
	"clickwheel/internal/transcode"
)

var audioExts = map[string]bool{
	".mp3": true, ".m4a": true, ".m4b": true, ".mp4": true,
	".ogg": true, ".opus": true, ".flac": true, ".wav": true,
	".aac": true, ".wma": true,
}

func formatExt(format string) string {
	switch format {
	case "aac":
		return ".m4a"
	case "opus":
		return ".opus"
	case "raw":
		return ".raw"
	default:
		return ".mp3"
	}
}

func formatFileType(format string) string {
	switch format {
	case "aac":
		return "m4a"
	default:
		return format
	}
}

type preparedTrack struct {
	tmpDir   string
	dataPath string
	data     []byte
	item     TrackItem
	err      error
}

func (p *preparedTrack) cleanup() {
	if p.tmpDir != "" {
		os.RemoveAll(p.tmpDir)
	}
}

func DownloadAndTranscode(ctx context.Context, sub *subsonic.Client, item TrackItem, format string, bitRate int) *preparedTrack {
	if format == "" {
		format = "aac"
	}
	if bitRate <= 0 {
		bitRate = 256
	}
	ext := formatExt(format)
	needsXcode := item.Suffix != format && !(item.Suffix == "m4a" && format == "aac")

	tmpDir, err := os.MkdirTemp("", "clickwheel-")
	if err != nil {
		return &preparedTrack{item: item, err: err}
	}

	var srcPath string
	if needsXcode {
		srcPath = filepath.Join(tmpDir, "source."+item.Suffix)
	} else {
		srcPath = filepath.Join(tmpDir, "source"+ext)
	}

	srcFile, err := os.Create(srcPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return &preparedTrack{item: item, err: err}
	}
	if err := sub.Download(item.SourceID, srcFile); err != nil {
		srcFile.Close()
		os.RemoveAll(tmpDir)
		return &preparedTrack{item: item, err: err}
	}
	srcFile.Close()

	var finalPath string
	if needsXcode {
		finalPath = filepath.Join(tmpDir, "output"+ext)
		if err := transcode.Transcode(ctx, srcPath, finalPath, format, bitRate); err != nil {
			os.RemoveAll(tmpDir)
			return &preparedTrack{item: item, err: fmt.Errorf("transcoding: %w", err)}
		}
	} else {
		finalPath = srcPath
	}

	data, err := os.ReadFile(finalPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return &preparedTrack{item: item, err: err}
	}
	os.RemoveAll(tmpDir)

	return &preparedTrack{item: item, data: data}
}

func InstallTrack(dev *ipod.Device, p *preparedTrack, format string, bitRate int) error {
	if p.err != nil {
		return p.err
	}
	defer p.cleanup()

	if format == "" {
		format = "aac"
	}
	if bitRate <= 0 {
		bitRate = 256
	}
	ext := formatExt(format)

	destPath := ipod.AllocateFilePath(dev.Info.MountPoint, ext)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(destPath, p.data, 0644); err != nil {
		return err
	}

	item := p.item
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
		Size:        uint32(len(p.data)),
		BitRate:     uint32(bitRate),
		SampleRate:  44100,
		FiletypeKey: formatFileType(format),
		MediaType:   itunesdb.MediaTypeMusic,
		SourceID:    item.SourceID,
	}

	dev.DB.AddTrack(track)
	return nil
}

func TransferTrack(ctx context.Context, sub *subsonic.Client, dev *ipod.Device, item TrackItem, format string, bitRate int) error {
	p := DownloadAndTranscode(ctx, sub, item, format, bitRate)
	return InstallTrack(dev, p, format, bitRate)
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
	m4bPath, err := ensureBookM4B(ctx, abs, item, onStep)
	if err != nil {
		return err
	}

	if item.SplitParts != nil {
		return transferSplitBook(ctx, abs, dev, item, m4bPath, onStep)
	}
	return transferWholeBook(ctx, abs, dev, item, m4bPath, onStep)
}

func ensureBookM4B(ctx context.Context, abs *audiobookshelf.Client, item BookItem, onStep func(string)) (string, error) {
	cacheDir, _ := bookCacheDir()
	cachedPath := ""
	if cacheDir != "" {
		cachedPath = filepath.Join(cacheDir, item.SourceID+".m4b")
	}

	if cachedPath != "" {
		if _, err := os.Stat(cachedPath); err == nil {
			onStep("Using cached")
			return cachedPath, nil
		}
	}

	tmpDir, err := os.MkdirTemp("", "clickwheel-book-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, "book.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}

	if err := abs.DownloadFile(item.SourceID, zipFile); err != nil {
		zipFile.Close()
		return "", fmt.Errorf("downloading %s: %w", item.Title, err)
	}
	zipFile.Close()

	onStep("Extracting")
	audioFiles, err := extractAudioFromZip(zipPath, tmpDir)
	if err != nil {
		return "", fmt.Errorf("extracting %s: %w", item.Title, err)
	}
	if len(audioFiles) == 0 {
		return "", fmt.Errorf("no audio files found in download for %s", item.Title)
	}

	onStep("Transcoding")
	transcodedPath := filepath.Join(tmpDir, "book.m4b")
	if err := audiobookshelf.MergeToM4B(ctx, audioFiles, item.Chapters, transcodedPath); err != nil {
		return "", fmt.Errorf("converting %s: %w", item.Title, err)
	}

	if cachedPath != "" {
		data, err := os.ReadFile(transcodedPath)
		if err == nil {
			_ = os.WriteFile(cachedPath, data, 0644)
		}
		return cachedPath, nil
	}

	return transcodedPath, nil
}

func transferWholeBook(ctx context.Context, abs *audiobookshelf.Client, dev *ipod.Device, item BookItem, m4bPath string, onStep func(string)) error {
	onStep("Copying to iPod")
	destPath := ipod.AllocateFilePath(dev.Info.MountPoint, ".m4b")
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(m4bPath)
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

func splitM4BPart(ctx context.Context, sourcePath string, startSec, endSec float64, workDir string, index int) (string, error) {
	outPath := filepath.Join(workDir, fmt.Sprintf("part_%d.m4b", index))
	args := []string{
		"-i", sourcePath,
		"-ss", fmt.Sprintf("%.3f", startSec),
		"-to", fmt.Sprintf("%.3f", endSec),
		"-c:a", "copy",
		"-vn",
		"-movflags", "+faststart",
		"-y", outPath,
	}
	if err := audiobookshelf.RunFFmpeg(ctx, args, workDir); err != nil {
		return "", fmt.Errorf("splitting part %d: %w", index, err)
	}
	return outPath, nil
}

func transferSplitBook(ctx context.Context, abs *audiobookshelf.Client, dev *ipod.Device, item BookItem, m4bPath string, onStep func(string)) error {
	workDir, err := os.MkdirTemp("", "clickwheel-split-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	var globalBookmarkSec float64
	progress, err := abs.GetProgress(item.SourceID)
	if err == nil && progress != nil {
		globalBookmarkSec = progress.CurrentTime
	}

	totalParts := len(item.SplitParts)
	for _, part := range item.SplitParts {
		onStep(fmt.Sprintf("Splitting part %d of %d", part.Index+1, totalParts))

		partPath, err := splitM4BPart(ctx, m4bPath, part.StartSec, part.EndSec, workDir, part.Index)
		if err != nil {
			return err
		}

		data, err := os.ReadFile(partPath)
		if err != nil {
			return err
		}

		destPath := ipod.AllocateFilePath(dev.Info.MountPoint, ".m4b")
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return err
		}

		partDurationMs := uint32((part.EndSec - part.StartSec) * 1000)
		var bookmarkTime uint32
		if globalBookmarkSec >= part.StartSec && globalBookmarkSec < part.EndSec {
			bookmarkTime = uint32((globalBookmarkSec - part.StartSec) * 1000)
		}

		track := &itunesdb.Track{
			UniqueID:          rand.Uint32(),
			Title:             fmt.Sprintf("%s - Part %d of %d", item.Title, part.Index+1, totalParts),
			Artist:            item.Author,
			Album:             item.Title,
			Path:              ipod.ToiPodPath(dev.Info.MountPoint, destPath),
			Duration:          partDurationMs,
			Size:              uint32(len(data)),
			FiletypeKey:       "m4b",
			BitRate:           64,
			SampleRate:        44100,
			MediaType:         itunesdb.MediaTypeAudiobook,
			RememberPosition:  1,
			SkipWhenShuffling: 1,
			BookmarkTime:      bookmarkTime,
			SourceID:          bookPartSourceID(item.SourceID, part.Index),
		}

		dev.DB.AddTrack(track)
	}

	return nil
}

func TransferPodcastEpisode(ctx context.Context, abs *audiobookshelf.Client, dev *ipod.Device, item PodcastEpisodeItem, onStep func(string)) error {
	tmpDir, err := os.MkdirTemp("", "clickwheel-podcast-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	ext := ".mp3"
	if item.Ext != "" {
		ext = strings.ToLower(item.Ext)
	}

	srcPath := filepath.Join(tmpDir, "episode"+ext)
	srcFile, err := os.Create(srcPath)
	if err != nil {
		return err
	}

	if err := abs.DownloadEpisodeFile(item.ItemID, item.Ino, srcFile); err != nil {
		srcFile.Close()
		return fmt.Errorf("downloading %s: %w", item.Title, err)
	}
	srcFile.Close()

	finalPath := srcPath
	needsTranscode := ext != ".mp3" && ext != ".m4a" && ext != ".m4b" && ext != ".aac"
	if needsTranscode {
		onStep("Transcoding")
		finalPath = filepath.Join(tmpDir, "output.mp3")
		if err := transcode.Transcode(ctx, srcPath, finalPath, "mp3", 128); err != nil {
			return fmt.Errorf("transcoding %s: %w", item.Title, err)
		}
		ext = ".mp3"
	}

	onStep("Copying to iPod")
	data, err := os.ReadFile(finalPath)
	if err != nil {
		return err
	}

	destPath := ipod.AllocateFilePath(dev.Info.MountPoint, ext)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return err
	}

	durationMs := uint32(item.Duration * 1000)

	var bookmarkTime uint32
	progress, err := abs.GetEpisodeProgress(item.ItemID, item.EpisodeID)
	if err == nil && progress != nil {
		bm := uint32(progress.CurrentTime * 1000)
		if bm < durationMs {
			bookmarkTime = bm
		}
	}

	fileType := "mp3"
	if ext == ".m4a" || ext == ".aac" {
		fileType = "m4a"
	} else if ext == ".m4b" {
		fileType = "m4b"
	}

	track := &itunesdb.Track{
		UniqueID:          rand.Uint32(),
		Title:             item.Title,
		Artist:            item.Author,
		Album:             item.ShowName,
		ShowName:          item.ShowName,
		Path:              ipod.ToiPodPath(dev.Info.MountPoint, destPath),
		Duration:          durationMs,
		Size:              uint32(len(data)),
		FiletypeKey:       fileType,
		BitRate:           128,
		SampleRate:        44100,
		MediaType:         itunesdb.MediaTypePodcast,
		PodcastFlag:       1,
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

