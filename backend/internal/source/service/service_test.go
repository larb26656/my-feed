package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/source/adapter/sqlite"
	"github.com/larb26656/my-feed/backend/internal/source/domain"
	"github.com/larb26656/my-feed/backend/internal/source/repository"
	"github.com/larb26656/my-feed/backend/internal/source/service"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

func TestEnsuringSavedLinksRepeatedlyKeepsOneSystemSource(t *testing.T) {
	// Arrange
	pool := testkit.NewDatabase(t)
	svc := service.New(sourcesqlite.NewRepository(pool.DB))
	first, err := svc.EnsureSavedLinks(context.Background())
	require.NoError(t, err)

	// Act: ensure the same source again and inspect the stored result.
	second, err := svc.EnsureSavedLinks(context.Background())
	require.NoError(t, err)
	sources, err := svc.ListSources(context.Background(), service.ListSourcesInput{})

	// Assert
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, sources, 1)
	require.Equal(t, domain.TypeManual, sources[0].Type)
	require.Equal(t, "Saved links", sources[0].Name)
}

type manualRepo struct {
	repository.Repository
	lookup func() (*domain.Source, error)
	create func(*domain.Source) error
}

func (r manualRepo) GetManualSource(context.Context) (*domain.Source, error) { return r.lookup() }
func (r manualRepo) CreateSource(_ context.Context, s *domain.Source) error  { return r.create(s) }

func TestEnsuringSavedLinksUsesTheSourceCreatedByAnotherWriter(t *testing.T) {
	// Arrange
	winner, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)
	lookups := 0
	svc := service.New(manualRepo{
		lookup: func() (*domain.Source, error) {
			lookups++
			if lookups == 1 {
				return nil, apperr.NotFound("source not found")
			}
			return winner, nil
		},
		create: func(*domain.Source) error { return apperr.Conflict("another writer won") },
	})

	// Act: ensure the system source.
	got, err := svc.EnsureSavedLinks(context.Background())

	// Assert
	require.NoError(t, err)
	require.Equal(t, winner, got)
	require.Equal(t, 2, lookups)
}

func TestEnsuringSavedLinksPropagatesStorageFailures(t *testing.T) {
	// Arrange
	failure := errors.New("storage unavailable")
	for _, stage := range []string{"lookup", "create", "lookup after conflict"} {
		t.Run(stage, func(t *testing.T) {
			// Arrange: force a failure at this storage stage.
			lookups := 0
			svc := service.New(manualRepo{
				lookup: func() (*domain.Source, error) {
					lookups++
					if stage == "lookup" || lookups == 2 {
						return nil, failure
					}
					return nil, apperr.NotFound("source not found")
				},
				create: func(*domain.Source) error {
					if stage == "create" {
						return failure
					}
					return apperr.Conflict("another writer won")
				},
			})

			// Act: ensure the system source.
			got, err := svc.EnsureSavedLinks(context.Background())

			// Assert
			require.ErrorIs(t, err, failure)
			require.Nil(t, got)
		})
	}
}
