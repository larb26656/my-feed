package http_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/larb26656/my-feed/backend/internal/testkit/api"
	"github.com/stretchr/testify/require"
)

type entry struct {
	ID, SourceID, Title, Link, Description, CreatedAt, UpdatedAt string
	Thumbnail, PublishedAt, ReadAt, SavedAt                      *string
}

func rssSource(t *testing.T, router http.Handler) string {
	t.Helper()
	response := testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{
		"name": "Daily news", "type": "rss", "rssUrl": "https://example.test/rss",
	}, 201)
	return testkit.Decode[struct{ ID string }](t, response).ID
}

func TestIngestingTheSameArticleRefreshesMetadataAndPreservesReaderChoices(t *testing.T) {
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := map[string]any{"sourceId": sourceID, "title": "Original", "link": "https://example.test/article"}
	first := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))
	require.NotEmpty(t, first.ID)
	require.Equal(t, sourceID, first.SourceID)
	require.Nil(t, first.ReadAt)
	require.Nil(t, first.SavedAt)
	path := "/api/v1/entries/" + first.ID
	read := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/read", nil, 200))
	saved := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/save", nil, 200))
	require.NotNil(t, read.ReadAt)
	require.NotNil(t, saved.SavedAt)
	article["title"] = "Updated headline"
	article["description"] = "Updated summary"
	article["thumbnail"] = "https://example.test/photo.png"
	article["publishedAt"] = "2025-01-02T03:04:05Z"
	refreshed := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 200))
	require.Equal(t, first.ID, refreshed.ID, "ingestion must reuse the source/link identity")
	require.Equal(t, first.CreatedAt, refreshed.CreatedAt)
	require.Equal(t, read.ReadAt, refreshed.ReadAt)
	require.Equal(t, saved.SavedAt, refreshed.SavedAt)
	require.Equal(t, "Updated headline", refreshed.Title)
	require.Equal(t, "Updated summary", refreshed.Description)
	require.NotNil(t, refreshed.Thumbnail)
	require.Equal(t, article["thumbnail"], *refreshed.Thumbnail)
	require.NotNil(t, refreshed.PublishedAt)
	require.Equal(t, article["publishedAt"], *refreshed.PublishedAt)
	repeated := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 200))
	require.Equal(t, refreshed.ID, repeated.ID)
	got := testkit.Decode[entry](t, testkit.Request(t, router, "GET", path, nil, 200))
	require.Equal(t, repeated, got, "refresh and reader choices must survive a database round trip")
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))
	require.Len(t, page.Data, 1, "repeated ingestion must not duplicate an article")
}

func TestSavingALinkUsesTheManualSourceAndKeepsRSSArticlesSeparate(t *testing.T) {
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := map[string]any{"title": "Personal bookmark", "link": "https://example.test/shared"}
	manual := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", article, 201))
	article["title"] = "Renamed bookmark"
	refreshed := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", article, 201))
	require.Equal(t, manual.ID, refreshed.ID)
	require.Equal(t, "Renamed bookmark", refreshed.Title)
	article["sourceId"] = sourceID
	rss := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))
	require.NotEqual(t, manual.ID, rss.ID)
	require.NotEqual(t, sourceID, manual.SourceID)
	sources := testkit.Decode[[]struct{ ID, Type string }](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
	for _, source := range sources {
		if source.ID == manual.SourceID {
			require.Equal(t, "manual", source.Type)
			return
		}
	}
	t.Fatal("saved link must belong to the system manual source")
}

func TestReaderCanChangeReadAndSavedIndependently(t *testing.T) {
	router := api.New(t)
	savedLink := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))
	path := "/api/v1/entries/" + savedLink.ID
	saved := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/save", nil, 200))
	require.NotNil(t, saved.SavedAt)
	require.Nil(t, saved.ReadAt)
	read := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/read", nil, 200))
	require.NotNil(t, read.ReadAt)
	require.Equal(t, saved.SavedAt, read.SavedAt)
	count := testkit.Decode[struct{ Count int }](t, testkit.Request(t, router, "GET", "/api/v1/entries/saved-count", nil, 200))
	require.Equal(t, 1, count.Count)
	unread := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/clear-read", nil, 200))
	require.Nil(t, unread.ReadAt)
	require.Equal(t, saved.SavedAt, unread.SavedAt)
	cleared := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/clear-save", nil, 200))
	require.Nil(t, cleared.SavedAt)
	count = testkit.Decode[struct{ Count int }](t, testkit.Request(t, router, "GET", "/api/v1/entries/saved-count", nil, 200))
	require.Zero(t, count.Count)
}

