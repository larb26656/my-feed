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
	SourceTypeBluesky SourceType = "bluesky"
	SourceTypeManual  SourceType = "manual"
)

// SourceRef is the Source data exposed to other modules.
type SourceRef struct {
	Config           map[string]any
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

// EntryReader provides Source lookups for Entry writes.
type EntryReader interface {
	Get(ctx context.Context, id uuid.UUID) (*SourceRef, error)
	GetManual(ctx context.Context) (*SourceRef, error)
}
