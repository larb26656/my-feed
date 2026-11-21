package service

import (
	"context"
	"strings"

	"github.com/google/uuid"
	fetchapi "github.com/larb26656/my-feed/backend/internal/modules/fetch/api"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

// Service coordinates Source changes that must commit with Fetch Job changes.
type Service struct {
	db     database.Transactor
	source sourceapi.TransactionalLifecycle
	fetch  fetchapi.TransactionalSourceLifecycle
}

var _ sourceapi.Lifecycle = (*Service)(nil)

func New(db database.Transactor, source sourceapi.TransactionalLifecycle, fetch fetchapi.TransactionalSourceLifecycle) *Service {
	return &Service{db: db, source: source, fetch: fetch}
}

func (s *Service) CreateSource(ctx context.Context, in sourceapi.CreateSourceInput) (*sourceapi.Source, error) {
	if strings.TrimSpace(in.Type) != string(sourceapi.SourceTypeYouTube) {
		return s.source.CreateSource(ctx, in)
	}
	var out *sourceapi.Source
	err := s.db.WithinTransaction(ctx, func(ctx context.Context, tx database.Transaction) error {
		src, err := s.source.WithTx(tx).CreateSource(ctx, in)
		if err != nil {
			return err
		}
		if err := s.fetch.WithTx(tx).QueueInitialJob(ctx, src.ID); err != nil {
			return err
		}
		out = src
		return nil
	})
	return out, err
}

func (s *Service) DeactivateSource(ctx context.Context, id uuid.UUID) (*sourceapi.Source, error) {
	var out *sourceapi.Source
	err := s.db.WithinTransaction(ctx, func(ctx context.Context, tx database.Transaction) error {
		src, err := s.source.WithTx(tx).DeactivateSource(ctx, id)
		if err != nil {
			return err
		}
		out = src
		return s.fetch.WithTx(tx).HandleSourceDeactivated(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
