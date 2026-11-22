package service_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	entrysqlite "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/sqlite"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/repository"
	entrysvc "github.com/larb26656/my-feed/backend/internal/modules/entry/service"
	fetchsqlite "github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/sqlite"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	sourcedomain "github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/bluesky"
	typesvc "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
	"github.com/larb26656/my-feed/backend/internal/platform/outbound"
	"github.com/larb26656/my-feed/backend/internal/testkit"
)

const blueskyDID = "did:plc:z72i7hdynmk6r22z27h6tvur"

func blueskyFeedJSON(posts ...string) string {
	if len(posts) == 0 {
		return `{"feed":[]}`
	}
	return `{"feed":[` + strings.Join(posts, ",") + `]}`
}

func blueskyPost(rkey, text, createdAt string) string {
	return `{
      "post": {
        "uri": "at://` + blueskyDID + `/app.bsky.feed.post/` + rkey + `",
        "author": {"did": "` + blueskyDID + `", "handle": "alice.bsky.social"},
        "record": {
          "$type": "app.bsky.feed.post",
          "text": ` + strconv.Quote(text) + `,
          "createdAt": "` + createdAt + `"
        }
      }
    }`
}

type blueskyHarness struct {
	ctx           context.Context
	pool          *database.Pool
	sources       *sourcesvc.Service
	entries       *entrysvc.Service
	fetch         *service.Service
	clock         *fakeClock
	requests      []string
	feedResponses []string
	feedStatus    int
	feedErr       error
}

func newBlueskyHarness(t *testing.T) *blueskyHarness {
	t.Helper()
	ctx := context.Background()
	pool := testkit.NewDatabase(t)
	sourceRepo := sourcesqlite.NewRepository(pool.DB)
	sources := sourcesvc.New(sourceRepo)
	h := &blueskyHarness{
		ctx:        ctx,
		pool:       pool,
		sources:    sources,
		feedStatus: http.StatusOK,
	}
	client := outbound.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		h.requests = append(h.requests, r.URL.String())
		if h.feedErr != nil {
			return nil, h.feedErr
		}
		status := h.feedStatus
		if status == 0 {
			status = http.StatusOK
		}
		body := `{"feed":[]}`
		if len(h.feedResponses) > 0 {
			body = h.feedResponses[0]
			if len(h.feedResponses) > 1 {
				h.feedResponses = h.feedResponses[1:]
			}
		}
		return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
	})})
	registry, err := typesvc.New(bluesky.Definition(client))
	require.NoError(t, err)
	sources.SetSourceTypes(registry)
	entryRepo := entrysqlite.NewRepository(pool.DB)
	h.entries = entrysvc.New(entryRepo, sources)
	h.clock = &fakeClock{now: time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)}
	h.fetch = service.New(
		fetchsqlite.NewRepository(pool.DB),
		sources,
		registry,
		entrysvc.NewIngester(h.entries),
		h.clock,
		service.DefaultRetryBackoff,
		service.DefaultScheduleInterval,
	)
	return h
}

func (h *blueskyHarness) createSource(t *testing.T) *sourcedomain.Source {
	t.Helper()
	src, err := sourcedomain.NewFetchableSource("Alice", sourcedomain.TypeBluesky, map[string]any{"did": blueskyDID}, blueskyDID)
	require.NoError(t, err)
	require.NoError(t, sourcesqlite.NewRepository(h.pool.DB).CreateSource(h.ctx, src))
	return src
}

