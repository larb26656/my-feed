package http

import (
	"time"

	previewapi "github.com/larb26656/my-feed/backend/internal/modules/sourcepreview/api"
)

type (
	previewResponse struct {
		Entry *previewEntryResponse `json:"entry" extensions:"x-nullable"`
	}
	previewEntryResponse struct {
		Title       string     `json:"title" binding:"required"`
		Link        string     `json:"link" binding:"required"`
		Description string     `json:"description,omitempty"`
		Thumbnail   *string    `json:"thumbnail,omitempty"`
		PublishedAt *time.Time `json:"publishedAt,omitempty"`
	}
)

func toPreviewResponse(e *previewapi.Entry) previewResponse {
	if e == nil {
		return previewResponse{}
	}
	return previewResponse{Entry: &previewEntryResponse{Title: e.Title, Link: e.Link, Description: e.Description, Thumbnail: e.Thumbnail, PublishedAt: e.PublishedAt}}
}
