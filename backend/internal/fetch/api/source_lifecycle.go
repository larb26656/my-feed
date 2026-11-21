package api

import (
	"context"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

// SourceLifecycle updates Fetch Jobs when a Source is created or deactivated.
type SourceLifecycle interface {
	QueueInitialJob(ctx context.Context, sourceID uuid.UUID) error
	HandleSourceDeactivated(ctx context.Context, sourceID uuid.UUID) error
}

// TransactionalSourceLifecycle binds Fetch Job operations to a caller-owned transaction.
type TransactionalSourceLifecycle interface {
	WithTx(tx database.Transaction) SourceLifecycle
}
