package http

import (
	"time"

	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/domain"
)

type fetchJobResponse struct {
	ID             uuid.UUID `json:"id"`
	SourceID       uuid.UUID `json:"sourceId"`
	Status         string    `json:"status"`
	Attempt        int       `json:"attempt"`
	MaxAttempts    int       `json:"maxAttempts"`
	EntriesCreated int       `json:"entriesCreated"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func toFetchJobResponse(job *domain.FetchJob) fetchJobResponse {
	return fetchJobResponse{job.ID, job.SourceID, string(job.Status), job.Attempt, job.MaxAttempts, job.EntriesCreated, job.CreatedAt.UTC(), job.UpdatedAt.UTC()}
}

// CursorResponse describes the list envelope emitted by httpx.Cursor.
type CursorResponse struct {
	Success bool               `json:"success"`
	Data    []fetchJobResponse `json:"data"`
	Meta    httpx.CursorMeta   `json:"meta"`
}
