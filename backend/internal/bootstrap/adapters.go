package bootstrap

import (
	"context"
	"time"

	"github.com/google/uuid"

	entrysvc "github.com/larb26656/my-feed/backend/internal/entry/service"
	fetchsvc "github.com/larb26656/my-feed/backend/internal/fetch/service"
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

// fetchSources adapts Source persistence for the Fetch module's consumer port.
type fetchSources struct {
	repo sourcerepo.Repository
}

func (f fetchSources) Get(ctx context.Context, id uuid.UUID) (*fetchsvc.SourceRef, error) {
	src, err := f.repo.GetSourceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toFetchSourceRef(src), nil
}

func (f fetchSources) ListActiveFetchable(ctx context.Context) ([]fetchsvc.SourceRef, error) {
	list, err := f.repo.ListSources(ctx, sourcerepo.ListSourcesFilter{IncludeDeactivated: false})
	if err != nil {
		return nil, err
	}
	out := make([]fetchsvc.SourceRef, 0, len(list))
	for _, src := range list {
		if src.Type != sourcedomain.TypeRSS {
			continue
		}
		out = append(out, *toFetchSourceRef(src))
	}
	return out, nil
}

func toFetchSourceRef(src *sourcedomain.Source) *fetchsvc.SourceRef {
	return &fetchsvc.SourceRef{
		ID:     src.ID,
		Type:   fetchsvc.SourceType(src.Type),
		Active: src.Active,
		RSSURL: src.RSSURL,
	}
}

// entryIngest adapts entry IngestEntry for Fetch without importing entry SQLite.
type entryIngest struct {
	svc *entrysvc.Service
}

func (e entryIngest) IngestEntry(ctx context.Context, in fetchsvc.IngestEntryInput) (*fetchsvc.IngestEntryResult, error) {
	result, err := e.svc.IngestEntry(ctx, entrysvc.IngestEntryInput{
		SourceID:    in.SourceID,
		Title:       in.Title,
		Link:        in.Link,
		Description: in.Description,
		Thumbnail:   in.Thumbnail,
		PublishedAt: in.PublishedAt,
	})
	if err != nil {
		return nil, err
	}
	return &fetchsvc.IngestEntryResult{Created: result.Created}, nil
}
