package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
)

func main() {
	limit := 0
	if len(os.Args) > 1 {
		limit, _ = strconv.Atoi(os.Args[1])
	}

	info, err := ipod.Detect()
	if err != nil || info == nil {
		fmt.Fprintln(os.Stderr, "no iPod found")
		os.Exit(1)
	}

	dbPath := filepath.Join(info.MountPoint, "iPod_Control", "iTunes", "iTunesDB")
	bakPath := dbPath + ".bak"
	// Use backup if it exists, otherwise backup current and use it
	data, err := os.ReadFile(bakPath)
	if err != nil {
		data, err = os.ReadFile(dbPath)
		if err == nil {
			os.WriteFile(bakPath, data, 0644)
			fmt.Println("Created backup from current DB")
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "read backup: %v\n", err)
		os.Exit(1)
	}

	origDB, err := itunesdb.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	tracks := origDB.Tracks
	start := 0
	if len(os.Args) > 2 {
		start, _ = strconv.Atoi(os.Args[2])
	}
	if start > 0 && start < len(tracks) {
		tracks = tracks[start:]
	}
	if limit > 0 && limit < len(tracks) {
		tracks = tracks[:limit]
	}

	db := itunesdb.NewDatabase(info.Name)
	db.Tracks = tracks
	db.Playlists[0].Tracks = tracks

	dev, _ := ipod.OpenDevice(info)
	caps := dev.Capabilities()

	minimal := len(os.Args) > 3 && os.Args[3] == "minimal"
	if minimal {
		caps.SupportsPodcast = false
		caps.SupportsLibraryIndex = false
		fmt.Println("MINIMAL mode: types 1+2 only")
	}

	out := itunesdb.SerializeDatabase(db, caps)
	fmt.Printf("Writing %d tracks (%d bytes)\n", len(tracks), len(out))

	if err := os.WriteFile(dbPath, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done. Eject and check.")
}
