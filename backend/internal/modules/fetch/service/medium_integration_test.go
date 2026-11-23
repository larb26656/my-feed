package service_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	entryapi "github.com/larb26656/my-feed/backend/internal/modules/entry/api"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/medium"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"
	typesvc "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	"github.com/larb26656/my-feed/backend/internal/platform/outbound"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

func TestFetchJobObtainsMediumEntriesFromStoredConfig(t *testing.T) {
	// Arrange
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	sourceService := sourcesvc.New(sourcesqlite.NewRepository(pool.DB))

	var requested []string
	client := outbound.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requested = append(requested, r.URL.String())
		body := `<rss version="2.0"><channel><title>Alice</title><item><title>Story</title><link>https://medium.com/@alice/story</link></item></channel></rss>`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})})
	registry, err := typesvc.New(medium.Definition(client))
	require.NoError(t, err)
	sourceService.SetSourceTypes(registry)

	src, err := sourceService.CreateSource(ctx, sourceapi.CreateSourceInput{
		Name: "Alice", Type: string(sourceapi.SourceTypeMedium), Config: map[string]any{"kind": "profile", "identifier": "@Alice"},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"kind": "profile", "identifier": "Alice"}, src.Config)

	h := newHarness()
	ingest := &fakeIngest{createdByLink: map[string]bool{"https://medium.com/@alice/story": true}}
	h.svc = service.New(h.repo, sourceService, registry, ingest, h.clock, service.DefaultRetryBackoff, service.DefaultScheduleInterval)

	// Act
	job, err := h.svc.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)
	result, err := h.svc.ExecuteFetchJob(ctx, job.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, []string{"https://medium.com/feed/@Alice"}, requested)
	require.Len(t, ingest.calls, 1)
	require.Equal(t, "Story", ingest.calls[0].item.Title)
	require.Equal(t, 1, result.EntriesCreated)

	// Act: re-fetch same link refreshes without counting as a new create
	ingest.createdByLink["https://medium.com/@alice/story"] = false
	job2, err := h.svc.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)
	result2, err := h.svc.ExecuteFetchJob(ctx, job2.ID)

	// Assert
	require.NoError(t, err)
	require.Len(t, ingest.calls, 2)
	require.Equal(t, 0, result2.EntriesCreated)
}

func TestFetchJobObtainsMediumPublicationEntriesFromStoredConfig(t *testing.T) {
	// Arrange
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	sourceService := sourcesvc.New(sourcesqlite.NewRepository(pool.DB))

	var requested []string
	client := outbound.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requested = append(requested, r.URL.String())
		body := `<rss version="2.0"><channel><title>Pub</title><item><title>Pub Story</title><link>https://medium.com/alice/story</link></item></channel></rss>`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})})
	registry, err := typesvc.New(medium.Definition(client))
	require.NoError(t, err)
	sourceService.SetSourceTypes(registry)

	src, err := sourceService.CreateSource(ctx, sourceapi.CreateSourceInput{
		Name: "Alice Pub", Type: string(sourceapi.SourceTypeMedium), Config: map[string]any{"kind": "publication", "identifier": "Alice"},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"kind": "publication", "identifier": "Alice"}, src.Config)

	h := newHarness()
	ingest := &fakeIngest{createdByLink: map[string]bool{"https://medium.com/alice/story": true}}
	h.svc = service.New(h.repo, sourceService, registry, ingest, h.clock, service.DefaultRetryBackoff, service.DefaultScheduleInterval)

	// Act
	job, err := h.svc.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)
	result, err := h.svc.ExecuteFetchJob(ctx, job.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, []string{"https://medium.com/feed/Alice"}, requested)
	require.Len(t, ingest.calls, 1)
	require.Equal(t, src.ID, ingest.calls[0].sourceID)
	require.Equal(t, 1, result.EntriesCreated)

	// Act: re-fetch refreshes without a new create
	ingest.createdByLink["https://medium.com/alice/story"] = false
	job2, err := h.svc.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)
	result2, err := h.svc.ExecuteFetchJob(ctx, job2.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, 0, result2.EntriesCreated)
}

