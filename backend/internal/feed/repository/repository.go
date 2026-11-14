package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/larb26656/quick-feed/backend/internal/feed/domain"
)

// EntryListCursor is the decoded position in the newest-first Entry stream.
type EntryListCursor struct {
	SortAt time.Time
	ID     uuid.UUID
}

type Repository interface {
	CreateSource(ctx context.Context, s *domain.Source) error
	GetSourceByID(ctx context.Context, id uuid.UUID) (*domain.Source, error)
	CreateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error)
	GetEntryByID(ctx context.Context, id uuid.UUID) (*domain.Entry, error)
	GetEntryBySourceAndLink(ctx context.Context, sourceID uuid.UUID, link string) (*domain.Entry, error)
	ListEntries(ctx context.Context, after *EntryListCursor, limit int) ([]*domain.Entry, error)
	UpdateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error)
	WithTx(tx pgx.Tx) Repository
}
