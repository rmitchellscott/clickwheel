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

	type coverGroup struct {
		tracks []*itunesdb.Track
		data   []byte
		hash   [16]byte
	}
	coverGroups := make(map[string]*coverGroup)
	for _, track := range dev.DB.Tracks {
		if track.SourceID == "" || track.MediaType != itunesdb.MediaTypeMusic {
			continue
		}
		coverID := track.CoverArtID
		if coverID == "" {
			continue
		}
		g, ok := coverGroups[coverID]
		if !ok {
			g = &coverGroup{}
			coverGroups[coverID] = g
		}
		g.tracks = append(g.tracks, track)
	}

	for coverID, g := range coverGroups {
		data, err := sub.GetCoverArt(coverID, 320)
		if err != nil {
			log.Printf("[artwork] failed to fetch cover art %s: %v", coverID, err)
			continue
		}
		if len(data) == 0 {
			continue
		}
		g.data = data
		g.hash = md5.Sum(data)
	}

	type convertedArt struct {
		formats map[int][]byte
		srcSize int
	}
	artByHash := make(map[[16]byte]*convertedArt)

	for _, g := range coverGroups {
		if len(g.data) == 0 {
			continue
		}
		if _, done := artByHash[g.hash]; done {
			continue
		}
		formats := make(map[int][]byte)
		for _, af := range caps.CoverArtFormats {
			rgb, err := itunesdb.ConvertArtForIPod(g.data, af)
			if err != nil {
				log.Printf("[artwork] failed to convert art for format %d: %v", af.FormatID, err)
				continue
			}
			formats[af.FormatID] = rgb
		}
		if len(formats) > 0 {
			artByHash[g.hash] = &convertedArt{formats: formats, srcSize: len(g.data)}
		}
		g.data = nil
	}

	if len(artByHash) == 0 {
		return
	}

	log.Printf("[artwork] converted %d unique cover art images", len(artByHash))

	var images []*itunesdb.ArtworkImage
	imageID := uint32(100)

	for _, g := range coverGroups {
		art, ok := artByHash[g.hash]
		if !ok {
			continue
		}
		for _, track := range g.tracks {
			img := &itunesdb.ArtworkImage{
				ImageID:  imageID,
				SongDBID: track.DBID,
				Formats:  art.formats,
				SrcSize:  art.srcSize,
			}
			images = append(images, img)
			track.MHIILink = imageID
			track.ArtworkCount = uint16(len(art.formats))
			track.ArtworkSize = uint32(art.srcSize)
			imageID++
		}
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
