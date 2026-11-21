package api

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// IngestEntryInput is the data needed to record an Entry.
type IngestEntryInput struct {
	SourceID    uuid.UUID
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

// IngestEntryResult reports whether ingest newly created an Entry.
type IngestEntryResult struct {
	Created bool
}

// Ingester records Entries into active fetchable Sources.
type Ingester interface {
	IngestEntry(ctx context.Context, in IngestEntryInput) (*IngestEntryResult, error)
}
