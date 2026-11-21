package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/auth/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

// Repository persists Sessions and their refresh tokens.
type Repository interface {
	CreateSession(ctx context.Context, session *domain.Session) error
	GetSession(ctx context.Context, id uuid.UUID) (*domain.Session, error)
	RevokeSession(ctx context.Context, session *domain.Session) error

	CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	MarkRefreshTokenReplaced(ctx context.Context, token *domain.RefreshToken) error

	WithTx(tx database.Transaction) Repository
}
