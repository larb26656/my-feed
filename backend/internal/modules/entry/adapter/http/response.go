package http

import (
	"time"

	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"

	"github.com/larb26656/my-feed/backend/internal/modules/entry/domain"
)

type savedCountResponse struct {
	Count int `json:"count"`
}

type entryResponse struct {
	ID          string     `json:"id"`
	SourceID    string     `json:"sourceId"`
	Title       string     `json:"title"`
	Link        string     `json:"link"`
	Description string     `json:"description"`
	Thumbnail   *string    `json:"thumbnail,omitempty"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	ReadAt      *time.Time `json:"readAt,omitempty"`
	SavedAt     *time.Time `json:"savedAt,omitempty"`
}

func toEntryResponse(e *domain.Entry) entryResponse {
	return entryResponse{
		ID:          e.ID.String(),
		SourceID:    e.SourceID.String(),
		Title:       e.Title,
		Link:        e.Link,
		Description: e.Description,
		Thumbnail:   e.Thumbnail,
		PublishedAt: e.PublishedAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		ReadAt:      e.ReadAt,
		SavedAt:     e.SavedAt,
	}
}

// CursorResponse describes the list envelope emitted by httpx.Cursor.
type CursorResponse struct {
	Success bool             `json:"success"`
	Data    []entryResponse  `json:"data"`
	Meta    httpx.CursorMeta `json:"meta"`
}
