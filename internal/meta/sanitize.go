package meta

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/photoprism/photoprism/pkg/clean"
	"github.com/photoprism/photoprism/pkg/fs"
	"github.com/photoprism/photoprism/pkg/txt"
)

// UnwantedStrings lists boilerplate captions we strip during import.
var UnwantedStrings = map[string]bool{
	"Created by Imlib":         true, // Apps
	"iClarified":               true,
	"OLYMPUS DIGITAL CAMERA":   true, // Olympus
	"MEDION DIGITAL CAMERA":    true, // Medion
	"My beautiful picture":     true,
	"HELLOMOTO":                true, // Motorola
	"SAMSUNG":                  true, // Samsung
	"SAMSUNG CAMERA PICTURES":  true,
	"<Digimax i5, Samsung #1>": true,
	"SONY DSC":                 true, // Sony
	"Scanner":                  true, // KODAK Slide N Scan
	"rhdr":                     true, // Huawei
	"hdrpl":                    true,
	"oznorWO":                  true,
	"frontbhdp":                true,
	"fbt":                      true,
	"rbt":                      true,
	"ptr":                      true,
	"fbthdr":                   true,
	"btr":                      true,
	"mon":                      true,
	"nor":                      true,
	"dav":                      true,
	"mde":                      true,
	"mde_soft":                 true,
	"edf":                      true,
	"btfmdn":                   true,
	"btf":                      true,
	"btfhdr":                   true,
	"frem":                     true,
	"oznor":                    true,
	"rpt":                      true,
	"burst":                    true,
	"sdr_HDRB":                 true,
	"cof":                      true,
	"qrf":                      true,
	"fshbty":                   true,
	"_cuva":                    true,
	"smart":                    true,
	"hdr":                      true,
	"bsh":                      true,
	"binary comment":           true, // Other
	"default":                  true,
	"Exif_JPEG_PICTURE":        true,
	"DVC 10.1 HDMI":            true,
	"charset=Ascii":            true,
	"Digital Camera":           true,
}

// LowerCaseRegexp matches lower-case tokens in generated filenames.
var LowerCaseRegexp = regexp.MustCompile(`[a-z\d_\-]+`)

// SanitizeUnicode returns the string as valid Unicode with whitespace trimmed, and removes control
// characters other than tabs and line breaks, which are normalized to "\n".
func SanitizeUnicode(s string) string {
	if s == "" {
		return ""
	}

	return clean.Unicode(strings.TrimSpace(clean.Text(s)))
}

// SanitizeString removes unwanted character from an exif value string.
func SanitizeString(s string) string {
	if s == "" {
		return ""
	}

	if strings.HasPrefix(s, "string with binary data") {
		return ""
	} else if strings.HasPrefix(s, "(Binary data") {
		return ""
	}

	s = FormatVRCX(s)

	return SanitizeUnicode(strings.ReplaceAll(s, "\"", ""))
}

// SanitizeUID normalizes unique IDs found in XMP or Exif metadata.
func SanitizeUID(s string) string {
	s = SanitizeString(s)

	if len(s) < 15 {
		return ""
	}

	if start := strings.LastIndex(s, ":"); start != -1 {
		s = s[start+1:]
	}

	// Not a unique ID?
	if len(s) < 15 || len(s) > 36 {
		s = ""
	}

	return strings.ToLower(s)
}

// SanitizeTitle normalizes titles and removes unwanted information.
func SanitizeTitle(title string) string {
	s := SanitizeString(title)

	if s == "" {
		return ""
	}

	result := fs.StripKnownExt(s)

	if fs.IsGenerated(result) || txt.IsTime(result) {
		result = ""
	} else if result == s {
		// Do nothing.
	} else if found := LowerCaseRegexp.FindString(result); found != result {
		result = strings.ReplaceAll(strings.ReplaceAll(result, "_", " "), "  ", " ")
	} else if formatted := txt.FileTitle(s); formatted != "" {
		result = formatted
	} else {
		result = txt.Title(strings.ReplaceAll(result, "-", " "))
	}

	return result
}

var (
	vrcxWorldRegexp      = regexp.MustCompile(`"?world"?\s*:\s*\{[^}]*"?name"?\s*:\s*"?([^,"}]+)"?`)
	vrcxWorldIDRegexp    = regexp.MustCompile(`"?world"?\s*:\s*\{[^}]*"?id"?\s*:\s*"?(wrld_[^,"}\s]+)"?`)
	vrcxAuthorRegexp     = regexp.MustCompile(`"?author"?\s*:\s*\{[^}]*"?displayName"?\s*:\s*"?([^,"}]+)"?`)
	vrcxAuthorIDRegexp   = regexp.MustCompile(`"?author"?\s*:\s*\{[^}]*"?id"?\s*:\s*"?(usr_[^,"}\s]+)"?`)
	vrcxPlayersRegexp    = regexp.MustCompile(`"?players"?\s*:\s*\[(.*?)\]`)
	vrcxPlayerObjRegexp  = regexp.MustCompile(`\{([^}]*)\}`)
	vrcxPlayerIDRegexp   = regexp.MustCompile(`"?id"?\s*:\s*"?(usr_[^,"}\s]+)"?`)
	vrcxPlayerNameRegexp = regexp.MustCompile(`"?displayName"?\s*:\s*"?([^,"}]+)"?`)
)

