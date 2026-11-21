package service

import (
	"context"

	entryapi "github.com/larb26656/my-feed/backend/internal/entry/api"
)

type ingester struct {
	service *Service
}

// NewIngester exposes the Entry ingestion contract for other modules.
func NewIngester(service *Service) entryapi.Ingester {
	return ingester{service: service}
}

func (i ingester) IngestEntry(ctx context.Context, in entryapi.IngestEntryInput) (*entryapi.IngestEntryResult, error) {
	result, err := i.service.IngestEntry(ctx, in)
	if err != nil {
		return nil, err
	}
	return &entryapi.IngestEntryResult{Created: result.Created}, nil
}
