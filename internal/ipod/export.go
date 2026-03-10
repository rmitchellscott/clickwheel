package ipod

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"clickwheel/internal/ipod/itunesdb"
)

type ExportProgressFunc func(current int, total int, title string)

type ExportOptions struct {
	Tracks         []*itunesdb.Track
	Playlists      []*itunesdb.Playlist
	MountPoint     string
	DestDir        string
	EmbedArtwork   bool
	ExportPlaylist bool
	OnProgress     ExportProgressFunc
}

type ExportResult struct {
	TrackFiles map[uint32]string // UniqueID → dest file path
}

func ExportTracks(opts ExportOptions) (*ExportResult, error) {
	var artworkMap map[uint64]*itunesdb.TrackArtwork
	if opts.EmbedArtwork {
		artDir := filepath.Join(opts.MountPoint, "iPod_Control", "Artwork")
		am, err := itunesdb.ReadArtworkDB(artDir)
		if err != nil {
			log.Printf("[export] could not read ArtworkDB: %v", err)
		} else {
			artworkMap = am
		}
	}

	result := &ExportResult{TrackFiles: make(map[uint32]string)}

	for i, t := range opts.Tracks {
		if opts.OnProgress != nil {
			opts.OnProgress(i, len(opts.Tracks), t.Title)
		}

		srcPath := FromiPodPath(opts.MountPoint, t.Path)
		ext := filepath.Ext(srcPath)
		artist := sanitizeExportFilename(t.Artist)
		title := sanitizeExportFilename(t.Title)
		destName := fmt.Sprintf("%s - %s%s", artist, title, ext)
		destPath := filepath.Join(opts.DestDir, destName)

		if err := copyFileExport(srcPath, destPath); err != nil {
			return result, fmt.Errorf("copy %s: %w", t.Title, err)
		}
		result.TrackFiles[t.UniqueID] = destPath

		if opts.EmbedArtwork && artworkMap != nil {
			embedArtwork(destPath, t, artworkMap, opts.MountPoint)
		}
	}

	if opts.ExportPlaylist {
		exportPlaylists(opts, result)
	}

	return result, nil
}

func embedArtwork(destPath string, t *itunesdb.Track, artworkMap map[uint64]*itunesdb.TrackArtwork, mountPoint string) {
	ta, ok := artworkMap[t.DBID]
	if !ok || len(ta.Refs) == 0 {
		return
	}

	var best itunesdb.ArtworkRef
	for _, ref := range ta.Refs {
		if ref.Width > best.Width {
			best = ref
		}
	}

	artDir := filepath.Join(mountPoint, "iPod_Control", "Artwork")
	rgb565Data, err := itunesdb.ReadArtworkData(artDir, best)
	if err != nil {
		log.Printf("[export] read artwork for %s: %v", t.Title, err)
		return
	}

	img := itunesdb.DecodeRGB565(rgb565Data, best.Width, best.Height)
	jpegData, err := itunesdb.EncodeJPEG(img)
	if err != nil {
		log.Printf("[export] encode jpeg for %s: %v", t.Title, err)
		return
	}

	ext := strings.ToLower(filepath.Ext(destPath))
	switch ext {
	case ".mp3":
		if err := embedID3v2Artwork(destPath, jpegData); err != nil {
			log.Printf("[export] embed mp3 artwork for %s: %v", t.Title, err)
		}
	case ".m4a", ".m4b", ".aac":
		if err := embedM4AArtwork(destPath, jpegData); err != nil {
			log.Printf("[export] embed m4a artwork for %s: %v", t.Title, err)
		}
	}
}

func embedID3v2Artwork(filePath string, jpegData []byte) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	apicFrame := buildAPICFrame(jpegData)
	var newData []byte

	if len(data) >= 3 && string(data[0:3]) == "ID3" {
		if len(data) < 10 {
			return fmt.Errorf("truncated ID3v2 header")
		}
		oldSize := id3v2Size(data[6:10])
		tagEnd := min(10+oldSize, len(data))
		existingFrames := stripAPICFrames(data[10:tagEnd])
		allFrames := append(existingFrames, apicFrame...)
		newData = buildID3v2Tag(data[3], data[4], data[5], allFrames)
		newData = append(newData, data[tagEnd:]...)
	} else {
		allFrames := apicFrame
		newData = buildID3v2Tag(4, 0, 0, allFrames)
		newData = append(newData, data...)
	}

	return os.WriteFile(filePath, newData, 0644)
}

