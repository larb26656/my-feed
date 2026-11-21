package api

import (
	"context"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/platform/database"
	"github.com/larb26656/my-feed/backend/internal/source/domain"
)

type Source = domain.Source

type CreateSourceInput struct {
	Name             string
	Type             string
	RSSURL           string
	YouTubeChannelID string
}

// Lifecycle exposes the Source operations that participate in cross-module transactions.
type Lifecycle interface {
	CreateSource(ctx context.Context, in CreateSourceInput) (*Source, error)
	DeactivateSource(ctx context.Context, id uuid.UUID) (*Source, error)
}

// TransactionalLifecycle can bind Source operations to a caller-owned transaction.
type TransactionalLifecycle interface {
	Lifecycle
	WithTx(tx database.Transaction) Lifecycle
}
