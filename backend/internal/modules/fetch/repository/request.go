package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/domain"
)

// FetchJobListCursor is a position in the newest-first Fetch Job history.
type FetchJobListCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// FetchJobListCriteria contains the supported Fetch Job history filters.
type FetchJobListCriteria struct {
	SourceID *uuid.UUID
	Status   *domain.Status
}
