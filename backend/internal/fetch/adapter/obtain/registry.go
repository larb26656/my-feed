package obtain

import (
	"context"
	"fmt"
	"github.com/larb26656/my-feed/backend/internal/fetch/domain"

	"github.com/larb26656/my-feed/backend/internal/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

// Registry dispatches obtain adapters by Source Type. It implements service.Obtain.
type Registry struct {
	byType map[service.SourceType]service.Obtain
}

func NewRegistry() *Registry {
	return &Registry{byType: make(map[service.SourceType]service.Obtain)}
}

// Register wires an obtain adapter for a Source Type. Later registrations replace earlier ones.
func (r *Registry) Register(typ service.SourceType, adapter service.Obtain) {
	if adapter == nil {
		return
	}
	r.byType[typ] = adapter
}

func (r *Registry) Obtain(ctx context.Context, source *service.SourceRef) ([]service.PreparedEntry, error) {
	if source == nil {
		return nil, apperr.InvalidInput.New("source is required")
	}
	adapter, ok := r.byType[source.Type]
	if !ok {
		return nil, domain.ErrConflict.New(fmt.Sprintf("no obtain adapter registered for source type %q", source.Type))
	}
	return adapter.Obtain(ctx, source)
}
