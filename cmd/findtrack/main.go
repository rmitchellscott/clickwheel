package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
)

func main() {
	info, _ := ipod.Detect()
	data, _ := os.ReadFile(filepath.Join(info.MountPoint, "iPod_Control", "iTunes", "iTunesDB"))
	db, _ := itunesdb.Parse(data)
	for _, t := range db.Tracks {
		if strings.Contains(strings.ToLower(t.Title), "blue strip") {
			fmt.Printf("Title: %s\nArtist: %s\nSourceID: %s\nPath: %s\n", t.Title, t.Artist, t.SourceID, t.Path)
		}
	}
}