// VRCPlayerMeta represents a VRChat player's immutable user ID and display name.
type VRCPlayerMeta struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ExtractVRCXMeta extracts the world ID and player ID/name pairs from raw VRCX metadata.
func ExtractVRCXMeta(s string) (worldID string, players []VRCPlayerMeta) {
	if !strings.Contains(s, "application:VRCX") && !strings.Contains(s, "\"application\":\"VRCX\"") && !strings.Contains(s, "\"application\": \"VRCX\"") {
		return "", nil
	}

	var raw struct {
		Author struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"author"`
		World struct {
			ID string `json:"id"`
		} `json:"world"`
		Players []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"players"`
	}

	seen := make(map[string]struct{})
	add := func(id, name string) {
		id = strings.TrimSpace(id)
		name = strings.TrimSpace(name)
		if id == "" && name == "" {
			return
		}
		key := id
		if key == "" {
			key = "name:" + strings.ToLower(name)
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		players = append(players, VRCPlayerMeta{ID: id, Name: name})
	}

	if err := json.Unmarshal([]byte(s), &raw); err == nil {
		worldID = strings.TrimSpace(raw.World.ID)
		add(raw.Author.ID, raw.Author.DisplayName)
		for _, p := range raw.Players {
			add(p.ID, p.DisplayName)
		}
		return worldID, players
	}

	// Fallback for unquoted ExifTool-formatted VRCX strings.
	if m := vrcxWorldIDRegexp.FindStringSubmatch(s); len(m) > 1 {
		worldID = strings.TrimSpace(m[1])
	}
	var authorID, authorName string
	if m := vrcxAuthorIDRegexp.FindStringSubmatch(s); len(m) > 1 {
		authorID = m[1]
	}
	if m := vrcxAuthorRegexp.FindStringSubmatch(s); len(m) > 1 {
		authorName = m[1]
	}
	add(authorID, authorName)

	if m := vrcxPlayersRegexp.FindStringSubmatch(s); len(m) > 1 {
		for _, obj := range vrcxPlayerObjRegexp.FindAllStringSubmatch(m[1], -1) {
			var pid, pname string
			if idm := vrcxPlayerIDRegexp.FindStringSubmatch(obj[1]); len(idm) > 1 {
				pid = idm[1]
			}
			if nm := vrcxPlayerNameRegexp.FindStringSubmatch(obj[1]); len(nm) > 1 {
				pname = nm[1]
			}
			add(pid, pname)
		}
	}

	return worldID, players
}

// EncodeVRCPlayersJSON merges an optional native VRChat author with VRCX players
// and encodes the deduplicated list as a compact JSON string.
func EncodeVRCPlayersJSON(players []VRCPlayerMeta, authorID, authorName string) string {
	seen := make(map[string]struct{}, len(players)+1)
	var merged []VRCPlayerMeta

	add := func(id, name string) {
		id = strings.TrimSpace(id)
		name = strings.TrimSpace(name)
		if id == "" && name == "" {
			return
		}
		key := id
		if key == "" {
			key = "name:" + strings.ToLower(name)
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		merged = append(merged, VRCPlayerMeta{ID: id, Name: name})
	}

	for _, p := range players {
		add(p.ID, p.Name)
	}
	if authorID != "" || authorName != "" {
		add(authorID, authorName)
	}

	if len(merged) == 0 {
		return ""
	}

	b, err := json.Marshal(merged)
	if err != nil {
		return ""
	}
	return string(b)
}

// FormatVRCX formats embedded VRChat metadata.
func FormatVRCX(s string) string {
	if !strings.Contains(s, "application:VRCX") && !strings.Contains(s, "\"application\":\"VRCX\"") {
		return s
	}

	var parts []string

	// Extract world name
	if m := vrcxWorldRegexp.FindStringSubmatch(s); len(m) > 1 {
		parts = append(parts, "World: "+m[1])
	}

	// Extract author
	if m := vrcxAuthorRegexp.FindStringSubmatch(s); len(m) > 1 {
		parts = append(parts, "Author: "+m[1])
	}

	// Extract players
	if m := vrcxPlayersRegexp.FindStringSubmatch(s); len(m) > 1 {
		playersStr := m[1]
		matches := vrcxPlayerNameRegexp.FindAllStringSubmatch(playersStr, -1)
		var players []string
		for _, match := range matches {
			players = append(players, match[1])
		}
		if len(players) > 0 {
			parts = append(parts, "Players: "+strings.Join(players, ", "))
		}
	}

	if len(parts) > 0 {
		return "VRChat - " + strings.Join(parts, " | ")
	}

	return s
}

// SanitizeCaption normalizes metadata captions and removes unwanted text.
func SanitizeCaption(s string) string {
	s = SanitizeString(s)

	switch {
	case s == "":
		return ""
	case UnwantedStrings[s]:
		return ""
	case strings.HasPrefix(s, "DCIM\\") && !strings.Contains(s, " "):
		return ""
	default:
		return s
	}
}

// SanitizeMeta normalizes metadata fields that may contain JSON arrays like keywords and subject.
func SanitizeMeta(s string) string {
	if s == "" {
		return ""
	}

	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		var words []string

		if err := json.Unmarshal([]byte(s), &words); err != nil {
			return SanitizeUnicode(s)
		}

		s = SanitizeUnicode(strings.Join(words, ", "))
	} else {
		s = SanitizeString(s)
	}

	return s
}
