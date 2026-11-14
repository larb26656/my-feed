package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/larb26656/quick-feed/backend/internal/source/domain"
)

type ListSourcesFilter struct {
	IncludeDeactivated bool
}

type Repository interface {
	CreateSource(ctx context.Context, s *domain.Source) error
	GetSourceByID(ctx context.Context, id uuid.UUID) (*domain.Source, error)
	GetManualSource(ctx context.Context) (*domain.Source, error)
	ListSources(ctx context.Context, filter ListSourcesFilter) ([]*domain.Source, error)
	UpdateSource(ctx context.Context, s *domain.Source) error
	WithTx(tx pgx.Tx) Repository
}
