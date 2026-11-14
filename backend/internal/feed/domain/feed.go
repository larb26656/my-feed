package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusPublished Status = "PUBLISHED"
	StatusArchived  Status = "ARCHIVED"
)

type Feed struct {
	ID          uuid.UUID
	Slug        string
	Title       string
	Description string
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NewFeed(slug, title, description string) (*Feed, error) {
	slug = strings.TrimSpace(slug)
	title = strings.TrimSpace(title)
	if slug == "" {
		return nil, apperr.Validation("slug is required")
	}
	if title == "" {
		return nil, apperr.Validation("title is required")
	}
	now := time.Now().UTC()
	return &Feed{
		ID:          uuid.New(),
		Slug:        slug,
		Title:       title,
		Description: strings.TrimSpace(description),
		Status:      StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func (f *Feed) Publish() error {
	if f.Status != StatusDraft {
		return apperr.FailedPrecondition("only DRAFT feeds can be published")
	}
	f.Status = StatusPublished
	f.UpdatedAt = time.Now().UTC()
	return nil
}

func (f *Feed) Archive() error {
	if f.Status != StatusPublished {
		return apperr.FailedPrecondition("only PUBLISHED feeds can be archived")
	}
	f.Status = StatusArchived
	f.UpdatedAt = time.Now().UTC()
	return nil
}
