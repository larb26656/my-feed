package bootstrap

import (
	"context"

	"github.com/google/uuid"
	fetchrepo "github.com/larb26656/quick-feed/backend/internal/fetch/repository"
	fetchsvc "github.com/larb26656/quick-feed/backend/internal/fetch/service"
	"github.com/larb26656/quick-feed/backend/internal/platform/database"
	sourcedomain "github.com/larb26656/quick-feed/backend/internal/source/domain"
	sourcerepo "github.com/larb26656/quick-feed/backend/internal/source/repository"
	sourcesvc "github.com/larb26656/quick-feed/backend/internal/source/service"
)

// sourceAPIWithFetch wraps Source service so deactivate fails pending Fetch Jobs
// in the same transaction as the Source update (no partial success).
type sourceAPIWithFetch struct {
	db         database.Transactor
	sourceRepo sourcerepo.Repository
	fetchRepo  fetchrepo.Repository
	inner      *sourcesvc.Service
	fetch      *fetchsvc.Service
}

func (s sourceAPIWithFetch) CreateSource(ctx context.Context, in sourcesvc.CreateSourceInput) (*sourcedomain.Source, error) {
	return s.inner.CreateSource(ctx, in)
}

func (s sourceAPIWithFetch) ListSources(ctx context.Context, in sourcesvc.ListSourcesInput) ([]*sourcedomain.Source, error) {
	return s.inner.ListSources(ctx, in)
}

func (s sourceAPIWithFetch) GetSource(ctx context.Context, id uuid.UUID) (*sourcedomain.Source, error) {
	return s.inner.GetSource(ctx, id)
}

func (s sourceAPIWithFetch) UpdateSource(ctx context.Context, id uuid.UUID, in sourcesvc.UpdateSourceInput) (*sourcedomain.Source, error) {
	return s.inner.UpdateSource(ctx, id, in)
}

func (s sourceAPIWithFetch) ReactivateSource(ctx context.Context, id uuid.UUID) (*sourcedomain.Source, error) {
	return s.inner.ReactivateSource(ctx, id)
}

func (s sourceAPIWithFetch) DeactivateSource(ctx context.Context, id uuid.UUID) (*sourcedomain.Source, error) {
	var out *sourcedomain.Source
	err := s.db.WithinTransaction(ctx, func(ctx context.Context, tx database.Transaction) error {
		src, err := sourcesvc.New(s.sourceRepo.WithTx(tx)).DeactivateSource(ctx, id)
		if err != nil {
			return err
		}
		out = src
		return s.fetch.WithRepository(s.fetchRepo.WithTx(tx)).HandleSourceDeactivated(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
