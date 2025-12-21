package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/photoprism/photoprism/internal/entity/search"
)

func TestVideoNormalizeFilter(t *testing.T) {
	t.Run("NormalizeTokens", func(t *testing.T) {
		args := []string{"foo", "2024/clip.mp4", "name:bar", "filename:2025/a.mov", ""}
		expected := "name:foo filename:2024/clip.mp4 name:bar filename:2025/a.mov"
		assert.Equal(t, expected, videoNormalizeFilter(args))
	})
}

func TestVideoParseTrimDuration(t *testing.T) {
	t.Run("Seconds", func(t *testing.T) {
		d, err := videoParseTrimDuration("5")
		assert.NoError(t, err)
		assert.Equal(t, 5*time.Second, d)
	})

	t.Run("NegativeSeconds", func(t *testing.T) {
		d, err := videoParseTrimDuration("-10")
		assert.NoError(t, err)
		assert.Equal(t, -10*time.Second, d)
	})

	t.Run("MinutesSeconds", func(t *testing.T) {
		d, err := videoParseTrimDuration("02:05")
		assert.NoError(t, err)
		assert.Equal(t, 2*time.Minute+5*time.Second, d)
	})

	t.Run("HoursMinutesSeconds", func(t *testing.T) {
		d, err := videoParseTrimDuration("01:02:03")
		assert.NoError(t, err)
		assert.Equal(t, time.Hour+2*time.Minute+3*time.Second, d)
	})

	t.Run("GoDuration", func(t *testing.T) {
		d, err := videoParseTrimDuration("2m5s")
		assert.NoError(t, err)
		assert.Equal(t, 2*time.Minute+5*time.Second, d)
	})

	t.Run("Invalid", func(t *testing.T) {
		_, err := videoParseTrimDuration("1:30s")
		assert.Error(t, err)
	})
}

func TestVideoListJSONRow(t *testing.T) {
	t.Run("NumericFields", func(t *testing.T) {
		found := search.Photo{
			FileName:     "clip.mp4",
			FileRoot:     "/",
			FileDuration: 2 * time.Second,
			FileCodec:    "avc1",
			FileMime:     "video/mp4",
			FileWidth:    1920,
			FileHeight:   1080,
			FileFPS:      29.97,
			FileFrames:   120,
			FileSize:     1234,
			FileHash:     "abc",
			FileSidecar:  true,
		}

		row := videoListJSONRow(found, true)
		assert.Equal(t, int64(2*time.Second), row["duration"])
		assert.Equal(t, int64(1234), row["size"])
		assert.Equal(t, true, row["sidecar"])
	})
}