func TestFetchJobObtainsBlueskyEntriesThroughRegistry(t *testing.T) {
	// Arrange
	h := newBlueskyHarness(t)
	src := h.createSource(t)
	h.feedResponses = []string{blueskyFeedJSON(
		blueskyPost("3quote", "Quoted take\nSecond line", "2026-01-01T12:00:00.000Z"),
		blueskyPost("3original", "Hello Bluesky", "2026-01-01T00:00:00.000Z"),
	)}
	job, err := h.fetch.CreateManualFetchJob(h.ctx, src.ID)
	require.NoError(t, err)

	// Act
	result, err := h.fetch.ExecuteFetchJob(h.ctx, job.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, result.Status)
	require.Equal(t, 2, result.EntriesCreated)
	require.Len(t, h.requests, 1)
	requested, err := url.Parse(h.requests[0])
	require.NoError(t, err)
	require.Equal(t, "/xrpc/app.bsky.feed.getAuthorFeed", requested.Path)
	require.Equal(t, blueskyDID, requested.Query().Get("actor"))
	require.Equal(t, "50", requested.Query().Get("limit"))

	page, err := h.entries.ListEntries(h.ctx, entrysvc.ListEntriesInput{
		Criteria: repository.EntryListCriteria{SourceID: &src.ID},
	})
	require.NoError(t, err)
	require.Len(t, page.Entries, 2)
	byLink := map[string]string{}
	for _, entry := range page.Entries {
		byLink[entry.Link] = entry.Title
	}
	require.Equal(t, "Hello Bluesky", byLink["https://bsky.app/profile/"+blueskyDID+"/post/3original"])
	require.Equal(t, "Quoted take", byLink["https://bsky.app/profile/"+blueskyDID+"/post/3quote"])
}

func TestFetchJobPreservesBlueskyEntriesAcrossOverlapAndHandleChange(t *testing.T) {
	// Arrange: first fetch creates Entries; Reader marks one read and saved.
	h := newBlueskyHarness(t)
	src := h.createSource(t)
	h.feedResponses = []string{blueskyFeedJSON(
		blueskyPost("3original", "Hello Bluesky", "2026-01-01T00:00:00.000Z"),
	)}
	job1, err := h.fetch.CreateManualFetchJob(h.ctx, src.ID)
	require.NoError(t, err)
	first, err := h.fetch.ExecuteFetchJob(h.ctx, job1.ID)
	require.NoError(t, err)
	require.Equal(t, 1, first.EntriesCreated)

	page, err := h.entries.ListEntries(h.ctx, entrysvc.ListEntriesInput{
		Criteria: repository.EntryListCriteria{SourceID: &src.ID},
	})
	require.NoError(t, err)
	require.Len(t, page.Entries, 1)
	original := page.Entries[0]
	_, err = h.entries.MarkEntryRead(h.ctx, original.ID)
	require.NoError(t, err)
	_, err = h.entries.MarkEntrySaved(h.ctx, original.ID)
	require.NoError(t, err)

	// Act: overlapping re-fetch after handle change still uses stored DID.
	h.requests = nil
	h.feedResponses = []string{blueskyFeedJSON(
		blueskyPost("3newer", "Fresh post", "2026-01-02T00:00:00.000Z"),
		blueskyPost("3original", "Hello Bluesky (edited title)", "2026-01-01T00:00:00.000Z"),
	)}
	job2, err := h.fetch.CreateManualFetchJob(h.ctx, src.ID)
	require.NoError(t, err)
	second, err := h.fetch.ExecuteFetchJob(h.ctx, job2.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, 1, second.EntriesCreated)
	require.Len(t, h.requests, 1)
	requested, err := url.Parse(h.requests[0])
	require.NoError(t, err)
	require.Equal(t, blueskyDID, requested.Query().Get("actor"))

	page, err = h.entries.ListEntries(h.ctx, entrysvc.ListEntriesInput{
		Criteria: repository.EntryListCriteria{SourceID: &src.ID},
	})
	require.NoError(t, err)
	require.Len(t, page.Entries, 2)
	found := false
	for _, entry := range page.Entries {
		if entry.Link != original.Link {
			continue
		}
		found = true
		require.Equal(t, original.ID, entry.ID)
		require.Equal(t, "Hello Bluesky (edited title)", entry.Title)
		require.NotNil(t, entry.ReadAt)
		require.NotNil(t, entry.SavedAt)
	}
	require.True(t, found)
}