func TestPaginationContinuesPastTiedTimestampsWithoutRepeatingOrSkippingArticles(t *testing.T) {
	router := api.New(t)
	sourceID := rssSource(t, router)
	var ids []string
	for i := 0; i < 3; i++ {
		item := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]any{
			"sourceId": sourceID, "title": fmt.Sprintf("Article %d", i), "link": fmt.Sprintf("https://example.test/%d", i), "publishedAt": "2025-01-01T00:00:00Z",
		}, 201))
		ids = append(ids, item.ID)
	}
	first := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=2", nil, 200))
	require.Len(t, first.Data, 2)
	require.True(t, first.Page.HasNext)
	require.NotNil(t, first.Page.NextCursor)
	// A newer arrival must not shift the next page as it would with offsets.
	testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]any{"sourceId": sourceID, "title": "Breaking news", "link": "https://example.test/new", "publishedAt": "2026-01-01T00:00:00Z"}, 201)
	second := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=2&cursor="+url.QueryEscape(*first.Page.NextCursor), nil, 200))
	require.Len(t, second.Data, 1)
	require.False(t, second.Page.HasNext)
	require.Nil(t, second.Page.NextCursor)
	require.ElementsMatch(t, ids, []string{first.Data[0].ID, first.Data[1].ID, second.Data[0].ID})
	require.Greater(t, first.Data[0].ID, first.Data[1].ID, "UUID breaks ties newest first")
	require.Greater(t, first.Data[1].ID, second.Data[0].ID)
	mismatch := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "GET", "/api/v1/entries/?savedOnly=true&cursor="+url.QueryEscape(*first.Page.NextCursor), nil, 400))
	require.Equal(t, "cursor does not match criteria", mismatch.Error)
}

func TestInvalidEntryRequestsReturnUsefulBusinessErrors(t *testing.T) {
	router := api.New(t)
	missing := uuid.NewString()
	for _, scenario := range []struct {
		name, method, path, body string
		status                   int
		code, message            string
	}{
		{"malformed JSON", "POST", "/api/v1/entries/", "{", 400, "validation", "invalid json body"},
		{"unknown field", "POST", "/api/v1/entries/save-link", `{"title":"A","link":"https://example.test/a","extra":true}`, 400, "validation", "invalid json body"},
		{"invalid source ID", "POST", "/api/v1/entries/", `{"sourceId":"bad"}`, 400, "validation", "invalid source id"},
		{"invalid publication time", "POST", "/api/v1/entries/save-link", `{"publishedAt":"yesterday"}`, 400, "validation", "invalid publishedAt"},
		{"missing source", "POST", "/api/v1/entries/", fmt.Sprintf(`{"sourceId":%q,"title":"A","link":"https://example.test/a"}`, missing), 404, "not_found", "source not found"},
		{"invalid entry ID", "GET", "/api/v1/entries/bad", "", 400, "validation", "invalid entry id"},
		{"missing entry", "GET", "/api/v1/entries/" + missing, "", 404, "not_found", "entry not found"},
		{"invalid cursor", "GET", "/api/v1/entries/?cursor=bad", "", 400, "validation", "invalid cursor"},
		{"invalid page size", "GET", "/api/v1/entries/?pageSize=many", "", 400, "validation", "invalid pageSize"},
		{"invalid unread filter", "GET", "/api/v1/entries/?unread=yes", "", 400, "validation", "invalid unread"},
		{"invalid saved filter", "GET", "/api/v1/entries/?savedOnly=yes", "", 400, "validation", "invalid savedOnly"},
		{"invalid source filter", "GET", "/api/v1/entries/?sourceId=bad", "", 400, "validation", "invalid source id"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, scenario.method, scenario.path, scenario.body, scenario.status))
			require.Equal(t, scenario.code, failure.Code)
			require.Equal(t, scenario.message, failure.Error)
		})
	}
	for _, action := range []string{"read", "save", "clear-read", "clear-save"} {
		t.Run("missing article cannot "+action, func(t *testing.T) {
			failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "POST", "/api/v1/entries/"+missing+"/"+action, nil, 404))
			require.Equal(t, "not_found", failure.Code)
		})
	}
}

