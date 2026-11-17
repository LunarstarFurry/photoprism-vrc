package api

import (
	"hash/fnv"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/photoprism/photoprism/internal/auth/acl"
	"github.com/photoprism/photoprism/internal/entity"
	"github.com/photoprism/photoprism/internal/entity/search"
)

// PlayerResult represents an aggregated VRChat player entry returned by GET /api/v1/players.
type PlayerResult struct {
	Name       string    `json:"Name"`
	PhotoCount int       `json:"PhotoCount"`
	Thumb      string    `json:"Thumb"`
	TakenAt    time.Time `json:"TakenAt"`
}

type playerPhotoRow struct {
	PhotoCaption string
	Notes        string
	FileHash     string
	TakenAt      time.Time
}

// GetPlayers finds and returns VRChat players extracted from photo metadata as JSON.
//
// GET /api/v1/players
func GetPlayers(router *gin.RouterGroup) {
	router.GET("/players", func(c *gin.Context) {
		s := Auth(c, acl.ResourcePhotos, acl.ActionSearch)

		if s.Abort(c) {
			return
		}

		q := strings.ToLower(strings.TrimSpace(c.Query("q")))
		showAll := c.Query("all") == "true" || c.Query("all") == "1"
		count := 240
		if rawCount := c.Query("count"); rawCount != "" {
			if parsed, err := strconv.Atoi(rawCount); err == nil && parsed > 0 {
				count = min(parsed, 1000)
			}
		}
		offset := 0
		if rawOffset := c.Query("offset"); rawOffset != "" {
			if parsed, err := strconv.Atoi(rawOffset); err == nil && parsed >= 0 {
				offset = parsed
			}
		}

		db := entity.Db().
			Table("photos").
			Select("photos.photo_caption, COALESCE(details.notes, '') AS notes, files.file_hash, photos.taken_at").
			Joins("JOIN files ON files.photo_id = photos.id AND files.file_primary = 1 AND files.deleted_at IS NULL").
			Joins("LEFT JOIN details ON details.photo_id = photos.id").
			Where("photos.deleted_at IS NULL").
			Where("photos.photo_caption LIKE ? OR details.notes LIKE ?", "%VRChat - %", "%VRChat - %").
			Order("photos.taken_at DESC")

		if acl.Rules.Deny(acl.ResourcePhotos, s.GetUserRole(), acl.AccessPrivate) {
			db = db.Where("photos.photo_private = 0")
		}

		var rows []playerPhotoRow
		if err := db.Scan(&rows).Error; err != nil {
			log.Errorf("players: %s", err)
			AbortSaveFailed(c)
			return
		}

		type playerAgg struct {
			result PlayerResult
			thumbs []string
		}

		byKey := make(map[string]*playerAgg)
		for _, row := range rows {
			players := search.ExtractVRCPlayers(row.PhotoCaption, row.Notes)
			for _, name := range players {
				key := strings.ToLower(name)
				if q != "" && !strings.Contains(key, q) {
					continue
				}

				existing, ok := byKey[key]
				if !ok {
					agg := &playerAgg{
						result: PlayerResult{
							Name:       name,
							PhotoCount: 1,
							TakenAt:    row.TakenAt,
						},
					}
					if row.FileHash != "" {
						agg.thumbs = append(agg.thumbs, row.FileHash)
					}
					byKey[key] = agg
					continue
				}

				existing.result.PhotoCount++
				if row.FileHash != "" {
					existing.thumbs = append(existing.thumbs, row.FileHash)
				}
			}
		}

		aggs := make([]*playerAgg, 0, len(byKey))
		for _, agg := range byKey {
			// When not searching and not explicitly requesting all, hide 1-photo passersby
			// so large group instances don't flood the grid.
			if q == "" && !showAll && agg.result.PhotoCount < 2 {
				continue
			}
			aggs = append(aggs, agg)
		}

		sort.Slice(aggs, func(i, j int) bool {
			if aggs[i].result.PhotoCount != aggs[j].result.PhotoCount {
				return aggs[i].result.PhotoCount > aggs[j].result.PhotoCount
			}
			return strings.ToLower(aggs[i].result.Name) < strings.ToLower(aggs[j].result.Name)
		})

		// Pick a distinct, deterministic cover thumbnail per player so cards don't all
		// display the same newest group photo.
		usedThumbs := make(map[string]bool, len(aggs))
		results := make([]PlayerResult, 0, len(aggs))
		for _, agg := range aggs {
			if n := len(agg.thumbs); n > 0 {
				h := fnv.New32a()
				_, _ = h.Write([]byte(strings.ToLower(agg.result.Name)))
				start := int(h.Sum32()) % n

				chosen := agg.thumbs[start]
				for i := 0; i < n; i++ {
					cand := agg.thumbs[(start+i)%n]
					if !usedThumbs[cand] {
						chosen = cand
						break
					}
				}
				usedThumbs[chosen] = true
				agg.result.Thumb = chosen
			}
			results = append(results, agg.result)
		}

		if offset >= len(results) {
			results = []PlayerResult{}
		} else {
			end := min(offset+count, len(results))
			results = results[offset:end]
		}

		AddLimitHeader(c, count)
		AddOffsetHeader(c, offset)
		AddTokenHeaders(c, s)

		c.JSON(http.StatusOK, results)
	})
}
