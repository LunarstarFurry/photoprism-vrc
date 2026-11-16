package search

import (
	"strings"
	"unicode"
)

const VRCWorldPrefix = "VRChat - World: "

// ExtractVRCWorld extracts the raw VRChat world name from a formatted caption or notes string.
func ExtractVRCWorld(s string) string {
	idx := strings.Index(s, VRCWorldPrefix)
	if idx == -1 {
		return ""
	}

	rest := s[idx+len(VRCWorldPrefix):]

	for _, sep := range []string{" | ", " / ", "\n", "\r"} {
		if end := strings.Index(rest, sep); end != -1 {
			rest = rest[:end]
		}
	}

	return strings.TrimSpace(rest)
}

// NormalizeVRCWorld strips leading and trailing parenthetical/bracketed event or version tags
// (including fullwidth （...） / 【...】 used in native VRChat XMP metadata).
func NormalizeVRCWorld(name string) string {
	n := strings.TrimSpace(name)
	pairs := [][2]string{
		{"(", ")"},
		{"[", "]"},
		{"（", "）"},
		{"【", "】"},
	}
	for {
		trimmed := n
		for _, p := range pairs {
			open, close := p[0], p[1]
			if strings.HasSuffix(trimmed, close) {
				if idx := strings.LastIndex(trimmed, open); idx > 0 {
					trimmed = strings.TrimSpace(trimmed[:idx])
					break
				}
			} else if strings.HasPrefix(trimmed, open) {
				if idx := strings.Index(trimmed, close); idx != -1 && idx < len(trimmed)-len(close) {
					trimmed = strings.TrimSpace(trimmed[idx+len(close):])
					break
				}
			}
		}
		if trimmed == n || trimmed == "" {
			break
		}
		n = trimmed
	}
	return n
}

// CanonicalVRCWorldKey returns a lowercase alphanumeric key with trailing tags removed,
// so "Poly Dance Studio", "Polydance Studio", and "Polydance Studio (Event)" share a key.
func CanonicalVRCWorldKey(name string) string {
	norm := NormalizeVRCWorld(name)
	var b strings.Builder
	b.Grow(len(norm))
	for _, r := range strings.ToLower(norm) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func levenshtein(a, b []rune) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = min(del, min(ins, sub))
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

// MatchVRCWorld returns true if two VRChat world names are equivalent or similar enough
// to belong to the same world (ignoring casing, spaces, punctuation, trailing tags,
// prefix extensions, or minor typos).
func MatchVRCWorld(a, b string) bool {
	ka := CanonicalVRCWorldKey(a)
	kb := CanonicalVRCWorldKey(b)
	if ka == "" || kb == "" {
		return false
	}

	if ka == kb {
		return true
	}

	// Substring match for worlds that prepend or append extra text (e.g. "NEW! Polydance Studio" or "Polydance Studio v2").
	if len(ka) >= 7 && strings.Contains(kb, ka) {
		return true
	}
	if len(kb) >= 7 && strings.Contains(ka, kb) {
		return true
	}

	// Fuzzy edit-distance match for minor typos/punctuation/pluralization differences.
	ra, rb := []rune(ka), []rune(kb)
	maxLen := max(len(ra), len(rb))
	if maxLen >= 6 && absInt(len(ra)-len(rb)) <= 2 {
		dist := levenshtein(ra, rb)
		if dist <= 2 && float64(maxLen-dist)/float64(maxLen) >= 0.80 {
			return true
		}
	}

	return false
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