func TestIngestionRequiresAnActiveRSSSourceAndValidMetadata(t *testing.T) {
	router := api.New(t)
	rssID := rssSource(t, router)
	sources := testkit.Decode[[]struct{ ID, Type string }](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
	var manualID string
	for _, source := range sources {
		if source.Type == "manual" {
			manualID = source.ID
		}
	}
	require.NotEmpty(t, manualID)
	for _, scenario := range []struct {
		name, sourceID, title, link, code string
		status                            int
	}{
		{"manual sources reject ingestion", manualID, "News", "https://example.test/a", "validation", 400},
		{"blank titles are rejected", rssID, " ", "https://example.test/a", "validation", 400},
		{"blank links are rejected", rssID, "News", " ", "validation", 400},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": scenario.sourceID, "title": scenario.title, "link": scenario.link}, scenario.status))
			require.Equal(t, scenario.code, failure.Code)
		})
	}
	article := map[string]string{"sourceId": rssID, "title": "Existing news", "link": "https://example.test/existing"}
	created := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))
	testkit.Request(t, router, "POST", "/api/v1/sources/"+rssID+"/deactivate", nil, 200)
	failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 422))
	require.Equal(t, "failed_precondition", failure.Code)
	require.Equal(t, "cannot ingest into a deactivated source", failure.Error)
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))
	require.Len(t, page.Data, 1, "pausing a source must not hide existing articles")
	require.Equal(t, created.ID, page.Data[0].ID)
}

func TestReaderCanFilterBySourceUnreadAndSavedTogether(t *testing.T) {
	router := api.New(t)
	rssID := rssSource(t, router)
	var entries []entry
	for i := 0; i < 3; i++ {
		item := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": rssID, "title": "News", "link": fmt.Sprintf("https://example.test/%d", i)}, 201))
		entries = append(entries, item)
	}
	for _, item := range entries[:2] {
		testkit.Request(t, router, "POST", "/api/v1/entries/"+item.ID+"/save", nil, 200)
	}
	testkit.Request(t, router, "POST", "/api/v1/entries/"+entries[0].ID+"/read", nil, 200)
	manual := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Bookmark", "link": "https://example.test/manual"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/entries/"+manual.ID+"/save", nil, 200)
	for _, scenario := range []struct {
		name, query string
		ids         []string
	}{
		{"source only", "sourceId=" + rssID, []string{entries[0].ID, entries[1].ID, entries[2].ID}},
		{"unread saved from source", "sourceId=" + rssID + "&unread=true&savedOnly=true", []string{entries[1].ID}},
		{"saved across sources", "savedOnly=true", []string{entries[0].ID, entries[1].ID, manual.ID}},
		{"unknown source is empty", "sourceId=" + uuid.NewString(), []string{}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?"+scenario.query, nil, 200))
			ids := make([]string, 0, len(page.Data))
			for _, item := range page.Data {
				ids = append(ids, item.ID)
			}
			require.ElementsMatch(t, scenario.ids, ids)
			require.False(t, page.Page.HasNext)
			require.NotNil(t, page.Data, "empty results must serialize as an array")
		})
	}
}

func TestOmittedOrNonpositivePageSizeUsesTwentyArticles(t *testing.T) {
	router := api.New(t)
	for i := 0; i < 21; i++ {
		testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": fmt.Sprintf("https://example.test/%d", i)}, 201)
	}
	for _, query := range []string{"", "?pageSize=0", "?pageSize=-1"} {
		t.Run(query, func(t *testing.T) {
			page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/"+query, nil, 200))
			require.Len(t, page.Data, 20)
			require.True(t, page.Page.HasNext)
		})
	}
}

func TestEntriesWithoutPublicationDatesUseTheirCreationDateForOrdering(t *testing.T) {
	router := api.New(t)
	sourceID := rssSource(t, router)
	published := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{
		"sourceId": sourceID, "title": "Older news", "link": "https://example.test/old", "publishedAt": "2000-01-01T00:00:00Z",
	}, 201))
	undated := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Just added", "link": "https://example.test/new"}, 201))
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))
	require.Len(t, page.Data, 2)
	require.Equal(t, undated.ID, page.Data[0].ID)
	require.Equal(t, published.ID, page.Data[1].ID)
}
