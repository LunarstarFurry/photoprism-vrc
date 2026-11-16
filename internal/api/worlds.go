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

const vrcWorldPrefix = "VRChat - World: "

// extractVRCWorld extracts the VRChat world name from a formatted caption or notes string.
func extractVRCWorld(s string) string {
	idx := strings.Index(s, vrcWorldPrefix)
	if idx == -1 {
		return ""
	}

	rest := s[idx+len(vrcWorldPrefix):]

	for _, sep := range []string{" | ", " / ", "\n", "\r"} {
		if end := strings.Index(rest, sep); end != -1 {
			rest = rest[:end]
		}
	}

	return strings.TrimSpace(rest)
}

// normalizeVRCWorld strips trailing parenthetical/bracketed event or version tags
// (e.g. "Polydance Studio (Just Dance Scoring Now Live!)" -> "Polydance Studio")
// so renamed variants of the same world group together.
func normalizeVRCWorld(name string) string {
	n := strings.TrimSpace(name)
	for {
		trimmed := n
		if strings.HasSuffix(trimmed, ")") {
			if idx := strings.LastIndex(trimmed, "("); idx > 0 {
				trimmed = strings.TrimSpace(trimmed[:idx])
			}
		} else if strings.HasSuffix(trimmed, "]") {
			if idx := strings.LastIndex(trimmed, "["); idx > 0 {
				trimmed = strings.TrimSpace(trimmed[:idx])
			}
		}
		if trimmed == n || trimmed == "" {
			break
		}
		n = trimmed
	}
	return n
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
			Where("photos.photo_caption LIKE ? OR details.notes LIKE ?", "%"+vrcWorldPrefix+"%", "%"+vrcWorldPrefix+"%").
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

		byName := make(map[string]*WorldResult)
		for _, row := range rows {
			name := extractVRCWorld(row.PhotoCaption)
			if name == "" {
				name = extractVRCWorld(row.Notes)
			}
			name = normalizeVRCWorld(name)
			if name == "" {
				continue
			}

			if q != "" && !strings.Contains(strings.ToLower(name), q) {
				continue
			}

			existing, ok := byName[name]
			if !ok {
				byName[name] = &WorldResult{
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

		results := make([]WorldResult, 0, len(byName))
		for _, w := range byName {
			results = append(results, *w)
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
