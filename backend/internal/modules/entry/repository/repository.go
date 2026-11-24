package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

type Repository interface {
	CreateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error)
	GetEntryByID(ctx context.Context, id uuid.UUID) (*domain.Entry, error)
	GetEntryBySourceAndLink(ctx context.Context, sourceID uuid.UUID, link string) (*domain.Entry, error)
	ListEntries(ctx context.Context, after *EntryListCursor, criteria EntryListCriteria, sort EntryListSort, limit int) ([]*domain.Entry, error)
	CountSavedEntries(ctx context.Context) (int, error)
	UpdateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error)
	WithTx(tx database.Transaction) Repository
}
