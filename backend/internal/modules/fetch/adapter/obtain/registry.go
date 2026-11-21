package obtain

import (
	"context"
	"fmt"

	"github.com/larb26656/my-feed/backend/internal/modules/fetch/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
)

// Registry dispatches obtain adapters by Source Type. It implements service.Obtain.
type Registry struct {
	byType map[sourceapi.SourceType]service.Obtain
}

func NewRegistry() *Registry {
	return &Registry{byType: make(map[sourceapi.SourceType]service.Obtain)}
}

// Register wires an obtain adapter for a Source Type. Later registrations replace earlier ones.
func (r *Registry) Register(typ sourceapi.SourceType, adapter service.Obtain) {
	if adapter == nil {
		return
	}
	r.byType[typ] = adapter
}

func (r *Registry) Obtain(ctx context.Context, source *sourceapi.SourceRef) ([]service.PreparedEntry, error) {
	if source == nil {
		return nil, apperr.InvalidInput.New("source is required")
	}
	adapter, ok := r.byType[source.Type]
	if !ok {
		return nil, domain.ErrConflict.New(fmt.Sprintf("no obtain adapter registered for source type %q", source.Type))
	}
	return adapter.Obtain(ctx, source)
}
