package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

// EntryOrder selects which timestamp orders an Entry list.
// The zero value orders by publication time.
type EntryOrder string

const (
	// EntryOrderPublished orders by publication time, then recorded time.
	EntryOrderPublished EntryOrder = ""
	// EntryOrderSaved orders by the time the reader saved the Entry.
	EntryOrderSaved EntryOrder = "saved"
	// EntryOrderRecorded orders by the time the Entry was recorded.
	EntryOrderRecorded EntryOrder = "recorded"
)

// EntryListCursor is the decoded position in the ordered Entry stream.
type EntryListCursor struct {
	SortAt time.Time
	ID     uuid.UUID
}

type EntryListCriteria struct {
	SourceID      *uuid.UUID
	Unread        bool
	SavedOnly     bool
	SortDirection string
	// Order is chosen by the service from the list being read.
	// Callers outside the service leave it empty to order by publication time.
	Order EntryOrder
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
