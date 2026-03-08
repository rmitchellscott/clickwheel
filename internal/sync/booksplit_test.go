package sync

import (
	"testing"

	"clickwheel/internal/audiobookshelf"
)

func TestComputeBookSplitPlan(t *testing.T) {
	ch := func(start, end float64) audiobookshelf.Chapter {
		return audiobookshelf.Chapter{Start: start, End: end}
	}

	tests := []struct {
		name       string
		chapters   []audiobookshelf.Chapter
		totalDur   float64
		limitHours int
		wantParts  int
		wantNil    bool
	}{
		{
			name:       "short book no split",
			chapters:   []audiobookshelf.Chapter{ch(0, 3600), ch(3600, 7200)},
			totalDur:   7200,
			limitHours: 8,
			wantNil:    true,
		},
		{
			name:       "short book no chapters no split",
			chapters:   nil,
			totalDur:   7200,
			limitHours: 8,
			wantNil:    true,
		},
		{
			name:       "long book no chapters splits by duration",
			chapters:   nil,
			totalDur:   100000,
			limitHours: 8,
			wantParts:  4,
		},
		{
			name: "20h book split at 8h",
			chapters: []audiobookshelf.Chapter{
				ch(0, 3600), ch(3600, 7200), ch(7200, 10800), ch(10800, 14400),
				ch(14400, 18000), ch(18000, 21600), ch(21600, 25200), ch(25200, 28800),
				ch(28800, 32400), ch(32400, 36000), ch(36000, 39600), ch(39600, 43200),
				ch(43200, 46800), ch(46800, 50400), ch(50400, 54000), ch(54000, 57600),
				ch(57600, 61200), ch(61200, 64800), ch(64800, 68400), ch(68400, 72000),
			},
			totalDur:   72000,
			limitHours: 8,
			wantParts:  3,
		},
		{
			name: "single long chapter gets own part",
			chapters: []audiobookshelf.Chapter{
				ch(0, 3600), ch(3600, 7200), ch(7200, 10800),
				ch(10800, 50000),
				ch(50000, 54000),
			},
			totalDur:   54000,
			limitHours: 8,
			wantParts:  3,
		},
		{
			name: "chapters fit exactly at limit",
			chapters: []audiobookshelf.Chapter{
				ch(0, 14400), ch(14400, 28800),
			},
			totalDur:   28800,
			limitHours: 4,
			wantParts:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := computeBookSplitPlan(tt.chapters, tt.totalDur, tt.limitHours)
			if tt.wantNil {
				if parts != nil {
					t.Fatalf("expected nil, got %d parts", len(parts))
				}
				return
			}
			if len(parts) != tt.wantParts {
				t.Fatalf("expected %d parts, got %d: %+v", tt.wantParts, len(parts), parts)
			}
			if parts[0].StartSec != 0 {
				t.Errorf("first part should start at 0, got %f", parts[0].StartSec)
			}
			last := parts[len(parts)-1]
			if last.EndSec != tt.totalDur {
				t.Errorf("last part should end at %f, got %f", tt.totalDur, last.EndSec)
			}
			for i := 1; i < len(parts); i++ {
				if parts[i].StartSec != parts[i-1].EndSec {
					t.Errorf("gap between part %d end (%f) and part %d start (%f)",
						i-1, parts[i-1].EndSec, i, parts[i].StartSec)
				}
			}
		})
	}
}

func TestSplitBookSourceID(t *testing.T) {
	tests := []struct {
		input     string
		wantBook  string
		wantIndex int
		wantSplit bool
	}{
		{"abc123", "abc123", 0, false},
		{"abc123#0", "abc123", 0, true},
		{"abc123#5", "abc123", 5, true},
		{"abc#def#2", "abc#def", 2, true},
	}

	for _, tt := range tests {
		book, idx, isSplit := splitBookSourceID(tt.input)
		if book != tt.wantBook || idx != tt.wantIndex || isSplit != tt.wantSplit {
			t.Errorf("splitBookSourceID(%q) = (%q, %d, %v), want (%q, %d, %v)",
				tt.input, book, idx, isSplit, tt.wantBook, tt.wantIndex, tt.wantSplit)
		}
	}
}

func TestBookPartSourceID(t *testing.T) {
	if got := bookPartSourceID("abc123", 3); got != "abc123#3" {
		t.Errorf("bookPartSourceID(abc123, 3) = %q, want abc123#3", got)
	}
}
