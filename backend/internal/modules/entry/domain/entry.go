package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type Entry struct {
	ID          uuid.UUID
	SourceID    uuid.UUID
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ReadAt      *time.Time
	SavedAt     *time.Time
}

type EntryMetadata struct {
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

func NewEntry(sourceID uuid.UUID, meta EntryMetadata) (*Entry, error) {
	title, link, description, err := normalizeEntryMetadata(meta)
	if err != nil {
		return nil, err
	}
	if sourceID == uuid.Nil {
		return nil, apperr.InvalidInput.New("source id is required")
	}
	now := time.Now().UTC()
	return &Entry{
		ID:          uuid.New(),
		SourceID:    sourceID,
		Title:       title,
		Link:        link,
		Description: description,
		Thumbnail:   cloneStringPtr(meta.Thumbnail),
		PublishedAt: cloneTimePtr(meta.PublishedAt),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (e *Entry) SortAt() time.Time {
	if e.PublishedAt != nil {
		return e.PublishedAt.UTC()
	}
	return e.CreatedAt.UTC()
}

func (e *Entry) RefreshMetadata(meta EntryMetadata) error {
	title, _, description, err := normalizeEntryMetadata(meta)
	if err != nil {
		return err
	}
	// Link is identity (Source+link); refresh only mutable metadata.
	e.Title = title
	e.Description = description
	e.Thumbnail = cloneStringPtr(meta.Thumbnail)
	e.PublishedAt = cloneTimePtr(meta.PublishedAt)
	e.UpdatedAt = time.Now().UTC()
	return nil
}

func (e *Entry) MarkRead(at time.Time) {
	t := at.UTC()
	e.ReadAt = &t
	e.UpdatedAt = time.Now().UTC()
}

func (e *Entry) ClearRead() {
	e.ReadAt = nil
	e.UpdatedAt = time.Now().UTC()
}

func (e *Entry) MarkSaved(at time.Time) {
	t := at.UTC()
	e.SavedAt = &t
	e.UpdatedAt = time.Now().UTC()
}

func (e *Entry) ClearSaved() {
	e.SavedAt = nil
	e.UpdatedAt = time.Now().UTC()
}

func normalizeEntryMetadata(meta EntryMetadata) (title, link, description string, err error) {
	title = strings.TrimSpace(meta.Title)
	link = strings.TrimSpace(meta.Link)
	if title == "" {
		return "", "", "", apperr.InvalidInput.New("title is required")
	}
	if link == "" {
		return "", "", "", apperr.InvalidInput.New("link is required")
	}
	return title, link, strings.TrimSpace(meta.Description), nil
}

func cloneStringPtr(v *string) *string {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func cloneTimePtr(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	cp := v.UTC()
	return &cp
}
