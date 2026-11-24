package service

import "github.com/larb26656/my-feed/backend/internal/modules/entry/domain"

// SaveLinkResult is the outcome of recording a link on the manual Source.
// Created is true only when a new Entry was written; metadata refresh is false.
type SaveLinkResult struct {
	Entry   *domain.Entry
	Created bool
}

// IngestEntryResult is the outcome of recording an Entry through ingest.
// Created is true only when a new Entry was written; metadata refresh is false.
type IngestEntryResult struct {
	Entry   *domain.Entry
	Created bool
}

type ListEntriesResult struct {
	Entries    []*domain.Entry
	NextCursor *string
}
