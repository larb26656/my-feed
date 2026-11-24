package repository

import (
	"time"

	"github.com/google/uuid"
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
	SourceID  *uuid.UUID `json:"sourceId"`
	Unread    bool       `json:"unread"`
	SavedOnly bool       `json:"savedOnly"`
}
