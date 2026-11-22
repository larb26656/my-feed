package service

import (
	"context"
	"fmt"

	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/obtain/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

// Registry dispatches obtain adapters by Source Type. It implements obtainapi.Obtainer.
type Registry struct {
	byType map[obtainapi.Provider]obtainapi.Obtainer
}

func NewRegistry() *Registry {
	return &Registry{byType: make(map[obtainapi.Provider]obtainapi.Obtainer)}
}

// Register wires an obtain adapter for a Source Type. Later registrations replace earlier ones.
func (r *Registry) Register(typ obtainapi.Provider, adapter obtainapi.Obtainer) {
	if adapter == nil {
		return
	}
	r.byType[typ] = adapter
}

func (r *Registry) Obtain(ctx context.Context, source *obtainapi.Input) ([]obtainapi.PreparedEntry, error) {
	if source == nil {
		return nil, apperr.InvalidInput.New("source is required")
	}
	adapter, ok := r.byType[source.Type]
	if !ok {
		return nil, apperr.ResourceConflict.New(fmt.Sprintf("no obtain adapter registered for source type %q", source.Type))
	}
	return adapter.Obtain(ctx, source)
}
