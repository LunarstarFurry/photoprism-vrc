package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/photoprism/photoprism/internal/entity/search"
	"github.com/photoprism/photoprism/pkg/txt/report"
)

// videoNormalizeFilter converts CLI args into a search query, mapping bare tokens to name/filename filters.
func videoNormalizeFilter(args []string) string {
	parts := make([]string, 0, len(args))

	for _, arg := range args {
		token := strings.TrimSpace(arg)
		if token == "" {
			continue
		}

		if strings.Contains(token, ":") {
			parts = append(parts, token)
			continue
		}

		if strings.Contains(token, "/") {
			parts = append(parts, fmt.Sprintf("filename:%s", token))
		} else {
			parts = append(parts, fmt.Sprintf("name:%s", token))
		}
	}

	return strings.TrimSpace(strings.Join(parts, " "))
}

// videoSplitTrimArgs separates filter args from the trailing trim duration argument.
func videoSplitTrimArgs(args []string) ([]string, string, error) {
	if len(args) == 0 {
		return nil, "", fmt.Errorf("missing duration argument")
	}

	filterArgs := make([]string, len(args)-1)
	copy(filterArgs, args[:len(args)-1])

	durationArg := strings.TrimSpace(args[len(args)-1])
	if durationArg == "" {
		return nil, "", fmt.Errorf("missing duration argument")
	}

	return filterArgs, durationArg, nil
}

// videoParseTrimDuration parses the trim duration string with the precedence and rules from the spec.
func videoParseTrimDuration(value string) (time.Duration, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return 0, fmt.Errorf("duration is empty")
	}

	sign := 1
	if strings.HasPrefix(raw, "-") {
		sign = -1
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "-"))
	}

	if raw == "" {
		return 0, fmt.Errorf("duration is empty")
	}

	if isDigits(raw) {
		secs, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", value)
		}
		if secs == 0 {
			return 0, fmt.Errorf("duration must be non-zero")
		}
		return time.Duration(sign) * time.Duration(secs) * time.Second, nil
	}

	if strings.Contains(raw, ":") {
		if strings.ContainsAny(raw, "hms") {
			return 0, fmt.Errorf("invalid duration %q", value)
		}

		parts := strings.Split(raw, ":")
		if len(parts) != 2 && len(parts) != 3 {
			return 0, fmt.Errorf("invalid duration %q", value)
		}

		for _, p := range parts {
			if !isDigits(p) {
				return 0, fmt.Errorf("invalid duration %q", value)
			}
		}

		if len(parts) == 2 && len(parts[1]) != 2 {
			return 0, fmt.Errorf("invalid duration %q", value)
		}

		if len(parts) == 3 && (len(parts[1]) != 2 || len(parts[2]) != 2) {
			return 0, fmt.Errorf("invalid duration %q", value)
		}

		var hours, minutes, seconds int64

		if len(parts) == 2 {
			minutes, _ = strconv.ParseInt(parts[0], 10, 64)
			seconds, _ = strconv.ParseInt(parts[1], 10, 64)
		} else {
			hours, _ = strconv.ParseInt(parts[0], 10, 64)
			minutes, _ = strconv.ParseInt(parts[1], 10, 64)
			seconds, _ = strconv.ParseInt(parts[2], 10, 64)
		}

		total := time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
		if total == 0 {
			return 0, fmt.Errorf("duration must be non-zero")
		}

		return time.Duration(sign) * total, nil
	}

	parsed, err := time.ParseDuration(applySign(raw, sign))
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", value)
	}

	if parsed == 0 {
		return 0, fmt.Errorf("duration must be non-zero")
	}

	return parsed, nil
}

// videoListColumns returns the ordered column list for the video ls output.
func videoListColumns(includeSidecar bool) []string {
	cols := []string{"Name", "Root"}
	if includeSidecar {
		cols = append(cols, "Sidecar")
	}
	return append(cols, "Duration", "Codec", "Mime", "Width", "Height", "FPS", "Frames", "Size", "Hash")
}

