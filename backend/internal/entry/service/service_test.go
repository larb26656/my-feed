package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/entry/repository"
	"github.com/larb26656/my-feed/backend/internal/entry/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/stretchr/testify/require"
)

// Embed the port so unexpected calls fail immediately. Only the upsert seam is
// implemented: these tests specify failures that HTTP/SQLite cannot force reliably.
type upsertRepo struct {
	repository.Repository
	lookup func() (*domain.Entry, error)
	create func(*domain.Entry) (*domain.Entry, error)
	update func(*domain.Entry) (*domain.Entry, error)
}

func (r upsertRepo) GetEntryBySourceAndLink(context.Context, uuid.UUID, string) (*domain.Entry, error) {
	return r.lookup()
}
func (r upsertRepo) CreateEntry(_ context.Context, e *domain.Entry) (*domain.Entry, error) {
	return r.create(e)
}
func (r upsertRepo) UpdateEntry(_ context.Context, e *domain.Entry) (*domain.Entry, error) {
	return r.update(e)
}

type rssLookup struct {
	service.Sources
	id uuid.UUID
}

func (s rssLookup) Get(context.Context, uuid.UUID) (*service.SourceRef, error) {
	return &service.SourceRef{ID: s.id, Type: service.SourceTypeRSS, Active: true}, nil
}

func TestIngestionRecoversWhenAnotherWriterCreatesTheSameArticleFirst(t *testing.T) {
	// Arrange
	sourceID := uuid.New()
	winner, err := domain.NewEntry(sourceID, domain.EntryMetadata{Title: "First writer", Link: "https://example.test/a"})
	require.NoError(t, err)
	at := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	winner.MarkRead(at)
	winner.MarkSaved(at)
	lookups := 0
	repo := upsertRepo{
		lookup: func() (*domain.Entry, error) {
			lookups++
			if lookups == 1 {
				return nil, apperr.NotFound("entry not found")
			}
			return winner, nil
		},
		create: func(*domain.Entry) (*domain.Entry, error) { return nil, apperr.Conflict("another writer won") },
		update: func(e *domain.Entry) (*domain.Entry, error) { require.Equal(t, winner.ID, e.ID); return e, nil },
	}

	// Act: ingest through the configured repository failure or conflict.
	result, err := service.New(repo, rssLookup{id: sourceID}).IngestEntry(context.Background(), service.IngestEntryInput{SourceID: sourceID, Title: "Latest metadata", Link: winner.Link})

	// Assert
	require.NoError(t, err)
	require.False(t, result.Created)
	require.Equal(t, 2, lookups)
	require.Equal(t, winner.ID, result.Entry.ID)
	require.Equal(t, "Latest metadata", result.Entry.Title)
	require.Equal(t, &at, result.Entry.ReadAt)
	require.Equal(t, &at, result.Entry.SavedAt)
}

func TestIngestionPropagatesStorageFailuresInsteadOfReportingSuccess(t *testing.T) {
	// Arrange
	storageFailure := errors.New("storage unavailable")
	for _, stage := range []string{"lookup", "create", "refresh", "lookup after conflict", "refresh after conflict"} {
		t.Run(stage, func(t *testing.T) {
			// Arrange: force a failure at this storage stage.
			sourceID := uuid.New()
			existing, err := domain.NewEntry(sourceID, domain.EntryMetadata{Title: "Existing", Link: "https://example.test/a"})
			require.NoError(t, err)
			calls := 0
			repo := upsertRepo{
				lookup: func() (*domain.Entry, error) {
					calls++
					if stage == "lookup" || (stage == "lookup after conflict" && calls == 2) {
						return nil, storageFailure
					}
					if stage == "refresh" || calls == 2 {
						return existing, nil
					}
					return nil, apperr.NotFound("entry not found")
				},
				create: func(*domain.Entry) (*domain.Entry, error) {
					if stage == "create" {
						return nil, storageFailure
					}
					return nil, apperr.Conflict("another writer won")
				},
				update: func(*domain.Entry) (*domain.Entry, error) { return nil, storageFailure },
			}

			// Act: ingest through the configured repository failure or conflict.
			result, err := service.New(repo, rssLookup{id: sourceID}).IngestEntry(context.Background(), service.IngestEntryInput{SourceID: sourceID, Title: "Incoming", Link: existing.Link})

			// Assert
			require.ErrorIs(t, err, storageFailure)
			require.Nil(t, result)
		})
	}
}
