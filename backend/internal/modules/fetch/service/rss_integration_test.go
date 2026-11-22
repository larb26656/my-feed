package service_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	sourcedomain "github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	rss "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/youtube"
	typesvc "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	"github.com/larb26656/my-feed/backend/internal/platform/outbound"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/larb26656/my-feed/backend/internal/testkit/customtype"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFetchJobLoadsStoredConfigAndRejectsPrivateRedirect(t *testing.T) {
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	repo := sourcesqlite.NewRepository(pool.DB)
	src, err := sourcedomain.NewRSSSource("Feed", "https://old.example/feed")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(ctx, src))
	requests := []string{}
	client := outbound.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.URL.String())
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"http://127.0.0.1/internal"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})})
	registry, err := typesvc.New(rss.Definition(client))
	require.NoError(t, err)
	h := newHarness()
	sourceReader := sourcesvc.New(repo)
	h.svc = service.New(h.repo, sourceReader, registry, &fakeIngest{createdByLink: map[string]bool{}}, h.clock, service.DefaultRetryBackoff, service.DefaultScheduleInterval)
	job, err := h.svc.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)
	_, err = pool.Exec(`UPDATE sources SET config = ? WHERE id = ?`, `{"url":"https://new.example/feed"}`, src.ID.String())
	require.NoError(t, err)
	result, err := h.svc.ExecuteFetchJob(ctx, job.ID)
	require.Error(t, err)
	require.Equal(t, 1, result.Attempt)
	require.Equal(t, []string{"https://new.example/feed"}, requests)
}

func TestFetchJobRecordsEntriesFromStoredRSSConfig(t *testing.T) {
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	repo := sourcesqlite.NewRepository(pool.DB)
	src, err := sourcedomain.NewRSSSource("Feed", "https://old.example/feed")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(ctx, src))
	var requested string
	client := outbound.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requested = r.URL.String()
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`<rss version="2.0"><channel><title>Feed</title><item><title>Entry</title><link>https://example.test/entry</link></item></channel></rss>`))}, nil
	})})
	registry, err := typesvc.New(rss.Definition(client))
	require.NoError(t, err)
	h := newHarness()
	ingest := &fakeIngest{createdByLink: map[string]bool{"https://example.test/entry": true}}
	h.svc = service.New(h.repo, sourcesvc.New(repo), registry, ingest, h.clock, service.DefaultRetryBackoff, service.DefaultScheduleInterval)
	job, err := h.svc.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)
	_, err = pool.Exec(`UPDATE sources SET config = ? WHERE id = ?`, `{"url":"https://new.example/feed"}`, src.ID.String())
	require.NoError(t, err)
	result, err := h.svc.ExecuteFetchJob(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, "https://new.example/feed", requested)
	require.Len(t, ingest.calls, 1)
	require.Equal(t, "Entry", ingest.calls[0].item.Title)
	require.Equal(t, 1, result.EntriesCreated)
}

func TestFetchJobObtainsYouTubeEntriesThroughRegistry(t *testing.T) {
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	repo := sourcesqlite.NewRepository(pool.DB)
	src, err := sourcedomain.NewYouTubeSource("Channel", "UC1234567890123456789012")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(ctx, src))
	var requested string
	client := outbound.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requested = r.URL.String()
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015"><entry><yt:videoId>video1</yt:videoId><title>Video</title></entry></feed>`))}, nil
	})})
	registry, err := typesvc.New(youtube.Definition(client))
	require.NoError(t, err)
	h := newHarness()
	ingest := &fakeIngest{createdByLink: map[string]bool{"https://www.youtube.com/watch?v=video1": true}}
	h.svc = service.New(h.repo, sourcesvc.New(repo), registry, ingest, h.clock, service.DefaultRetryBackoff, service.DefaultScheduleInterval)
	job, err := h.svc.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)
	result, err := h.svc.ExecuteFetchJob(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, "https://www.youtube.com/feeds/videos.xml?channel_id=UC1234567890123456789012", requested)
	require.Len(t, ingest.calls, 1)
	require.Equal(t, "Video", ingest.calls[0].item.Title)
	require.Equal(t, 1, result.EntriesCreated)
}

func TestFetchAttemptsUseConfigLoadedAtTheirStart(t *testing.T) {
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	repo := sourcesqlite.NewRepository(pool.DB)
	src, err := sourcedomain.NewRSSSource("Feed", "https://old.example/feed")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(ctx, src))
	sourceService := sourcesvc.New(repo)
	var requested []string
	client := outbound.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requested = append(requested, r.URL.String())
		if len(requested) == 1 {
			_, err := sourceService.UpdateSource(ctx, src.ID, sourcesvc.UpdateSourceInput{Name: "Feed", Config: map[string]any{"url": "https://new.example/feed"}})
			require.NoError(t, err)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`<rss version="2.0"><channel><title>Feed</title></channel></rss>`))}, nil
	})})
	registry, err := typesvc.New(rss.Definition(client))
	require.NoError(t, err)
	sourceService.SetSourceTypes(registry)
	h := newHarness()
	h.svc = service.New(h.repo, sourceService, registry, &fakeIngest{createdByLink: map[string]bool{}}, h.clock, service.DefaultRetryBackoff, service.DefaultScheduleInterval)
	for range 2 {
		job, err := h.svc.CreateManualFetchJob(ctx, src.ID)
		require.NoError(t, err)
		_, err = h.svc.ExecuteFetchJob(ctx, job.ID)
		require.NoError(t, err)
	}
	require.Equal(t, []string{"https://old.example/feed", "https://new.example/feed"}, requested)
}

func TestFetchJobObtainsRegisteredTypeConfig(t *testing.T) {
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	sourceService := sourcesvc.New(sourcesqlite.NewRepository(pool.DB))
	definition := customtype.Definition(t)
	registry, err := typesvc.New(definition)
	require.NoError(t, err)
	sourceService.SetSourceTypes(registry)
	src, err := sourceService.CreateSource(ctx, sourceapi.CreateSourceInput{Name: "Custom", Type: "custom", Config: customtype.Input("ok-alpha")})
	require.NoError(t, err)
	h := newHarness()
	ingest := &fakeIngest{createdByLink: map[string]bool{"https://example.test/custom": true}}
	h.svc = service.New(h.repo, sourceService, registry, ingest, h.clock, service.DefaultRetryBackoff, service.DefaultScheduleInterval)
	job, err := h.svc.CreateManualFetchJob(ctx, src.ID)
	require.NoError(t, err)
	_, err = h.svc.ExecuteFetchJob(ctx, job.ID)
	require.NoError(t, err)
	require.Len(t, ingest.calls, 1)
	require.Equal(t, "ok-alpha", ingest.calls[0].item.Title)
}