func TestFetchJobRejectsMediumCustomDomainRedirect(t *testing.T) {
	// Arrange
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	sourceService := sourcesvc.New(sourcesqlite.NewRepository(pool.DB))

	var requested []string
	client := outbound.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requested = append(requested, r.URL.String())
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://custom.example/feed"}},
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})})
	registry, err := typesvc.New(medium.Definition(client))
	require.NoError(t, err)
	sourceService.SetSourceTypes(registry)

	src, err := sourceService.CreateSource(ctx, sourceapi.CreateSourceInput{
		Name: "Alice", Type: string(sourceapi.SourceTypeMedium), Config: map[string]any{"kind": "profile", "identifier": "alice"},
	})
	require.NoError(t, err)

	h := newHarness()
	h.svc = service.New(h.repo, sourceService, registry, &fakeIngest{createdByLink: map[string]bool{}}, h.clock, service.DefaultRetryBackoff, service.DefaultScheduleInterval)
	job, err := h.svc.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)

	// Act
	_, err = h.svc.ExecuteFetchJob(ctx, job.ID)

	// Assert
	require.Error(t, err)
	require.Equal(t, []string{"https://medium.com/feed/@alice"}, requested)
}

type perSourceIngest struct {
	mu    sync.Mutex
	seen  map[string]bool
	calls []ingestCall
}

func (f *perSourceIngest) IngestEntry(_ context.Context, in entryapi.IngestEntryInput) (*entryapi.IngestEntryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, ingestCall{sourceID: in.SourceID, item: in})
	key := in.SourceID.String() + "\n" + in.Link
	created := !f.seen[key]
	f.seen[key] = true
	return &entryapi.IngestEntryResult{Created: created}, nil
}

func TestMediumAndRSSSameFeedURLHaveSeparateEntryIdentity(t *testing.T) {
	// Arrange
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	sourceService := sourcesvc.New(sourcesqlite.NewRepository(pool.DB))

	const feedURL = "https://medium.com/feed/alice"
	const article = "https://medium.com/alice/story"
	body := `<rss version="2.0"><channel><title>Alice</title><item><title>Story</title><link>` + article + `</link></item></channel></rss>`
	client := outbound.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, feedURL, r.URL.String())
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})})
	registry, err := typesvc.New(medium.Definition(client), rss.Definition(client))
	require.NoError(t, err)
	sourceService.SetSourceTypes(registry)

	mediumSrc, err := sourceService.CreateSource(ctx, sourceapi.CreateSourceInput{
		Name: "Alice Pub", Type: string(sourceapi.SourceTypeMedium), Config: map[string]any{"kind": "publication", "identifier": "alice"},
	})
	require.NoError(t, err)
	rssSrc, err := sourceService.CreateSource(ctx, sourceapi.CreateSourceInput{
		Name: "Alice RSS", Type: "rss", Config: map[string]any{"url": feedURL},
	})
	require.NoError(t, err)
	require.NotEqual(t, mediumSrc.ID, rssSrc.ID)

	ingest := &perSourceIngest{seen: map[string]bool{}}
	h := newHarness()
	h.svc = service.New(h.repo, sourceService, registry, ingest, h.clock, service.DefaultRetryBackoff, service.DefaultScheduleInterval)

	// Act
	for _, id := range []uuid.UUID{mediumSrc.ID, rssSrc.ID} {
		job, err := h.svc.CreateManualFetchJob(ctx, id)
		require.NoError(t, err)
		result, err := h.svc.ExecuteFetchJob(ctx, job.ID)
		require.NoError(t, err)
		require.Equal(t, 1, result.EntriesCreated)
	}

	// Assert
	require.Len(t, ingest.calls, 2)
	require.Equal(t, mediumSrc.ID, ingest.calls[0].sourceID)
	require.Equal(t, rssSrc.ID, ingest.calls[1].sourceID)
	require.Equal(t, article, ingest.calls[0].item.Link)
	require.Equal(t, article, ingest.calls[1].item.Link)

	// Act: repeat fetch refreshes within each Source without creating again
	for _, id := range []uuid.UUID{mediumSrc.ID, rssSrc.ID} {
		job, err := h.svc.CreateManualFetchJob(ctx, id)
		require.NoError(t, err)
		result, err := h.svc.ExecuteFetchJob(ctx, job.ID)
		require.NoError(t, err)
		require.Equal(t, 0, result.EntriesCreated)
	}

	// Assert
	require.Len(t, ingest.calls, 4)
}
