package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/quick-feed/backend/internal/entry/domain"
	"github.com/larb26656/quick-feed/backend/internal/platform/database"
)

// EntryListCursor is the decoded position in the newest-first Entry stream.
type EntryListCursor struct {
	SortAt time.Time
	ID     uuid.UUID
}

type EntryListCriteria struct {
	SourceID  *uuid.UUID
	Unread    bool
	SavedOnly bool
}

type Repository interface {
	CreateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error)
	GetEntryByID(ctx context.Context, id uuid.UUID) (*domain.Entry, error)
	GetEntryBySourceAndLink(ctx context.Context, sourceID uuid.UUID, link string) (*domain.Entry, error)
	ListEntries(ctx context.Context, after *EntryListCursor, criteria EntryListCriteria, limit int) ([]*domain.Entry, error)
	UpdateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error)
	WithTx(tx database.Transaction) Repository
}
