package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/obtain/api"
	obtain "github.com/larb26656/my-feed/backend/internal/modules/obtain/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type stubObtain struct {
	items []obtainapi.PreparedEntry
}

func (s stubObtain) Obtain(_ context.Context, _ *obtainapi.Input) ([]obtainapi.PreparedEntry, error) {
	return s.items, nil
}

func TestRegistryDispatchesBySourceType(t *testing.T) {
	reg := obtain.NewRegistry()
	reg.Register(obtainapi.ProviderRSS, stubObtain{
		items: []obtainapi.PreparedEntry{{Title: "A", Link: "https://example.com/a"}},
	})

	items, err := reg.Obtain(context.Background(), &obtainapi.Input{
		Type: obtainapi.ProviderRSS,
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "A", items[0].Title)
}

func TestRegistryRejectsUnregisteredType(t *testing.T) {
	reg := obtain.NewRegistry()
	_, err := reg.Obtain(context.Background(), &obtainapi.Input{
		Type: "manual",
	})
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
}
