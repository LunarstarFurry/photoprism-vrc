package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/photoprism/photoprism/internal/config"
	"github.com/photoprism/photoprism/internal/entity/search"
	"github.com/photoprism/photoprism/internal/meta"
	"github.com/photoprism/photoprism/internal/photoprism"
	"github.com/photoprism/photoprism/pkg/clean"
)

// VideoInfoCommand configures the command name, flags, and action.
var VideoInfoCommand = &cli.Command{
	Name:      "info",
	Usage:     "Displays diagnostic information for indexed videos",
	ArgsUsage: "[filter]...",
	Flags: []cli.Flag{
		videoCountFlag,
		OffsetFlag,
		JsonFlag(),
		videoVerboseFlag,
	},
	Action: videoInfoAction,
}

// videoInfoAction prints indexed, ExifTool, and ffprobe metadata for matching videos.
func videoInfoAction(ctx *cli.Context) error {
	return CallWithDependencies(ctx, func(conf *config.Config) error {
		filter := videoNormalizeFilter(ctx.Args().Slice())
		results, err := videoSearchResults(filter, ctx.Uint(videoCountFlag.Name), ctx.Int(OffsetFlag.Name), false)
		if err != nil {
			return err
		}

		entries := make([]videoInfoEntry, 0, len(results))
		for _, found := range results {
			entry, err := videoInfoEntryFor(conf, found, ctx.Bool(videoVerboseFlag.Name))
			if err != nil {
				log.Warnf("info: %s", clean.Error(err))
			}
			entries = append(entries, entry)
		}

		if ctx.Bool("json") {
			payload, err := json.Marshal(entries)
			if err != nil {
				return err
			}
			fmt.Println(string(payload))
			return nil
		}

		for _, entry := range entries {
			videoPrintInfo(entry)
		}

		return nil
	})
}

// videoInfoEntry describes all metadata sections for a single video.
type videoInfoEntry struct {
	Index   map[string]interface{} `json:"index"`
	Exif    interface{}            `json:"exif,omitempty"`
	FFprobe interface{}            `json:"ffprobe,omitempty"`
	Raw     map[string]string      `json:"raw,omitempty"`
}

// videoInfoEntryFor collects indexed, ExifTool, and ffprobe metadata for a search result.
func videoInfoEntryFor(conf *config.Config, found search.Photo, verbose bool) (videoInfoEntry, error) {
	entry := videoInfoEntry{
		Index: videoIndexSummary(found),
	}

	filePath := photoprism.FileName(found.FileRoot, found.FileName)
	mediaFile, err := photoprism.NewMediaFile(filePath)
	if err != nil {
		return entry, err
	}

	if conf.DisableExifTool() {
		entry.Exif = nil
	} else {
		exif := mediaFile.MetaData()
		entry.Exif = exif
		if verbose {
			entry.ensureRaw()
			entry.Raw["exif"] = videoPrettyJSON(exif)
		}
	}

	ffprobeBin := conf.FFprobeBin()
	if ffprobeBin == "" {
		entry.FFprobe = nil
	} else if ffprobe, raw, err := videoRunFFprobe(ffprobeBin, filePath); err != nil {
		entry.FFprobe = nil
		if verbose {
			entry.ensureRaw()
			entry.Raw["ffprobe"] = raw
		}
	} else {
		entry.FFprobe = ffprobe
		if verbose {
			entry.ensureRaw()
			entry.Raw["ffprobe"] = raw
		}
	}

	return entry, nil
}

// videoIndexSummary builds a concise map of indexed fields for diagnostics.
func videoIndexSummary(found search.Photo) map[string]interface{} {
	return map[string]interface{}{
		"file_name":       found.FileName,
		"file_root":       found.FileRoot,
		"file_uid":        found.FileUID,
		"photo_uid":       found.PhotoUID,
		"media_type":      found.MediaType,
		"file_type":       found.FileType,
		"file_mime":       found.FileMime,
		"file_codec":      found.FileCodec,
		"file_hash":       found.FileHash,
		"file_size":       found.FileSize,
		"file_duration":   found.FileDuration.Nanoseconds(),
		"photo_duration":  found.PhotoDuration.Nanoseconds(),
		"file_frames":     found.FileFrames,
		"file_fps":        found.FileFPS,
		"file_width":      found.FileWidth,
		"file_height":     found.FileHeight,
		"file_sidecar":    found.FileSidecar,
		"file_missing":    found.FileMissing,
		"file_video":      found.FileVideo,
		"original_name":   found.OriginalName,
		"instance_id":     found.InstanceID,
		"photo_taken_at":  found.TakenAt,
		"photo_taken_src": found.TakenSrc,
	}
}

// videoRunFFprobe executes ffprobe and returns parsed JSON plus raw output.
func videoRunFFprobe(ffprobeBin, filePath string) (interface{}, string, error) {
	cmd := exec.Command(ffprobeBin, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", filePath) //nolint:gosec // args are validated paths

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, strings.TrimSpace(stdout.String()), fmt.Errorf("ffprobe failed: %s", strings.TrimSpace(stderr.String()))
	}

	raw := strings.TrimSpace(stdout.String())
	if raw == "" {
		return nil, raw, nil
	}

	var data interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, raw, nil
	}

	return data, raw, nil
}

// ensureRaw initializes the raw map for verbose output.
func (v *videoInfoEntry) ensureRaw() {
	if v.Raw == nil {
		v.Raw = make(map[string]string)
	}
}

// videoPrettyJSON returns indented JSON for human-readable output.
func videoPrettyJSON(value interface{}) string {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}

	return string(data)
}

// videoPrintInfo prints a human-readable metadata summary to stdout.
func videoPrintInfo(entry videoInfoEntry) {
	fmt.Println("Indexed Metadata:")
	fmt.Println(videoPrettyJSON(entry.Index))

	if entry.Exif == nil {
		fmt.Println("ExifTool Metadata: disabled or unavailable")
	} else if exifMap, ok := entry.Exif.(meta.Data); ok {
		fmt.Println("ExifTool Metadata:")
		fmt.Println(videoPrettyJSON(exifMap))
	} else {
		fmt.Println("ExifTool Metadata:")
		fmt.Println(videoPrettyJSON(entry.Exif))
	}

	if entry.FFprobe == nil {
		fmt.Println("FFprobe Diagnostics: unavailable")
	} else {
		fmt.Println("FFprobe Diagnostics:")
		fmt.Println(videoPrettyJSON(entry.FFprobe))
	}

	if len(entry.Raw) > 0 {
		fmt.Println("Raw Metadata:")
		fmt.Println(videoPrettyJSON(entry.Raw))
	}
}
