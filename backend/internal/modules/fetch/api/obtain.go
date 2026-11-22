package api

import (
	"context"
	"time"

	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
)

// PreparedEntry contains fields obtained from an origin without persisting them.
type PreparedEntry struct {
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

// Obtain reads entries from a Source without persisting them.
type Obtain interface {
	Obtain(context.Context, *sourceapi.SourceRef) ([]PreparedEntry, error)
}
