package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/larb26656/quick-feed/backend/internal/feed/domain"
)

type Repository interface {
	Create(ctx context.Context, f *domain.Feed) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Feed, error)
	Update(ctx context.Context, f *domain.Feed) error
	WithTx(tx pgx.Tx) Repository
}
