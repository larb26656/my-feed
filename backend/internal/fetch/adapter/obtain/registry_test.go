package obtain_test

import (
	"context"
	"github.com/larb26656/my-feed/backend/internal/fetch/domain"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/fetch/adapter/obtain"
	"github.com/larb26656/my-feed/backend/internal/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type stubObtain struct {
	items []service.PreparedEntry
}

func (s stubObtain) Obtain(_ context.Context, _ *service.SourceRef) ([]service.PreparedEntry, error) {
	return s.items, nil
}

func TestRegistryDispatchesBySourceType(t *testing.T) {
	reg := obtain.NewRegistry()
	reg.Register(service.SourceTypeRSS, stubObtain{
		items: []service.PreparedEntry{{Title: "A", Link: "https://example.com/a"}},
	})

	items, err := reg.Obtain(context.Background(), &service.SourceRef{
		ID:   uuid.New(),
		Type: service.SourceTypeRSS,
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "A", items[0].Title)
}

func TestRegistryRejectsUnregisteredType(t *testing.T) {
	reg := obtain.NewRegistry()
	_, err := reg.Obtain(context.Background(), &service.SourceRef{
		ID:   uuid.New(),
		Type: service.SourceTypeManual,
	})
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, domain.ErrConflict.Code, ae.Code)
}
