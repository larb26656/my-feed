package api

import (
	"context"

	"github.com/google/uuid"
)

// SourceType is the type of a Source exposed by Reader.
type SourceType string

const (
	SourceTypeRSS     SourceType = "rss"
	SourceTypeYouTube SourceType = "youtube"
	SourceTypeManual  SourceType = "manual"
)

// SourceRef is the Source data needed to schedule and execute Fetch Jobs.
type SourceRef struct {
	ID               uuid.UUID
	Type             SourceType
	Active           bool
	RSSURL           string
	YouTubeChannelID string
}

// Reader provides Source data for Fetch.
type Reader interface {
	Get(ctx context.Context, id uuid.UUID) (*SourceRef, error)
	// ListActiveFetchable returns active fetchable Sources (excludes manual and deactivated).
	ListActiveFetchable(ctx context.Context) ([]SourceRef, error)
}
