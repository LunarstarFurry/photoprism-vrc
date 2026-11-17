package api

import (
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

		byKey := make(map[string]*PlayerResult)
		for _, row := range rows {
			players := search.ExtractVRCPlayers(row.PhotoCaption, row.Notes)
			for _, name := range players {
				key := strings.ToLower(name)
				if q != "" && !strings.Contains(key, q) {
					continue
				}

				existing, ok := byKey[key]
				if !ok {
					byKey[key] = &PlayerResult{
						Name:       name,
						PhotoCount: 1,
						Thumb:      row.FileHash,
						TakenAt:    row.TakenAt,
					}
					continue
				}

				existing.PhotoCount++
				if existing.Thumb == "" && row.FileHash != "" {
					existing.Thumb = row.FileHash
				}
			}
		}

		results := make([]PlayerResult, 0, len(byKey))
		for _, p := range byKey {
			// When not searching and not explicitly requesting all, hide 1-photo passersby
			// so large group instances don't flood the grid.
			if q == "" && !showAll && p.PhotoCount < 2 {
				continue
			}
			results = append(results, *p)
		}

		sort.Slice(results, func(i, j int) bool {
			if results[i].PhotoCount != results[j].PhotoCount {
				return results[i].PhotoCount > results[j].PhotoCount
			}
			return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
		})

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
