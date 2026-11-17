package search

import (
	"encoding/json"
	"strings"
)

// VRCPlayerEntry represents a VRChat player's optional user ID and display name.
type VRCPlayerEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const (
	vrcAuthorTag  = "Author: "
	vrcPlayersTag = "Players: "
)

func extractVRCSegment(s, tag string) string {
	idx := strings.Index(s, tag)
	if idx == -1 {
		return ""
	}

	rest := s[idx+len(tag):]
	for _, sep := range []string{" | ", " / ", "\n", "\r"} {
		if end := strings.Index(rest, sep); end != -1 {
			rest = rest[:end]
		}
	}

	return strings.TrimSpace(rest)
}

func isUnresolvedVRCUserID(name string) bool {
	return strings.HasPrefix(name, "usr_") && len(name) > 30
}

// ExtractVRCPlayers extracts a deduplicated list of VRChat player display names
// from a photo's Caption and Notes (covering both Author and Players).
func ExtractVRCPlayers(caption, notes string) []string {
	seen := make(map[string]struct{})
	var players []string

	addPlayer := func(raw string) {
		name := strings.TrimSpace(raw)
		if name == "" || isUnresolvedVRCUserID(name) {
			return
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		players = append(players, name)
	}

	for _, src := range []string{caption, notes} {
		if !strings.Contains(src, "VRChat - ") {
			continue
		}

		if author := extractVRCSegment(src, vrcAuthorTag); author != "" {
			addPlayer(author)
		}

		if list := extractVRCSegment(src, vrcPlayersTag); list != "" {
			for _, p := range strings.Split(list, ", ") {
				addPlayer(p)
			}
		}
	}

	return players
}

// MatchVRCPlayer returns true if query matches any player in the slice (case-insensitive exact match).
func MatchVRCPlayer(players []string, query string) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return false
	}

	for _, p := range players {
		if strings.EqualFold(p, q) {
			return true
		}
	}

	return false
}

// ExtractVRCPlayerEntries returns deduplicated player entries from stored VRCPlayers JSON
// when available, falling back to parsing display names from Caption and Notes for older photos.
func ExtractVRCPlayerEntries(vrcPlayersJSON, caption, notes string) []VRCPlayerEntry {
	if strings.TrimSpace(vrcPlayersJSON) != "" {
		var parsed []VRCPlayerEntry
		if err := json.Unmarshal([]byte(vrcPlayersJSON), &parsed); err == nil && len(parsed) > 0 {
			seen := make(map[string]struct{}, len(parsed))
			out := make([]VRCPlayerEntry, 0, len(parsed))
			for _, p := range parsed {
				id := strings.TrimSpace(p.ID)
				name := strings.TrimSpace(p.Name)
				if name == "" && isUnresolvedVRCUserID(id) {
					continue
				}
				if name == "" && id == "" {
					continue
				}
				key := id
				if key == "" {
					key = "name:" + strings.ToLower(name)
				}
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				out = append(out, VRCPlayerEntry{ID: id, Name: name})
			}
			if len(out) > 0 {
				return out
			}
		}
	}

	names := ExtractVRCPlayers(caption, notes)
	out := make([]VRCPlayerEntry, 0, len(names))
	for _, n := range names {
		out = append(out, VRCPlayerEntry{Name: n})
	}
	return out
}
