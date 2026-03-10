package itunesdb

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf16"
)

const (
	mhfdHeaderSize = 132
	mhsdArtHeaderSize = 96
	mhliHeaderSize = 92
	mhlaHeaderSize = 92
	mhlfHeaderSize = 92
	mhiiHeaderSize = 152
	mhodArtHeaderSize = 24
	mhniHeaderSize = 76
	mhifHeaderSize = 124
)

type ArtworkImage struct {
	ImageID  uint32
	SongDBID uint64
	Formats  map[int][]byte // formatID → RGB565 pixel data
	SrcSize  int
}

func WriteArtworkDB(artworkDir string, images []*ArtworkImage, formats []ArtworkFormat) error {
	if err := os.MkdirAll(artworkDir, 0755); err != nil {
		return err
	}

	sortedFmts := make([]ArtworkFormat, len(formats))
	copy(sortedFmts, formats)
	sort.Slice(sortedFmts, func(i, j int) bool {
		return sortedFmts[i].FormatID < sortedFmts[j].FormatID
	})

	itmbOffsets := make(map[int]int)
	itmbFiles := make(map[int]*os.File)
	for _, f := range sortedFmts {
		itmbOffsets[f.FormatID] = 0
	}

	for _, f := range sortedFmts {
		tmpPath := filepath.Join(artworkDir, fmt.Sprintf("F%d_1.ithmb.tmp", f.FormatID))
		file, err := os.Create(tmpPath)
		if err != nil {
			for _, of := range itmbFiles {
				of.Close()
			}
			return err
		}
		itmbFiles[f.FormatID] = file
	}
	defer func() {
		for _, f := range itmbFiles {
			f.Close()
		}
	}()

	imageOffsets := make(map[uint32]map[int]int) // imageID → formatID → offset
	for _, img := range images {
		offsets := make(map[int]int)
		for _, f := range sortedFmts {
			data, ok := img.Formats[f.FormatID]
			if !ok {
				continue
			}
			offsets[f.FormatID] = itmbOffsets[f.FormatID]
			if _, err := itmbFiles[f.FormatID].Write(data); err != nil {
				return err
			}
			itmbOffsets[f.FormatID] += len(data)
		}
		imageOffsets[img.ImageID] = offsets
	}

	for _, f := range itmbFiles {
		f.Sync()
	}

	imageSizes := make(map[int]int)
	for _, f := range sortedFmts {
		imageSizes[f.FormatID] = f.Width * f.Height * 2
	}

	ds1 := writeArtMHSD(1, writeArtMHLI(images, imageOffsets, sortedFmts))
	ds2 := writeArtMHSD(2, writeArtMHLA())
	ds3 := writeArtMHSD(3, writeArtMHLF(sortedFmts, imageSizes))

	var body []byte
	body = append(body, ds1...)
	body = append(body, ds2...)
	body = append(body, ds3...)

	nextID := uint32(100)
	if len(images) > 0 {
		nextID = images[len(images)-1].ImageID + 1
	}
	header := writeArtMHFD(3, nextID, len(body))
	artDB := append(header, body...)

	tmpPath := filepath.Join(artworkDir, "ArtworkDB.tmp")
	if err := os.WriteFile(tmpPath, artDB, 0644); err != nil {
		return err
	}

	for _, f := range sortedFmts {
		src := filepath.Join(artworkDir, fmt.Sprintf("F%d_1.ithmb.tmp", f.FormatID))
		dst := filepath.Join(artworkDir, fmt.Sprintf("F%d_1.ithmb", f.FormatID))
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return os.Rename(tmpPath, filepath.Join(artworkDir, "ArtworkDB"))
}

func writeArtMHFD(childCount int, nextImageID uint32, bodyLen int) []byte {
	totalLen := mhfdHeaderSize + bodyLen
	buf := make([]byte, mhfdHeaderSize)
	copy(buf[0:4], "mhfd")
	binary.LittleEndian.PutUint32(buf[4:8], mhfdHeaderSize)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(totalLen))
	binary.LittleEndian.PutUint32(buf[16:20], 2)
	binary.LittleEndian.PutUint32(buf[20:24], uint32(childCount))
	binary.LittleEndian.PutUint32(buf[28:32], nextImageID)
	binary.LittleEndian.PutUint32(buf[48:52], 2)
	return buf
}

func writeArtMHSD(dsType int, child []byte) []byte {
	totalLen := mhsdArtHeaderSize + len(child)
	buf := make([]byte, mhsdArtHeaderSize)
	copy(buf[0:4], "mhsd")
	binary.LittleEndian.PutUint32(buf[4:8], mhsdArtHeaderSize)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(totalLen))
	binary.LittleEndian.PutUint16(buf[12:14], uint16(dsType))
	return append(buf, child...)
}

func writeArtMHLI(images []*ArtworkImage, offsets map[uint32]map[int]int, formats []ArtworkFormat) []byte {
	var children []byte
	for _, img := range images {
		children = append(children, writeArtMHII(img, offsets[img.ImageID], formats)...)
	}
	buf := make([]byte, mhliHeaderSize)
	copy(buf[0:4], "mhli")
	binary.LittleEndian.PutUint32(buf[4:8], mhliHeaderSize)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(len(images)))
	return append(buf, children...)
}

