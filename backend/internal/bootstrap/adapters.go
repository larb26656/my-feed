package bootstrap

import (
	"context"
	"time"

	"github.com/google/uuid"

	entrysvc "github.com/larb26656/my-feed/backend/internal/entry/service"
	sourcedomain "github.com/larb26656/my-feed/backend/internal/source/domain"
	sourcerepo "github.com/larb26656/my-feed/backend/internal/source/repository"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// sourceLookup adapts the Source repository port for Entry without importing
// source/adapter/sqlite into the entry module.
type sourceLookup struct {
	repo sourcerepo.Repository
}

func (s sourceLookup) Get(ctx context.Context, id uuid.UUID) (*entrysvc.SourceRef, error) {
	src, err := s.repo.GetSourceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toEntrySourceRef(src), nil
}

func (s sourceLookup) GetManual(ctx context.Context) (*entrysvc.SourceRef, error) {
	src, err := s.repo.GetManualSource(ctx)
	if err != nil {
		return nil, err
	}
	return toEntrySourceRef(src), nil
}

func toEntrySourceRef(src *sourcedomain.Source) *entrysvc.SourceRef {
	return &entrysvc.SourceRef{
		ID:     src.ID,
		Type:   entrysvc.SourceType(src.Type),
		Active: src.Active,
	}
}
