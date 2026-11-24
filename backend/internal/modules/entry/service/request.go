package service

import (
	"time"

	entryapi "github.com/larb26656/my-feed/backend/internal/modules/entry/api"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/repository"
)

type SaveLinkInput struct {
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

type IngestEntryInput = entryapi.IngestEntryInput

type ListEntriesInput struct {
	Cursor   *string
	PageSize *int
	Criteria repository.EntryListCriteria
	Sort     repository.EntryListSort
}
