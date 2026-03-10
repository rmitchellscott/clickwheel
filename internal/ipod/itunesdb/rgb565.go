package itunesdb

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
)

func DecodeImage(data []byte) (image.Image, error) {
	r := bytes.NewReader(data)
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func init() {
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("png", "\x89PNG", png.Decode, png.DecodeConfig)
}

func ResizeImage(img image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	return dst
}

func EncodeRGB565(img image.Image) []byte {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	buf := make([]byte, w*h*2)

	for y := range h {
		for x := range w {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			r5 := uint16(r>>11) & 0x1F
			g6 := uint16(g>>10) & 0x3F
			b5 := uint16(b>>11) & 0x1F
			pixel := (r5 << 11) | (g6 << 5) | b5
			offset := (y*w + x) * 2
			binary.LittleEndian.PutUint16(buf[offset:offset+2], pixel)
		}
	}
	return buf
}

func ConvertArtForIPod(imgData []byte, format ArtworkFormat) ([]byte, error) {
	img, err := DecodeImage(imgData)
	if err != nil {
		return nil, err
	}
	resized := ResizeImage(img, format.Width, format.Height)
	return EncodeRGB565(resized), nil
}

func DecodeRGB565(data []byte, width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			offset := (y*width + x) * 2
			if offset+2 > len(data) {
				break
			}
			pixel := binary.LittleEndian.Uint16(data[offset : offset+2])
			r := uint8(((pixel >> 11) & 0x1F) * 255 / 31)
			g := uint8(((pixel >> 5) & 0x3F) * 255 / 63)
			b := uint8((pixel & 0x1F) * 255 / 31)
			i := img.PixOffset(x, y)
			img.Pix[i+0] = r
			img.Pix[i+1] = g
			img.Pix[i+2] = b
			img.Pix[i+3] = 255
		}
	}
	return img
}

func EncodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