func buildAPICFrame(jpegData []byte) []byte {
	frameID := []byte("APIC")
	mime := []byte("image/jpeg")
	payload := make([]byte, 0, len(mime)+4+len(jpegData))
	payload = append(payload, 0) // text encoding: ISO-8859-1
	payload = append(payload, mime...)
	payload = append(payload, 0)    // null terminator
	payload = append(payload, 0x03) // picture type: front cover
	payload = append(payload, 0)    // description null terminator
	payload = append(payload, jpegData...)

	frame := make([]byte, 10+len(payload))
	copy(frame[0:4], frameID)
	binary.BigEndian.PutUint32(frame[4:8], uint32(len(payload)))
	frame[8] = 0 // flags
	frame[9] = 0
	copy(frame[10:], payload)
	return frame
}

func buildID3v2Tag(version, revByte, flags byte, frames []byte) []byte {
	totalSize := len(frames)
	header := make([]byte, 10)
	copy(header[0:3], "ID3")
	header[3] = version
	header[4] = revByte
	header[5] = flags
	encodeSyncsafe(header[6:10], totalSize)
	return append(header, frames...)
}

func stripAPICFrames(data []byte) []byte {
	var result []byte
	offset := 0
	for offset+10 <= len(data) {
		frameID := string(data[offset : offset+4])
		if frameID[0] == 0 {
			break
		}
		frameSize := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
		frameTotal := 10 + frameSize
		if offset+frameTotal > len(data) {
			break
		}
		if frameID != "APIC" {
			result = append(result, data[offset:offset+frameTotal]...)
		}
		offset += frameTotal
	}
	return result
}

func id3v2Size(b []byte) int {
	return int(b[0])<<21 | int(b[1])<<14 | int(b[2])<<7 | int(b[3])
}

func encodeSyncsafe(b []byte, size int) {
	b[0] = byte((size >> 21) & 0x7F)
	b[1] = byte((size >> 14) & 0x7F)
	b[2] = byte((size >> 7) & 0x7F)
	b[3] = byte(size & 0x7F)
}

func embedM4AArtwork(filePath string, jpegData []byte) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	covrAtom := buildCovrAtom(jpegData)
	newData := injectCovrAtom(data, covrAtom)
	if newData == nil {
		return fmt.Errorf("could not find moov/udta/meta/ilst in M4A")
	}

	return os.WriteFile(filePath, newData, 0644)
}

func buildCovrAtom(jpegData []byte) []byte {
	dataAtomSize := 16 + len(jpegData)
	dataAtom := make([]byte, dataAtomSize)
	binary.BigEndian.PutUint32(dataAtom[0:4], uint32(dataAtomSize))
	copy(dataAtom[4:8], "data")
	binary.BigEndian.PutUint32(dataAtom[8:12], 13) // type: JPEG
	copy(dataAtom[16:], jpegData)

	covrSize := 8 + len(dataAtom)
	covr := make([]byte, 8, covrSize)
	binary.BigEndian.PutUint32(covr[0:4], uint32(covrSize))
	copy(covr[4:8], "covr")
	covr = append(covr, dataAtom...)
	return covr
}

func injectCovrAtom(data []byte, covr []byte) []byte {
	ilstOffset, ilstSize := findAtomPath(data, 0, len(data), []string{"moov", "udta", "meta", "ilst"})
	if ilstOffset < 0 {
		moovOff, moovSize := findAtom(data, 0, len(data), "moov")
		if moovOff < 0 {
			return nil
		}
		udtaOff, udtaSize := findAtom(data, moovOff+8, moovOff+moovSize, "udta")
		if udtaOff < 0 {
			udta := wrapAtom("udta", wrapAtom("meta", append(make([]byte, 4), wrapAtom("ilst", covr)...)))
			return insertAndGrow(data, moovOff, moovSize, moovOff+moovSize, udta)
		}
		metaOff, metaSize := findAtom(data, udtaOff+8, udtaOff+udtaSize, "meta")
		if metaOff < 0 {
			meta := wrapAtom("meta", append(make([]byte, 4), wrapAtom("ilst", covr)...))
			return insertAndGrow(data, udtaOff, udtaSize, udtaOff+udtaSize, meta)
		}
		metaHeaderSize := 12
		ilst := wrapAtom("ilst", covr)
		return insertAndGrow(data, metaOff, metaSize, metaOff+metaHeaderSize, ilst)
	}

	existingCovr, covrSize := findAtom(data, ilstOffset+8, ilstOffset+ilstSize, "covr")
	if existingCovr >= 0 {
		result := make([]byte, 0, len(data)-covrSize+len(covr))
		result = append(result, data[:existingCovr]...)
		result = append(result, covr...)
		result = append(result, data[existingCovr+covrSize:]...)
		diff := len(covr) - covrSize
		growParents(result, 0, len(result), []string{"moov", "udta", "meta", "ilst"}, diff)
		return result
	}

	insertPos := ilstOffset + ilstSize
	result := make([]byte, 0, len(data)+len(covr))
	result = append(result, data[:insertPos]...)
	result = append(result, covr...)
	result = append(result, data[insertPos:]...)
	growParents(result, 0, len(result), []string{"moov", "udta", "meta", "ilst"}, len(covr))
	return result
}

