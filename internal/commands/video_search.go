package commands

import (
	"github.com/photoprism/photoprism/internal/entity/search"
	"github.com/photoprism/photoprism/internal/entity/sortby"
	"github.com/photoprism/photoprism/internal/form"
)

// videoSearchResults runs a video-only search and applies offset/count after sidecar filtering.
func videoSearchResults(query string, count int, offset int, includeSidecar bool) ([]search.Photo, error) {
	if offset < 0 {
		offset = 0
	}

	if count <= 0 {
		return []search.Photo{}, nil
	}

	frm := form.SearchPhotos{
		Query:   query,
		Primary: false,
		Merged:  false,
		Video:   true,
		Order:   sortby.Name,
	}

	if includeSidecar {
		frm.Count = count
		frm.Offset = offset
		results, _, err := search.Photos(frm)
		return results, err
	}

	target := count + offset
	if target < 0 {
		target = 0
	}

	collected := make([]search.Photo, 0, target)
	searchOffset := 0
	batchSize := count
	if batchSize < 200 {
		batchSize = 200
	}

	for len(collected) < target {
		frm.Count = batchSize
		frm.Offset = searchOffset

		results, _, err := search.Photos(frm)
		if err != nil {
			return nil, err
		}

		if len(results) == 0 {
			break
		}

		for _, found := range results {
			if found.FileSidecar {
				continue
			}
			collected = append(collected, found)
		}

		searchOffset += len(results)
		if len(results) < batchSize {
			break
		}
	}

	if offset >= len(collected) {
		return []search.Photo{}, nil
	}

	end := offset + count
	if end > len(collected) {
		end = len(collected)
	}

	return collected[offset:end], nil
}