func TestFetchJobSucceedsWithZeroEligibleBlueskyEntries(t *testing.T) {
	// Arrange
	h := newBlueskyHarness(t)
	src := h.createSource(t)
	h.feedResponses = []string{`{"feed":[{
      "post": {
        "uri": "at://` + blueskyDID + `/app.bsky.feed.post/reply1",
        "author": {"did": "` + blueskyDID + `", "handle": "alice.bsky.social"},
        "record": {"$type": "app.bsky.feed.post", "text": "reply", "createdAt": "2026-01-01T00:00:00.000Z", "reply": {"root": {"uri": "at://x"}, "parent": {"uri": "at://x"}}}
      },
      "reply": {"root": {"uri": "at://x"}, "parent": {"uri": "at://x"}}
    }]}`}
	job, err := h.fetch.CreateManualFetchJob(h.ctx, src.ID)
	require.NoError(t, err)

	// Act
	result, err := h.fetch.ExecuteFetchJob(h.ctx, job.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, result.Status)
	require.Equal(t, 0, result.EntriesCreated)
	page, err := h.entries.ListEntries(h.ctx, entrysvc.ListEntriesInput{
		Criteria: repository.EntryListCriteria{SourceID: &src.ID},
	})
	require.NoError(t, err)
	require.Empty(t, page.Entries)
}

func TestFetchJobRetriesBlueskyUpstreamFailure(t *testing.T) {
	// Arrange
	h := newBlueskyHarness(t)
	src := h.createSource(t)
	h.feedStatus = http.StatusServiceUnavailable
	job, err := h.fetch.CreateManualFetchJob(h.ctx, src.ID)
	require.NoError(t, err)

	// Act
	result, err := h.fetch.ExecuteFetchJob(h.ctx, job.ID)

	// Assert
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
	require.Equal(t, domain.StatusPending, result.Status)
	require.Equal(t, 1, result.Attempt)
	require.False(t, result.NextAttemptAt.IsZero())
}

func TestScheduleRoundCreatesFetchJobForActiveBlueskySource(t *testing.T) {
	// Arrange
	h := newBlueskyHarness(t)
	src := h.createSource(t)

	// Act
	created, err := h.fetch.ScheduleRound(h.ctx)

	// Assert
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.Equal(t, src.ID, created[0].SourceID)
	require.Equal(t, domain.StatusPending, created[0].Status)
}

func TestDeactivatedBlueskySourceReceivesNoNewFetchJobs(t *testing.T) {
	// Arrange: fetch once so Entries exist, then deactivate.
	h := newBlueskyHarness(t)
	src := h.createSource(t)
	h.feedResponses = []string{blueskyFeedJSON(
		blueskyPost("3original", "Hello Bluesky", "2026-01-01T00:00:00.000Z"),
	)}
	job, err := h.fetch.CreateManualFetchJob(h.ctx, src.ID)
	require.NoError(t, err)
	_, err = h.fetch.ExecuteFetchJob(h.ctx, job.ID)
	require.NoError(t, err)
	_, err = h.sources.DeactivateSource(h.ctx, src.ID)
	require.NoError(t, err)

	// Act
	_, err = h.fetch.CreateManualFetchJob(h.ctx, src.ID)
	scheduled, scheduleErr := h.fetch.ScheduleRound(h.ctx)

	// Assert
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
	require.NoError(t, scheduleErr)
	require.Empty(t, scheduled)

	page, err := h.entries.ListEntries(h.ctx, entrysvc.ListEntriesInput{
		Criteria: repository.EntryListCriteria{SourceID: &src.ID},
	})
	require.NoError(t, err)
	require.Len(t, page.Entries, 1)
	require.Equal(t, "Hello Bluesky", page.Entries[0].Title)
}