func findAtomPath(data []byte, start, end int, path []string) (int, int) {
	searchStart := start
	searchEnd := end
	var offset, size int
	for i, name := range path {
		if i > 0 {
			searchStart = offset + 8
			if name == "ilst" || path[i-1] == "meta" {
				searchStart = offset + 12
			}
			searchEnd = offset + size
		}
		off, sz := findAtom(data, searchStart, searchEnd, name)
		if off < 0 {
			return -1, 0
		}
		offset = off
		size = sz
	}
	return offset, size
}

func findAtom(data []byte, start, end int, name string) (int, int) {
	offset := start
	for offset+8 <= end {
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		atomName := string(data[offset+4 : offset+8])
		if size < 8 {
			break
		}
		if offset+size > end {
			break
		}
		if atomName == name {
			return offset, size
		}
		offset += size
	}
	return -1, 0
}

func wrapAtom(name string, content []byte) []byte {
	size := 8 + len(content)
	atom := make([]byte, 8, size)
	binary.BigEndian.PutUint32(atom[0:4], uint32(size))
	copy(atom[4:8], name)
	atom = append(atom, content...)
	return atom
}

func insertAndGrow(data []byte, parentOff, parentSize, insertPos int, child []byte) []byte {
	result := make([]byte, 0, len(data)+len(child))
	result = append(result, data[:insertPos]...)
	result = append(result, child...)
	result = append(result, data[insertPos:]...)
	newParentSize := parentSize + len(child)
	binary.BigEndian.PutUint32(result[parentOff:parentOff+4], uint32(newParentSize))
	growAncestors(result, 0, len(result), parentOff, len(child))
	return result
}

func growAncestors(data []byte, start, end, targetOff, diff int) {
	offset := start
	for offset+8 <= end {
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		if size < 8 || offset+size > end {
			break
		}
		if targetOff >= offset && targetOff < offset+size && offset != targetOff {
			newSize := size + diff
			binary.BigEndian.PutUint32(data[offset:offset+4], uint32(newSize))
			growAncestors(data, offset+8, offset+newSize, targetOff, 0)
			return
		}
		offset += size
	}
}

func growParents(data []byte, start, end int, path []string, diff int) {
	if diff == 0 || len(path) == 0 {
		return
	}
	offset := start
	for offset+8 <= end {
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		name := string(data[offset+4 : offset+8])
		if size < 8 || offset+size > end {
			break
		}
		if name == path[0] {
			binary.BigEndian.PutUint32(data[offset:offset+4], uint32(size+diff))
			childStart := offset + 8
			if name == "meta" {
				childStart = offset + 12
			}
			growParents(data, childStart, offset+size+diff, path[1:], diff)
			return
		}
		offset += size
	}
}

func exportPlaylists(opts ExportOptions, result *ExportResult) {
	trackIDSet := make(map[uint32]bool)
	for _, t := range opts.Tracks {
		trackIDSet[t.UniqueID] = true
	}

	for _, pl := range opts.Playlists {
		if pl.IsMaster {
			continue
		}

		var entries []string
		for _, t := range pl.Tracks {
			if destPath, ok := result.TrackFiles[t.UniqueID]; ok {
				entries = append(entries, filepath.Base(destPath))
			}
		}
		if len(entries) == 0 {
			continue
		}

		plName := sanitizeExportFilename(pl.Name)
		plPath := filepath.Join(opts.DestDir, plName+".m3u")

		var sb strings.Builder
		sb.WriteString("#EXTM3U\n")
		for _, entry := range entries {
			sb.WriteString(entry)
			sb.WriteString("\n")
		}
		if err := os.WriteFile(plPath, []byte(sb.String()), 0644); err != nil {
			log.Printf("[export] write playlist %s: %v", pl.Name, err)
		}
	}
}

func sanitizeExportFilename(s string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(s)
}

func copyFileExport(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
