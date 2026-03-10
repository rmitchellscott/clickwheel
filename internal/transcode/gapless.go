package transcode

import (
	"encoding/binary"
	"io"
	"os"
)

type GaplessInfo struct {
	EncoderDelay uint32
	Padding      uint32
	SampleCount  uint64
}

func ProbeGapless(filePath string) *GaplessInfo {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	header := make([]byte, 4)
	if _, err := io.ReadFull(f, header); err != nil {
		return nil
	}
	f.Seek(0, io.SeekStart)

	if header[0] == 0xFF && (header[1]&0xE0) == 0xE0 {
		return probeMP3Gapless(f)
	}

	if string(header[:4]) == "ftyp" || (header[0] == 0 && header[3] > 0) {
		return probeM4AGapless(f)
	}

	return nil
}

func probeMP3Gapless(f *os.File) *GaplessInfo {
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	if n < 200 {
		return nil
	}

	for i := 0; i < n-4; i++ {
		if string(buf[i:i+4]) == "Xing" || string(buf[i:i+4]) == "Info" {
			return parseLAMEHeader(buf[i:n])
		}
	}
	return nil
}

func parseLAMEHeader(data []byte) *GaplessInfo {
	if len(data) < 120+4+24 {
		return nil
	}

	flags := binary.BigEndian.Uint32(data[4:8])
	offset := 8
	if flags&0x01 != 0 { // frames
		offset += 4
	}
	if flags&0x02 != 0 { // bytes
		offset += 4
	}
	if flags&0x04 != 0 { // toc
		offset += 100
	}
	if flags&0x08 != 0 { // quality
		offset += 4
	}

	if offset+9+12 > len(data) {
		return nil
	}

	if string(data[offset:offset+4]) != "LAME" && string(data[offset:offset+4]) != "Lavf" && string(data[offset:offset+4]) != "Lavc" {
		return nil
	}

	gapOffset := offset + 9 + 12
	if gapOffset+3 > len(data) {
		return nil
	}

	delay := (uint32(data[gapOffset]) << 4) | (uint32(data[gapOffset+1]) >> 4)
	padding := (uint32(data[gapOffset+1]&0x0F) << 8) | uint32(data[gapOffset+2])

	var totalFrames uint32
	if flags&0x01 != 0 {
		totalFrames = binary.BigEndian.Uint32(data[8:12])
	}

	var sampleCount uint64
	if totalFrames > 0 {
		sampleCount = uint64(totalFrames)*1152 - uint64(delay) - uint64(padding)
	}

	if delay == 0 && padding == 0 {
		return nil
	}

	return &GaplessInfo{
		EncoderDelay: delay,
		Padding:      padding,
		SampleCount:  sampleCount,
	}
}

func probeM4AGapless(f *os.File) *GaplessInfo {
	f.Seek(0, io.SeekStart)
	stat, err := f.Stat()
	if err != nil {
		return nil
	}
	fileSize := stat.Size()

	var info GaplessInfo
	found := false

	var walk func(offset, end int64) bool
	walk = func(offset, end int64) bool {
		for offset < end-8 {
			header := make([]byte, 8)
			if _, err := f.ReadAt(header, offset); err != nil {
				return false
			}
			size := int64(binary.BigEndian.Uint32(header[0:4]))
			atomType := string(header[4:8])

			if size == 0 {
				size = end - offset
			} else if size == 1 {
				ext := make([]byte, 8)
				if _, err := f.ReadAt(ext, offset+8); err != nil {
					return false
				}
				size = int64(binary.BigEndian.Uint64(ext))
			}
			if size < 8 || offset+size > end {
				return false
			}

			switch atomType {
			case "moov", "trak", "mdia", "minf", "stbl", "udta":
				headerSize := int64(8)
				if walk(offset+headerSize, offset+size) {
					return true
				}
			case "stts":
				if size > 16 {
					data := make([]byte, size-8)
					if _, err := f.ReadAt(data, offset+8); err != nil {
						break
					}
					if len(data) >= 8 {
						entryCount := binary.BigEndian.Uint32(data[4:8])
						var total uint64
						for i := uint32(0); i < entryCount && int(12+i*8+4) <= len(data); i++ {
							count := binary.BigEndian.Uint32(data[8+i*8 : 12+i*8])
							dur := binary.BigEndian.Uint32(data[12+i*8 : 16+i*8])
							total += uint64(count) * uint64(dur)
						}
						if total > 0 {
							info.SampleCount = total
						}
					}
				}
			case "edts":
				if walk(offset+8, offset+size) {
					return true
				}
			case "elst":
				if size > 20 {
					data := make([]byte, size-8)
					if _, err := f.ReadAt(data, offset+8); err != nil {
						break
					}
					if len(data) >= 16 {
						version := data[0]
						if version == 0 && len(data) >= 16 {
							mediaTime := int32(binary.BigEndian.Uint32(data[8:12]))
							if mediaTime > 0 {
								info.EncoderDelay = uint32(mediaTime)
								found = true
							}
						}
					}
				}
			}
			offset += size
		}
		return false
	}

	walk(0, fileSize)
	if found || info.SampleCount > 0 {
		return &info
	}
	return nil
}
