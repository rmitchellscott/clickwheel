package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"clickwheel/internal/ipod"
	"clickwheel/internal/ipod/itunesdb"
)

func main() {
	info, err := ipod.Detect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "detect error: %v\n", err)
		os.Exit(1)
	}
	if info == nil {
		fmt.Fprintln(os.Stderr, "no iPod found")
		os.Exit(1)
	}
	fmt.Printf("iPod: %s at %s (family=%s gen=%s model=%s icon=%s)\n", info.Name, info.MountPoint, info.Family, info.Generation, info.Model, info.Icon)

	dbPath := filepath.Join(info.MountPoint, "iPod_Control", "iTunes", "iTunesDB")
	data, err := os.ReadFile(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read iTunesDB: %v\n", err)
		os.Exit(1)
	}

	db, err := itunesdb.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse iTunesDB: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Tracks: %d, Playlists: %d\n\n", len(db.Tracks), len(db.Playlists))

	// Check Play Counts file
	pcPath := filepath.Join(info.MountPoint, "iPod_Control", "iTunes", "Play Counts")
	if pcData, err := os.ReadFile(pcPath); err == nil {
		fmt.Printf("Play Counts file: %d bytes\n", len(pcData))
		if len(pcData) >= 16 {
			fmt.Printf("  header bytes: %x\n", pcData[:16])
			entrySize := uint32(pcData[8]) | uint32(pcData[9])<<8 | uint32(pcData[10])<<16 | uint32(pcData[11])<<24
			entryCount := uint32(pcData[12]) | uint32(pcData[13])<<8 | uint32(pcData[14])<<16 | uint32(pcData[15])<<24
			fmt.Printf("  entry size: %d, entry count: %d\n", entrySize, entryCount)
		}
		entries, err := itunesdb.ParsePlayCounts(pcData)
		if err != nil {
			fmt.Printf("  parse error: %v\n", err)
		} else {
			fmt.Printf("  parsed %d entries\n", len(entries))
			nonzero := 0
			for _, e := range entries {
				if e.PlayCount > 0 || e.LastPlayed > 0 {
					nonzero++
				}
			}
			fmt.Printf("  %d entries with non-zero play count or last played\n", nonzero)
			for i, e := range entries {
				if e.PlayCount > 0 || e.LastPlayed > 0 {
					lp := "never"
					if e.LastPlayed != 0 {
						lp = itunesdb.FromMacTimestamp(e.LastPlayed).Format(time.RFC3339)
					}
					fmt.Printf("  [%d] plays=%d lastPlayed=%s (raw=%d) bookmark=%d rating=%d skips=%d\n",
						i, e.PlayCount, lp, e.LastPlayed, e.BookmarkTime, e.Rating, e.SkipCount)
					if i > 20 {
						fmt.Println("  ... (truncated)")
						break
					}
				}
			}
		}
		fmt.Println()
	} else {
		fmt.Println("No Play Counts file found\n")
	}

	fmt.Println("=== All tracks (from iTunesDB) ===")
	for i, t := range db.Tracks {
		lp := "never"
		if t.LastPlayed != 0 {
			lp = itunesdb.FromMacTimestamp(t.LastPlayed).Format(time.RFC3339)
		}
		da := "unknown"
		if !t.DateAdded.IsZero() {
			da = t.DateAdded.Format(time.RFC3339)
		}
		fmt.Printf("[%d] %q by %q\n", i, t.Title, t.Artist)
		fmt.Printf("    UniqueID=%d MediaType=%d Size=%d Duration=%dms\n", t.UniqueID, t.MediaType, t.Size, t.Duration)
		fmt.Printf("    PlayCount=%d LastPlayed=%s (raw=%d)\n", t.PlayCount, lp, t.LastPlayed)
		fmt.Printf("    DateAdded=%s BookmarkTime=%d\n", da, t.BookmarkTime)
		fmt.Printf("    PodcastFlag=%d SourceID=%q\n", t.PodcastFlag, t.SourceID)
		fmt.Printf("    Path=%s\n", t.Path)
		fmt.Println()
	}

	fmt.Println("=== Playlists ===")
	for i, pl := range db.Playlists {
		fmt.Printf("[%d] %q master=%v podcastFlag=%d tracks=%d\n",
			i, pl.Name, pl.IsMaster, pl.PodcastFlag, len(pl.Tracks))
	}
	fmt.Println()

	// After merging play counts
	if pcData, err := os.ReadFile(pcPath); err == nil {
		if entries, err := itunesdb.ParsePlayCounts(pcData); err == nil {
			itunesdb.MergePlayCounts(db, entries)
			fmt.Println("=== After merging Play Counts ===")
			for i, t := range db.Tracks {
				lp := "never"
				if t.LastPlayed != 0 {
					lp = itunesdb.FromMacTimestamp(t.LastPlayed).Format(time.RFC3339)
				}
				fmt.Printf("[%d] %q plays=%d lastPlayed=%s\n", i, t.Title, t.PlayCount, lp)
			}
		}
	}

	// Dump raw track with highest play count
	fmt.Println("\n=== Tracks with plays (from DB before merge) ===")
	// Re-parse to get clean data
	db2, _ := itunesdb.Parse(data)
	for _, t := range db2.Tracks {
		if t.PlayCount > 0 {
			lp := "never"
			if t.LastPlayed != 0 {
				lp = itunesdb.FromMacTimestamp(t.LastPlayed).Format(time.RFC3339)
			}
			fmt.Printf("  %q plays=%d lastPlayed=%s (raw=%d)\n", t.Title, t.PlayCount, lp, t.LastPlayed)
		}
	}

	_ = json.Marshal // keep import
}