// videoListRow renders a search result row for table outputs with human-friendly values.
func videoListRow(found search.Photo, includeSidecar bool) []string {
	row := []string{found.FileName, found.FileRoot}
	if includeSidecar {
		row = append(row, strconv.FormatBool(found.FileSidecar))
	}

	row = append(row,
		videoHumanDuration(found.FileDuration),
		found.FileCodec,
		found.FileMime,
		videoHumanInt(found.FileWidth),
		videoHumanInt(found.FileHeight),
		videoHumanFloat(found.FileFPS),
		videoHumanInt(found.FileFrames),
		videoHumanSize(found.FileSize),
		found.FileHash,
	)

	return row
}

// videoListJSONRow renders a search result row for JSON output with raw numeric values.
func videoListJSONRow(found search.Photo, includeSidecar bool) map[string]interface{} {
	data := map[string]interface{}{
		"name":     found.FileName,
		"root":     found.FileRoot,
		"duration": found.FileDuration.Nanoseconds(),
		"codec":    found.FileCodec,
		"mime":     found.FileMime,
		"width":    found.FileWidth,
		"height":   found.FileHeight,
		"fps":      found.FileFPS,
		"frames":   found.FileFrames,
		"size":     videoNonNegativeSize(found.FileSize),
		"hash":     found.FileHash,
	}

	if includeSidecar {
		data["sidecar"] = found.FileSidecar
	}

	return data
}

// videoListJSON marshals a list of JSON rows using the canonical keys for each column.
func videoListJSON(rows []map[string]interface{}, cols []string) (string, error) {
	canon := make([]string, len(cols))
	for i, col := range cols {
		canon[i] = report.CanonKey(col)
	}

	payload := make([]map[string]interface{}, 0, len(rows))

	for _, row := range rows {
		item := make(map[string]interface{}, len(canon))
		for _, key := range canon {
			item[key] = row[key]
		}
		payload = append(payload, item)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// videoHumanDuration formats a duration for human-readable tables.
func videoHumanDuration(d time.Duration) string {
	if d <= 0 {
		return ""
	}

	return d.String()
}

// videoHumanInt formats non-zero integers for human-readable tables.
func videoHumanInt(value int) string {
	if value <= 0 {
		return ""
	}

	return strconv.Itoa(value)
}

// videoHumanFloat formats non-zero floats without unnecessary trailing zeros.
func videoHumanFloat(value float64) string {
	if value <= 0 {
		return ""
	}

	return strconv.FormatFloat(value, 'f', -1, 64)
}

// videoHumanSize formats file sizes with human-readable units.
func videoHumanSize(size int64) string {
	return humanize.Bytes(uint64(videoNonNegativeSize(size))) //nolint:gosec // size is bounded to non-negative values
}

// videoNonNegativeSize clamps negative sizes to zero before formatting.
func videoNonNegativeSize(size int64) int64 {
	if size < 0 {
		return 0
	}

	return size
}

// videoTempPath creates a temporary file path in the destination directory.
func videoTempPath(dir, pattern string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("temp directory is empty")
	}

	tmpFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}

	if err = tmpFile.Close(); err != nil {
		return "", err
	}

	if err = os.Remove(tmpFile.Name()); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

// videoFFmpegSeconds converts a duration into an ffmpeg-friendly seconds string.
func videoFFmpegSeconds(d time.Duration) string {
	seconds := d.Seconds()
	return strconv.FormatFloat(seconds, 'f', 3, 64)
}

// isDigits reports whether the string contains only decimal digits.
func isDigits(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// applySign applies a numeric sign to a duration string for parsing.
func applySign(value string, sign int) string {
	if sign >= 0 {
		return value
	}

	return "-" + value
}

// videoSidecarPath builds the sidecar destination path for an originals file without creating directories.
func videoSidecarPath(srcName, originalsPath, sidecarPath string) string {
	src := filepath.ToSlash(srcName)
	orig := filepath.ToSlash(originalsPath)

	if orig != "" {
		orig = strings.TrimSuffix(orig, "/") + "/"
	}

	rel := strings.TrimPrefix(src, orig)
	if rel == src {
		rel = filepath.Base(srcName)
	}

	rel = strings.TrimPrefix(rel, "/")
	return filepath.Join(sidecarPath, filepath.FromSlash(rel))
}
