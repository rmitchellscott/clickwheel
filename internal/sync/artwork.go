package sync

import (
	"crypto/md5"
	"log"
	"path/filepath"

	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
	"clickwheel/internal/subsonic"
)

func syncArtwork(dev *ipod.Device, sub *subsonic.Client, onProgress ProgressFunc) {
	caps := dev.Capabilities()
	if !caps.SupportsArtwork || len(caps.CoverArtFormats) == 0 {
		return
	}
	if sub == nil {
		return
	}

	onProgress(Progress{Phase: "artwork", Message: "Syncing album artwork..."})

	type artEntry struct {
		hash    [16]byte
		imgData []byte
	}
	artByHash := make(map[[16]byte]*artEntry)
	trackArt := make(map[uint64][16]byte) // DBID → art hash

	for _, track := range dev.DB.Tracks {
		if track.SourceID == "" || track.MediaType != itunesdb.MediaTypeMusic {
			continue
		}

		coverID := track.CoverArtID
		if coverID == "" {
			continue
		}

		data, err := sub.GetCoverArt(coverID, 320)
		if err != nil {
			log.Printf("[artwork] failed to fetch cover art for %s: %v", track.Title, err)
			continue
		}
		if len(data) == 0 {
			continue
		}

		hash := md5.Sum(data)
		if _, ok := artByHash[hash]; !ok {
			artByHash[hash] = &artEntry{hash: hash, imgData: data}
		}
		trackArt[track.DBID] = hash
	}

	if len(artByHash) == 0 {
		return
	}

	log.Printf("[artwork] fetched %d unique cover art images for %d tracks", len(artByHash), len(trackArt))

	hashToImageID := make(map[[16]byte]uint32)
	var images []*itunesdb.ArtworkImage
	imageID := uint32(100)

	for _, track := range dev.DB.Tracks {
		hash, ok := trackArt[track.DBID]
		if !ok {
			continue
		}

		entry := artByHash[hash]
		formats := make(map[int][]byte)
		for _, af := range caps.CoverArtFormats {
			rgb, err := itunesdb.ConvertArtForIPod(entry.imgData, af)
			if err != nil {
				log.Printf("[artwork] failed to convert art for format %d: %v", af.FormatID, err)
				continue
			}
			formats[af.FormatID] = rgb
		}
		if len(formats) == 0 {
			continue
		}

		img := &itunesdb.ArtworkImage{
			ImageID:  imageID,
			SongDBID: track.DBID,
			Formats:  formats,
			SrcSize:  len(entry.imgData),
		}
		images = append(images, img)

		if _, exists := hashToImageID[hash]; !exists {
			hashToImageID[hash] = imageID
		}

		track.MHIILink = imageID
		track.ArtworkCount = uint16(len(formats))
		track.ArtworkSize = uint32(len(entry.imgData))
		imageID++
	}

	if len(images) == 0 {
		return
	}

	artworkDir := filepath.Join(dev.Info.MountPoint, "iPod_Control", "Artwork")
	if err := itunesdb.WriteArtworkDB(artworkDir, images, caps.CoverArtFormats); err != nil {
		log.Printf("[artwork] failed to write ArtworkDB: %v", err)
		return
	}

	log.Printf("[artwork] wrote ArtworkDB with %d images", len(images))
}
