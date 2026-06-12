package entity

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize/english"

	"github.com/photoprism/photoprism/internal/ai/classify"
	"github.com/photoprism/photoprism/pkg/clean"
	"github.com/photoprism/photoprism/pkg/fs"
	"github.com/photoprism/photoprism/pkg/txt"
)

// HasTitle checks if the photo has a title.
func (m *Photo) HasTitle() bool {
	return m.PhotoTitle != ""
}

// NoTitle reports whether the photo has no title.
func (m *Photo) NoTitle() bool {
	return m.PhotoTitle == ""
}

// GetTitle returns the photo title, if any.
func (m *Photo) GetTitle() string {
	return m.PhotoTitle
}

// SetTitle updates the photo title when the supplied source outranks the current one.
// The title is normalized, quotes are unified, and the final value is clipped to 300 characters.
func (m *Photo) SetTitle(title, source string) {
	title = strings.Trim(title, "_&|{}<>: \n\r\t\\")
	title = strings.ReplaceAll(title, "\"", "'")
	title = txt.Shorten(title, txt.ClipLongName, txt.Ellipsis)

	// Get source priority.
	p := SrcPriority[source]

	// Compare the source priority with the priority of the current title source.
	// Ignore requests from lower ranked sources so manual and trusted titles stay in place.
	if (p < SrcPriority[m.TitleSrc]) && m.HasTitle() {
		return
	}

	// Allow users to manually delete existing titles.
	if title == "" && p != 1 && p < SrcPriority[SrcManual] {
		return
	}

	m.PhotoTitle = title
	m.TitleSrc = source
}

// GenerateTitle derives an automatic title using location, labels, and subject metadata
// when the current title source allows auto-generation.
func (m *Photo) GenerateTitle(labels classify.Labels) error {
	if m.TitleSrc != SrcAuto {
		return fmt.Errorf("photo: %s keeps existing %s title", m.String(), SrcString(m.TitleSrc))
	}

	start := time.Now()
	oldTitle := m.PhotoTitle

	// Find people in the picture to generate caption.
	m.GenerateCaption(m.SubjectNames())

	if !m.GetTakenAtLocal().IsZero() {
		m.SetTitle(m.GetTakenAtLocal().Format("2006-01-02 15:04:05"), SrcAuto)
	} else {
		m.SetTitle(UnknownTitle, SrcAuto)
	}

	// Log changes for debugging and auditing.
	if m.PhotoTitle != oldTitle {
		log.Debugf("photo: %s has new title %s [%s]", m.String(), clean.Log(m.PhotoTitle), time.Since(start))
	}

	return nil
}

// GenerateAndSaveTitle updates the photo title and saves it.
func (m *Photo) GenerateAndSaveTitle() error {
	if !m.HasID() {
		return fmt.Errorf("photo id is missing")
	}

	m.PhotoFaces = m.FaceCount()

	labels := m.ClassifyLabels()

	m.UpdateDateFields()

	if err := m.GenerateTitle(labels); err != nil {
		log.Info(err)
	}

	if err := m.IndexKeywords(); err != nil {
		log.Errorf("photo: %s", err.Error())
	}

	if err := m.Save(); err != nil {
		return err
	}

	return nil
}

// FileTitle returns a photo title based on the file name and/or path.
func (m *Photo) FileTitle() string {
	// Generate a title from the photo name when the name was not generated automatically.
	if !fs.IsGenerated(m.PhotoName) {
		if title := txt.FileTitle(m.PhotoName); title != "" {
			return title
		}
	}

	// Generate a title from the original file name, if available.
	if m.OriginalName != "" {
		if title := txt.FileTitle(m.OriginalName); !fs.IsGenerated(m.OriginalName) && title != "" {
			return title
		} else if title := txt.FileTitle(filepath.Dir(m.OriginalName)); title != "" {
			return title
		}
	}

	// Fall back to the photo path when no other title could be inferred.
	if m.PhotoPath != "" && !fs.IsGenerated(m.PhotoPath) {
		return txt.FileTitle(m.PhotoPath)
	}

	return ""
}

// UpdateTitleLabels updates the labels assigned based on the photo title.
func (m *Photo) UpdateTitleLabels() error {
	if m == nil {
		return nil
	} else if m.PhotoTitle == "" {
		return nil
	} else if SrcPriority[m.TitleSrc] < SrcPriority[SrcName] {
		return nil
	}

	keywords := txt.UniqueKeywords(m.PhotoTitle)

	var labelIds []uint

	for _, w := range keywords {
		if label, err := FindLabel(w, true); err == nil {
			if label.Skip() {
				continue
			}

			labelIds = append(labelIds, label.ID)
			FirstOrCreatePhotoLabel(NewPhotoLabel(m.ID, label.ID, 10, classify.SrcTitle))
		}
	}

	// Remove stale title-based labels so the photo reflects the current title.
	return Db().Where("label_src = ? AND photo_id = ? AND label_id NOT IN (?)", classify.SrcTitle, m.ID, labelIds).Delete(&PhotoLabel{}).Error
}
