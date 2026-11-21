package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

type ListSourcesFilter struct {
	IncludeDeactivated bool
}

type SourcePageQuery struct {
	NameLike       string
	TypeEqual      string
	ActiveEqual    *bool
	SortBy         string
	SortDirection  string
	Page, PageSize int
}

type Repository interface {
	CreateSource(ctx context.Context, s *domain.Source) error
	GetSourceByID(ctx context.Context, id uuid.UUID) (*domain.Source, error)
	GetManualSource(ctx context.Context) (*domain.Source, error)
	ListSources(ctx context.Context, filter ListSourcesFilter) ([]*domain.Source, error)
	ListSourcesPage(ctx context.Context, query SourcePageQuery) ([]*domain.Source, int64, error)
	UpdateSource(ctx context.Context, s *domain.Source) error
	WithTx(tx database.Transaction) Repository
}
