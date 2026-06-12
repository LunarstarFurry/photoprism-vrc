package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/photoprism/photoprism/internal/ai/classify"
)

func TestPhoto_HasTitle(t *testing.T) {
	t.Run("False", func(t *testing.T) {
		m := PhotoFixtures.Get("Photo03")
		assert.False(t, m.HasTitle())
	})
	t.Run("True", func(t *testing.T) {
		m := PhotoFixtures.Get("Photo04")
		assert.True(t, m.HasTitle())
	})
}

func TestPhoto_NoTitle(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		m := PhotoFixtures.Get("Photo03")
		assert.True(t, m.NoTitle())
	})
	t.Run("False", func(t *testing.T) {
		m := PhotoFixtures.Get("Photo04")
		assert.False(t, m.NoTitle())
	})
}

func TestPhoto_SetTitle(t *testing.T) {
	t.Run("ManuallyDeleteTitle", func(t *testing.T) {
		// Photo15 has title source "name" (SrcName).
		m := PhotoFixtures.Get("Photo15")
		assert.Equal(t, "TitleToBeSet", m.PhotoTitle)
		// Manually delete existing title.
		m.SetTitle("", SrcManual)
		assert.Equal(t, "", m.PhotoTitle)
	})
	t.Run("LowerSourcePriority", func(t *testing.T) {
		// Photo15 has title source "name" (SrcName).
		m := PhotoFixtures.Get("Photo15")
		assert.Equal(t, "TitleToBeSet", m.PhotoTitle)
		// Set title with lower source priority.
		m.SetTitle("NewTitleSet", SrcAuto)
		assert.Equal(t, "TitleToBeSet", m.PhotoTitle)
	})
	t.Run("SameSourcePriority", func(t *testing.T) {
		// Photo15 has title source "name" (SrcName).
		m := PhotoFixtures.Get("Photo15")
		assert.Equal(t, "TitleToBeSet", m.PhotoTitle)
		// Try to delete existing title with same source priority.
		m.SetTitle("", SrcName)
		assert.Equal(t, "TitleToBeSet", m.PhotoTitle)
		// Replace existing title with same source priority.
		m.SetTitle("NewTitleSet", SrcName)
		assert.Equal(t, "NewTitleSet", m.PhotoTitle)
	})
}

func TestPhoto_GenerateTitle(t *testing.T) {
	t.Run("WonTUpdateTitleWasModified", func(t *testing.T) {
		m := PhotoFixtures.Get("Photo08")
		classifyLabels := &classify.Labels{}
		assert.Equal(t, "Black beach", m.PhotoTitle)
		err := m.GenerateTitle(*classifyLabels)
		if err == nil {
			t.Fatal()
		}
		assert.Equal(t, "Black beach", m.PhotoTitle)
	})
	t.Run("UseLocalTimeForTitle", func(t *testing.T) {
		m := PhotoFixtures.Get("Photo10")
		classifyLabels := &classify.Labels{}
		assert.Equal(t, "Title", m.PhotoTitle)
		
		// Reset title so it auto-generates
		m.SetTitle("", SrcManual)
		
		err := m.GenerateTitle(*classifyLabels)
		if err != nil {
			t.Fatal(err)
		}
		
		// It should equal the time string format
		assert.Equal(t, m.GetTakenAtLocal().Format("2006-01-02 15:04:05"), m.PhotoTitle)
	})
}

func TestPhoto_FileTitle(t *testing.T) {
	t.Run("NonLatin", func(t *testing.T) {
		photo := Photo{PhotoName: "桥", PhotoPath: "", OriginalName: ""}
		result := photo.FileTitle()
		assert.Equal(t, "桥", result)
	})
	t.Run("Slash", func(t *testing.T) {
		photo := Photo{PhotoName: "20200102_194030_9EFA9E5E", PhotoPath: "2000/05", OriginalName: "flickr import/changing-of-the-guard--buckingham-palace_7925318070_o.jpg"}
		result := photo.FileTitle()
		assert.Equal(t, "Changing of the Guard / Buckingham Palace", result)
	})
	t.Run("Empty", func(t *testing.T) {
		photo := Photo{PhotoName: "", PhotoPath: "", OriginalName: ""}
		result := photo.FileTitle()
		assert.Equal(t, "", result)
	})
	t.Run("Name", func(t *testing.T) {
		photo := Photo{PhotoName: "sun, beach, fun", PhotoPath: "", OriginalName: "", PhotoTitle: ""}
		result := photo.FileTitle()
		assert.Equal(t, "Sun, Beach, Fun", result)
	})
	t.Run("Path", func(t *testing.T) {
		photo := Photo{PhotoName: "", PhotoPath: "vacation", OriginalName: "20200102_194030_9EFA9E5E", PhotoTitle: ""}
		result := photo.FileTitle()
		assert.Equal(t, "Vacation", result)
	})
}

func TestPhoto_UpdateTitleLabels(t *testing.T) {
	FirstOrCreateLabel(NewLabel("Food", 1))
	FirstOrCreateLabel(NewLabel("Wine", 2))
	FirstOrCreateLabel(&Label{LabelName: "Bar", LabelSlug: "bar", CustomSlug: "bar", DeletedAt: TimeStamp()})

	t.Run("Success", func(t *testing.T) {
		details := &Details{Keywords: "snake, otter, food", KeywordsSrc: SrcMeta}
		photo := Photo{ID: 234567, PhotoTitle: "I was in a nice Wine Bar!", TitleSrc: SrcName, PhotoCaption: "cow, flower, food", CaptionSrc: SrcMeta, Details: details}

		if err := photo.Save(); err != nil {
			t.Fatal(err)
		}

		p := FindPhoto(photo)

		assert.Equal(t, 0, len(p.Labels))

		if err := p.UpdateTitleLabels(); err != nil {
			t.Fatal(err)
		}

		t.Logf("(1) %#v", p.Labels)

		p = FindPhoto(*p)

		t.Logf("(2) %#v", p.Labels)

		assert.Equal(t, "I was in a nice Wine Bar!", p.PhotoTitle)
		assert.Equal(t, "cow, flower, food", p.PhotoCaption)
		assert.Equal(t, "snake, otter, food", p.Details.Keywords)
		assert.Equal(t, 1, len(p.Labels))
	})
	t.Run("EmptyTitle", func(t *testing.T) {
		details := &Details{Keywords: "snake, otter, food", KeywordsSrc: SrcMeta}
		photo := Photo{ID: 234568, PhotoTitle: "", TitleSrc: SrcName, PhotoCaption: "cow, flower, food", CaptionSrc: SrcMeta, Details: details}

		if err := photo.Save(); err != nil {
			t.Fatal(err)
		}

		p := FindPhoto(photo)

		assert.Equal(t, 0, len(p.Labels))

		if err := p.UpdateTitleLabels(); err != nil {
			t.Fatal(err)
		}

		p = FindPhoto(*p)

		assert.Equal(t, "", p.PhotoTitle)
		assert.Equal(t, "cow, flower, food", p.PhotoCaption)
		assert.Equal(t, "snake, otter, food", p.Details.Keywords)
		assert.Equal(t, 0, len(p.Labels))
	})
}
