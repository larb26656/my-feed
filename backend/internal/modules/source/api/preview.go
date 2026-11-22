package api

import "context"

type PreviewInput struct {
	Type             string
	RSSURL           string
	YouTubeChannelID string
}

// PreviewValidator validates an unsaved Source configuration for preview.
type PreviewValidator interface {
	ValidatePreview(context.Context, PreviewInput) (*SourceRef, error)
}
