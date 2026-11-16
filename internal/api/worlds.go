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

// WorldResult represents an aggregated VRChat world entry returned by GET /api/v1/worlds.
type WorldResult struct {
	Name       string    `json:"Name"`
	PhotoCount int       `json:"PhotoCount"`
	Thumb      string    `json:"Thumb"`
	TakenAt    time.Time `json:"TakenAt"`
}

type worldPhotoRow struct {
	PhotoCaption string
	Notes        string
	FileHash     string
	TakenAt      time.Time
}

// GetWorlds finds and returns VRChat worlds extracted from photo metadata as JSON.
//
// GET /api/v1/worlds
func GetWorlds(router *gin.RouterGroup) {
	router.GET("/worlds", func(c *gin.Context) {
		s := Auth(c, acl.ResourcePhotos, acl.ActionSearch)

		if s.Abort(c) {
			return
		}

		q := strings.ToLower(strings.TrimSpace(c.Query("q")))
		count := 100
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
			Where("photos.photo_caption LIKE ? OR details.notes LIKE ?", "%"+search.VRCWorldPrefix+"%", "%"+search.VRCWorldPrefix+"%").
			Order("photos.taken_at DESC")

		if acl.Rules.Deny(acl.ResourcePhotos, s.GetUserRole(), acl.AccessPrivate) {
			db = db.Where("photos.photo_private = 0")
		}

		var rows []worldPhotoRow
		if err := db.Scan(&rows).Error; err != nil {
			log.Errorf("worlds: %s", err)
			AbortSaveFailed(c)
			return
		}

		var groups []*WorldResult
		for _, row := range rows {
			rawName := search.ExtractVRCWorld(row.PhotoCaption)
			if rawName == "" {
				rawName = search.ExtractVRCWorld(row.Notes)
			}
			normName := search.NormalizeVRCWorld(rawName)
			if normName == "" {
				continue
			}

			var target *WorldResult
			for _, g := range groups {
				if search.MatchVRCWorld(g.Name, normName) {
					target = g
					break
				}
			}

			if target == nil {
				groups = append(groups, &WorldResult{
					Name:       normName,
					PhotoCount: 1,
					Thumb:      row.FileHash,
					TakenAt:    row.TakenAt,
				})
				continue
			}

			target.PhotoCount++
			// Prefer the shorter/cleaner display name for the group label.
			if len(normName) < len(target.Name) {
				target.Name = normName
			}
			if target.Thumb == "" && row.FileHash != "" {
				target.Thumb = row.FileHash
			}
		}

		results := make([]WorldResult, 0, len(groups))
		for _, g := range groups {
			if q != "" && !strings.Contains(strings.ToLower(g.Name), q) {
				continue
			}
			results = append(results, *g)
		}

		sort.Slice(results, func(i, j int) bool {
			if results[i].PhotoCount != results[j].PhotoCount {
				return results[i].PhotoCount > results[j].PhotoCount
			}
			return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
		})

		if offset >= len(results) {
			results = []WorldResult{}
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
