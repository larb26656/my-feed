package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/platform/database"

	"github.com/larb26656/my-feed/backend/internal/modules/fetch/domain"
)

// Repository is the Fetch Job persistence port.
type Repository interface {
	CreateFetchJob(ctx context.Context, job *domain.FetchJob) error
	GetFetchJobByID(ctx context.Context, id uuid.UUID) (*domain.FetchJob, error)
	ListFetchJobs(ctx context.Context, after *FetchJobListCursor, criteria FetchJobListCriteria, limit int) ([]*domain.FetchJob, error)
	// GetUnfinishedFetchJobBySourceID returns the pending or running Fetch Job for a Source, if any.
	GetUnfinishedFetchJobBySourceID(ctx context.Context, sourceID uuid.UUID) (*domain.FetchJob, error)
	// ListClaimableFetchJobs returns pending jobs whose retry backoff has elapsed (or never set).
	ListClaimableFetchJobs(ctx context.Context, now time.Time) ([]*domain.FetchJob, error)
	// ClaimFetchJob atomically moves a pending job to running when still claimable.
	// Returns FailedPrecondition when another claimer won or backoff has not elapsed.
	ClaimFetchJob(ctx context.Context, id uuid.UUID, now time.Time) (*domain.FetchJob, error)
	UpdateFetchJob(ctx context.Context, job *domain.FetchJob) error
	WithTx(tx database.Transaction) Repository
}

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
