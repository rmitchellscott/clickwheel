package sync

import (
	"fmt"
	"strconv"
	"strings"

	"clickwheel/internal/audiobookshelf"
	"clickwheel/internal/config"
)

func computeBookSplitPlan(chapters []audiobookshelf.Chapter, totalDuration float64, limitHours int) []config.BookSplitPart {
	limitSec := float64(limitHours) * 3600

	if totalDuration <= limitSec {
		return nil
	}

	if len(chapters) == 0 {
		return splitByDuration(totalDuration, limitSec)
	}

	var parts []config.BookSplitPart
	partStart := 0.0
	partEnd := 0.0

	for _, ch := range chapters {
		chEnd := ch.End
		if chEnd > totalDuration {
			chEnd = totalDuration
		}

		wouldExceed := (chEnd - partStart) > limitSec
		hasContent := partEnd > partStart

		if wouldExceed && hasContent {
			parts = append(parts, config.BookSplitPart{
				Index:    len(parts),
				StartSec: partStart,
				EndSec:   partEnd,
			})
			partStart = ch.Start
		}
		partEnd = chEnd
	}

	if partEnd > partStart {
		parts = append(parts, config.BookSplitPart{
			Index:    len(parts),
			StartSec: partStart,
			EndSec:   partEnd,
		})
	}

	if len(parts) <= 1 {
		return nil
	}

	return parts
}

func splitByDuration(totalDuration, limitSec float64) []config.BookSplitPart {
	var parts []config.BookSplitPart
	pos := 0.0
	for pos < totalDuration {
		end := pos + limitSec
		if end > totalDuration {
			end = totalDuration
		}
		parts = append(parts, config.BookSplitPart{
			Index:    len(parts),
			StartSec: pos,
			EndSec:   end,
		})
		pos = end
	}
	if len(parts) <= 1 {
		return nil
	}
	return parts
}

func splitBookSourceID(sourceID string) (bookID string, partIndex int, isSplit bool) {
	idx := strings.LastIndex(sourceID, "#")
	if idx < 0 {
		return sourceID, 0, false
	}
	n, err := strconv.Atoi(sourceID[idx+1:])
	if err != nil {
		return sourceID, 0, false
	}
	return sourceID[:idx], n, true
}

func bookPartSourceID(bookID string, index int) string {
	return fmt.Sprintf("%s#%d", bookID, index)
}
