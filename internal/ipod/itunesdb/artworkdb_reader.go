package itunesdb

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ArtworkRef struct {
	FormatID int
	Width    int
	Height   int
	ItmbFile string
	Offset   int
	DataSize int
}

type TrackArtwork struct {
	DBID    uint64
	ImageID uint32
	Refs    []ArtworkRef
}

func ReadArtworkDB(artworkDir string) (map[uint64]*TrackArtwork, error) {
	dbPath := filepath.Join(artworkDir, "ArtworkDB")
	data, err := os.ReadFile(dbPath)
	if err != nil {
		return nil, err
	}
	if len(data) < 12 || string(data[0:4]) != "mhfd" {
		return nil, fmt.Errorf("invalid ArtworkDB")
	}

	totalLen := int(binary.LittleEndian.Uint32(data[8:12]))
	totalLen = min(totalLen, len(data))

	result := make(map[uint64]*TrackArtwork)
	headerLen := int(binary.LittleEndian.Uint32(data[4:8]))
	offset := headerLen

	for offset < totalLen-8 {
		if string(data[offset:offset+4]) != "mhsd" {
			break
		}
		mhsdHeader := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		mhsdTotal := int(binary.LittleEndian.Uint32(data[offset+8 : offset+12]))
		dsType := binary.LittleEndian.Uint16(data[offset+12 : offset+14])

		if dsType == 1 {
			parseArtMHLI(data[offset+mhsdHeader:offset+mhsdTotal], result)
		}
		offset += mhsdTotal
	}

	return result, nil
}

func parseArtMHLI(data []byte, result map[uint64]*TrackArtwork) {
	if len(data) < 12 || string(data[0:4]) != "mhli" {
		return
	}
	headerLen := int(binary.LittleEndian.Uint32(data[4:8]))
	count := int(binary.LittleEndian.Uint32(data[8:12]))
	offset := headerLen

	for i := 0; i < count && offset < len(data)-8; i++ {
		if string(data[offset:offset+4]) != "mhii" {
			break
		}
		mhiiHeader := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		mhiiTotal := int(binary.LittleEndian.Uint32(data[offset+8 : offset+12]))

		if mhiiHeader >= 28 {
			imageID := binary.LittleEndian.Uint32(data[offset+16 : offset+20])
			songDBID := binary.LittleEndian.Uint64(data[offset+20 : offset+28])

			refs := parseArtMHIIChildren(data[offset+mhiiHeader : offset+mhiiTotal])
			if len(refs) > 0 {
				result[songDBID] = &TrackArtwork{
					DBID:    songDBID,
					ImageID: imageID,
					Refs:    refs,
				}
			}
		}
		offset += mhiiTotal
	}
}

func parseArtMHIIChildren(data []byte) []ArtworkRef {
	var refs []ArtworkRef
	offset := 0

	for offset < len(data)-8 {
		magic := string(data[offset : offset+4])
		if magic != "mhod" {
			break
		}
		mhodHeader := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		mhodTotal := int(binary.LittleEndian.Uint32(data[offset+8 : offset+12]))
		mhodType := binary.LittleEndian.Uint16(data[offset+12 : offset+14])

		if mhodType == 2 && mhodHeader+8 <= mhodTotal {
			childData := data[offset+mhodHeader : offset+mhodTotal]
			if ref, ok := parseArtMHNI(childData); ok {
				refs = append(refs, ref)
			}
		}
		offset += mhodTotal
	}
	return refs
}

func parseArtMHNI(data []byte) (ArtworkRef, bool) {
	if len(data) < 44 || string(data[0:4]) != "mhni" {
		return ArtworkRef{}, false
	}
	mhniHeader := int(binary.LittleEndian.Uint32(data[4:8]))
	mhniTotal := int(binary.LittleEndian.Uint32(data[8:12]))

	formatID := int(binary.LittleEndian.Uint32(data[16:20]))
	itmbOffset := int(binary.LittleEndian.Uint32(data[20:24]))
	dataSize := int(binary.LittleEndian.Uint32(data[24:28]))
	height := int(binary.LittleEndian.Uint16(data[32:34]))
	width := int(binary.LittleEndian.Uint16(data[34:36]))

	filename := ""
	childOffset := mhniHeader
	for childOffset < mhniTotal-8 && childOffset < len(data)-8 {
		if string(data[childOffset:childOffset+4]) != "mhod" {
			break
		}
		cmhodHeader := int(binary.LittleEndian.Uint32(data[childOffset+4 : childOffset+8]))
		cmhodTotal := int(binary.LittleEndian.Uint32(data[childOffset+8 : childOffset+12]))
		cmhodType := binary.LittleEndian.Uint16(data[childOffset+12 : childOffset+14])

		if cmhodType == 3 && cmhodHeader+4 <= cmhodTotal {
			strData := data[childOffset+cmhodHeader : childOffset+cmhodTotal]
			if len(strData) >= 4 {
				strLen := int(binary.LittleEndian.Uint32(strData[0:4]))
				if len(strData) >= 8+strLen {
					u16 := make([]uint16, strLen/2)
					for i := range u16 {
						u16[i] = binary.LittleEndian.Uint16(strData[8+i*2 : 10+i*2])
					}
					runes := make([]rune, len(u16))
					for i, c := range u16 {
						runes[i] = rune(c)
					}
					filename = string(runes)
				}
			}
		}
		childOffset += cmhodTotal
	}

	if filename == "" {
		filename = fmt.Sprintf(":F%d_1.ithmb", formatID)
	}

	return ArtworkRef{
		FormatID: formatID,
		Width:    width,
		Height:   height,
		ItmbFile: strings.TrimPrefix(filename, ":"),
		Offset:   itmbOffset,
		DataSize: dataSize,
	}, true
}

func ReadArtworkData(artworkDir string, ref ArtworkRef) ([]byte, error) {
	itmbPath := filepath.Join(artworkDir, ref.ItmbFile)
	f, err := os.Open(itmbPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data := make([]byte, ref.DataSize)
	if _, err := f.ReadAt(data, int64(ref.Offset)); err != nil {
		return nil, err
	}
	return data, nil
}
