package api

import "time"

type PreviewEntry struct {
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

type PreviewInput struct {
	Type             string
	RSSURL           string
	YouTubeChannelID string
}
