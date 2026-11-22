package api

import (
	"context"
	"time"
)

type Entry struct {
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

type Input struct {
	Type             string
	RSSURL           string
	YouTubeChannelID string
}

type Previewer interface {
	Preview(context.Context, Input) (*Entry, error)
}
