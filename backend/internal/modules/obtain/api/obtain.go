package api

import (
	"context"
	"time"
)

// PreparedEntry contains fields obtained from an origin without persisting them.
type PreparedEntry struct {
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

type Provider string

const (
	ProviderRSS     Provider = "rss"
	ProviderYouTube Provider = "youtube"
)

// Input identifies external content to read.
type Input struct {
	Type             Provider
	RSSURL           string
	YouTubeChannelID string
}

// Obtainer reads and normalizes entries from an external provider.
type Obtainer interface {
	Obtain(context.Context, *Input) ([]PreparedEntry, error)
}
