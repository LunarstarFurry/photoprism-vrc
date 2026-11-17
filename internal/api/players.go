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
	PlayerID   string    `json:"PlayerID,omitempty"`
	Name       string    `json:"Name"`
	PhotoCount int       `json:"PhotoCount"`
	Thumb      string    `json:"Thumb"`
	TakenAt    time.Time `json:"TakenAt"`
}

type playerPhotoRow struct {
	VRCPlayers   string
	PhotoCaption string
	Notes        string
	FileHash     string
	TakenAt      time.Time
}

// GetPlayers finds and returns VRChat players extracted from photo metadata as JSON.
// Groups by usr_... ID when available, falling back to display-name matching for older photos.
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
			Select("COALESCE(details.vrc_players, '') AS vrc_players, photos.photo_caption, COALESCE(details.notes, '') AS notes, files.file_hash, photos.taken_at").
			Joins("JOIN files ON files.photo_id = photos.id AND files.file_primary = 1 AND files.deleted_at IS NULL").
			Joins("LEFT JOIN details ON details.photo_id = photos.id").
			Where("photos.deleted_at IS NULL").
			Where("photos.photo_caption LIKE ? OR details.notes LIKE ? OR COALESCE(details.vrc_players, '') <> ''", "%VRChat - %", "%VRChat - %").
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
			result     PlayerResult
			thumbs     []string
			knownNames map[string]struct{}
		}

		var aggs []*playerAgg
		byID := make(map[string]*playerAgg)
		byName := make(map[string]*playerAgg)

		for _, row := range rows {
			entries := search.ExtractVRCPlayerEntries(row.VRCPlayers, row.PhotoCaption, row.Notes)
			seenInPhoto := make(map[*playerAgg]struct{}, len(entries))

			for _, entry := range entries {
				id := strings.TrimSpace(entry.ID)
				name := strings.TrimSpace(entry.Name)
				nameKey := strings.ToLower(name)

				var target *playerAgg
				if id != "" {
					target = byID[id]
				}
				if target == nil && nameKey != "" {
					if candidate, ok := byName[nameKey]; ok {
						if id == "" || candidate.result.PlayerID == "" || candidate.result.PlayerID == id {
							target = candidate
						}
					}
				}

				if target == nil {
					displayName := name
					if displayName == "" {
						displayName = id
					}
					target = &playerAgg{
						result: PlayerResult{
							PlayerID:   id,
							Name:       displayName,
							PhotoCount: 0,
							TakenAt:    row.TakenAt,
						},
						knownNames: make(map[string]struct{}),
					}
					aggs = append(aggs, target)
				}

				if id != "" {
					if target.result.PlayerID == "" {
						target.result.PlayerID = id
					}
					byID[id] = target
				}
				if nameKey != "" {
					target.knownNames[nameKey] = struct{}{}
					if _, exists := byName[nameKey]; !exists {
						byName[nameKey] = target
					}
				}

				if _, alreadyCounted := seenInPhoto[target]; alreadyCounted {
					continue
				}
				seenInPhoto[target] = struct{}{}

				target.result.PhotoCount++
				if row.FileHash != "" {
					target.thumbs = append(target.thumbs, row.FileHash)
				}
			}
		}

		filtered := make([]*playerAgg, 0, len(aggs))
		for _, agg := range aggs {
			if q != "" {
				matched := strings.Contains(strings.ToLower(agg.result.Name), q) ||
					strings.Contains(strings.ToLower(agg.result.PlayerID), q)
				if !matched {
					for kn := range agg.knownNames {
						if strings.Contains(kn, q) {
							matched = true
							break
						}
					}
				}
				if !matched {
					continue
				}
			} else if !showAll && agg.result.PhotoCount < 2 {
				continue
			}
			filtered = append(filtered, agg)
		}

		sort.Slice(filtered, func(i, j int) bool {
			if filtered[i].result.PhotoCount != filtered[j].result.PhotoCount {
				return filtered[i].result.PhotoCount > filtered[j].result.PhotoCount
			}
			return strings.ToLower(filtered[i].result.Name) < strings.ToLower(filtered[j].result.Name)
		})

		// Pick a distinct, deterministic cover thumbnail per player.
		usedThumbs := make(map[string]bool, len(filtered))
		results := make([]PlayerResult, 0, len(filtered))
		for _, agg := range filtered {
			if n := len(agg.thumbs); n > 0 {
				seed := agg.result.PlayerID
				if seed == "" {
					seed = strings.ToLower(agg.result.Name)
				}
				h := fnv.New32a()
				_, _ = h.Write([]byte(seed))
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
