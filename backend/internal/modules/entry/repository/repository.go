package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
	sortoption "github.com/larb26656/my-feed/backend/internal/platform/sort"
)

// EntryListCursor is the decoded position in the ordered Entry stream.
type EntryListCursor struct {
	CreatedAt   time.Time  `json:"createdAt"`
	PublishedAt *time.Time `json:"publishedAt"`
	SavedAt     *time.Time `json:"savedAt"`
	ID          uuid.UUID  `json:"id"`
}

type EntryListSort struct {
	CreatedAtSort   sortoption.Direction `json:"createdAtSort"`
	PublishedAtSort sortoption.Direction `json:"publishedAtSort"`
	SavedAtSort     sortoption.Direction `json:"savedAtSort"`
}

type EntryListCriteria struct {
	SourceID  *uuid.UUID    `json:"sourceId"`
	Unread    bool          `json:"unread"`
	SavedOnly bool          `json:"savedOnly"`
	Sort      EntryListSort `json:"sort"`
}

type Repository interface {
	CreateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error)
	GetEntryByID(ctx context.Context, id uuid.UUID) (*domain.Entry, error)
	GetEntryBySourceAndLink(ctx context.Context, sourceID uuid.UUID, link string) (*domain.Entry, error)
	ListEntries(ctx context.Context, after *EntryListCursor, criteria EntryListCriteria, limit int) ([]*domain.Entry, error)
	CountSavedEntries(ctx context.Context) (int, error)
	UpdateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error)
	WithTx(tx database.Transaction) Repository
}