func writeArtMHII(img *ArtworkImage, offsets map[int]int, formats []ArtworkFormat) []byte {
	var children []byte
	childCount := 0
	for _, f := range formats {
		offset, ok := offsets[f.FormatID]
		if !ok {
			continue
		}
		dataSize := f.Width * f.Height * 2
		mhni := writeArtMHNI(f.FormatID, offset, dataSize, f.Width, f.Height)
		mhod := writeArtMHODContainer(2, mhni)
		children = append(children, mhod...)
		childCount++
	}

	totalLen := mhiiHeaderSize + len(children)
	buf := make([]byte, mhiiHeaderSize)
	copy(buf[0:4], "mhii")
	binary.LittleEndian.PutUint32(buf[4:8], mhiiHeaderSize)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(totalLen))
	binary.LittleEndian.PutUint32(buf[12:16], uint32(childCount))
	binary.LittleEndian.PutUint32(buf[16:20], img.ImageID)
	binary.LittleEndian.PutUint64(buf[20:28], img.SongDBID)
	binary.LittleEndian.PutUint32(buf[48:52], uint32(img.SrcSize))
	return append(buf, children...)
}

func writeArtMHNI(formatID, itmbOffset, dataSize, width, height int) []byte {
	filename := fmt.Sprintf(":F%d_1.ithmb", formatID)
	mhod3 := writeArtMHODFilename(filename)

	totalLen := mhniHeaderSize + len(mhod3)
	buf := make([]byte, mhniHeaderSize)
	copy(buf[0:4], "mhni")
	binary.LittleEndian.PutUint32(buf[4:8], mhniHeaderSize)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(totalLen))
	binary.LittleEndian.PutUint32(buf[12:16], 1) // child count
	binary.LittleEndian.PutUint32(buf[16:20], uint32(formatID))
	binary.LittleEndian.PutUint32(buf[20:24], uint32(itmbOffset))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(dataSize))
	binary.LittleEndian.PutUint16(buf[32:34], uint16(height))
	binary.LittleEndian.PutUint16(buf[34:36], uint16(width))
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))
	return append(buf, mhod3...)
}

func writeArtMHODFilename(filename string) []byte {
	u16 := utf16.Encode([]rune(filename))
	strBytes := make([]byte, len(u16)*2)
	for i, c := range u16 {
		binary.LittleEndian.PutUint16(strBytes[i*2:i*2+2], c)
	}

	bodyLen := 12 + len(strBytes)
	pad := (4 - (len(strBytes) % 4)) % 4
	bodyLen += pad

	totalLen := mhodArtHeaderSize + bodyLen
	buf := make([]byte, mhodArtHeaderSize+12)
	copy(buf[0:4], "mhod")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(mhodArtHeaderSize))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(totalLen))
	binary.LittleEndian.PutUint16(buf[12:14], 3)
	binary.LittleEndian.PutUint32(buf[mhodArtHeaderSize:mhodArtHeaderSize+4], uint32(len(strBytes)))
	buf[mhodArtHeaderSize+4] = 2 // UTF-16LE encoding
	result := append(buf, strBytes...)
	if pad > 0 {
		result = append(result, make([]byte, pad)...)
	}
	return result
}

func writeArtMHODContainer(mhodType int, child []byte) []byte {
	totalLen := mhodArtHeaderSize + len(child)
	buf := make([]byte, mhodArtHeaderSize)
	copy(buf[0:4], "mhod")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(mhodArtHeaderSize))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(totalLen))
	binary.LittleEndian.PutUint16(buf[12:14], uint16(mhodType))
	return append(buf, child...)
}

func writeArtMHLA() []byte {
	buf := make([]byte, mhlaHeaderSize)
	copy(buf[0:4], "mhla")
	binary.LittleEndian.PutUint32(buf[4:8], mhlaHeaderSize)
	return buf
}

func writeArtMHLF(formats []ArtworkFormat, imageSizes map[int]int) []byte {
	var children []byte
	for _, f := range formats {
		children = append(children, writeArtMHIF(f.FormatID, imageSizes[f.FormatID])...)
	}
	buf := make([]byte, mhlfHeaderSize)
	copy(buf[0:4], "mhlf")
	binary.LittleEndian.PutUint32(buf[4:8], mhlfHeaderSize)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(len(formats)))
	return append(buf, children...)
}

func writeArtMHIF(formatID, imageSize int) []byte {
	buf := make([]byte, mhifHeaderSize)
	copy(buf[0:4], "mhif")
	binary.LittleEndian.PutUint32(buf[4:8], mhifHeaderSize)
	binary.LittleEndian.PutUint32(buf[8:12], mhifHeaderSize)
	binary.LittleEndian.PutUint32(buf[16:20], uint32(formatID))
	binary.LittleEndian.PutUint32(buf[20:24], uint32(imageSize))
	return buf
}
