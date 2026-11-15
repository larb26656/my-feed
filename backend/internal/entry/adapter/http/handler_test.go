package http_test

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
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

func TestIngestingANewArticleReturnsItsIdentityAndMetadata(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/article"}

	// Act
	created := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))

	// Assert
	require.NotEmpty(t, created.ID)
	require.Equal(t, sourceID, created.SourceID)
	require.Equal(t, article["title"], created.Title)
	require.Equal(t, article["link"], created.Link)
	require.Nil(t, created.ReadAt)
	require.Nil(t, created.SavedAt)
}

func TestIngestingTheSameArticleDoesNotCreateADuplicate(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/article"}
	first := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))

	// Act
	repeated := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 200))
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))

	// Assert
	require.Equal(t, first.ID, repeated.ID)
	require.Equal(t, first.CreatedAt, repeated.CreatedAt)
	require.Len(t, page.Data, 1)
	require.Equal(t, first.ID, page.Data[0].ID)
}

func TestReingestionPersistsRefreshedMetadata(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := map[string]any{"sourceId": sourceID, "title": "Original", "link": "https://example.test/article"}
	first := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))
	article["title"] = "Updated headline"
	article["description"] = "Updated summary"
	article["thumbnail"] = "https://example.test/photo.png"
	article["publishedAt"] = "2025-01-02T03:04:05Z"

	// Act
	refreshed := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 200))
	got := testkit.Decode[entry](t, testkit.Request(t, router, "GET", "/api/v1/entries/"+first.ID, nil, 200))

	// Assert
	require.Equal(t, first.ID, refreshed.ID)
	require.Equal(t, first.CreatedAt, refreshed.CreatedAt)
	require.Equal(t, "Updated headline", got.Title)
	require.Equal(t, "Updated summary", got.Description)
	require.NotNil(t, got.Thumbnail)
	require.Equal(t, article["thumbnail"], *got.Thumbnail)
	require.NotNil(t, got.PublishedAt)
	require.Equal(t, article["publishedAt"], *got.PublishedAt)
	require.Equal(t, refreshed, got)
}

func TestReingestionPreservesReadAndSavedChoices(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := map[string]string{"sourceId": sourceID, "title": "Original", "link": "https://example.test/article"}
	first := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))
	path := "/api/v1/entries/" + first.ID
	read := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/read", nil, 200))
	saved := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/save", nil, 200))
	require.NotNil(t, read.ReadAt)
	require.NotNil(t, saved.SavedAt)
	article["title"] = "Updated headline"

	// Act
	testkit.Request(t, router, "POST", "/api/v1/entries/", article, 200)
	got := testkit.Decode[entry](t, testkit.Request(t, router, "GET", path, nil, 200))

	// Assert
	require.Equal(t, read.ReadAt, got.ReadAt)
	require.Equal(t, saved.SavedAt, got.SavedAt)
}

func TestSavingALinkAssignsTheSystemManualSource(t *testing.T) {
	// Arrange
	router := api.New(t)
	sources := testkit.Decode[[]struct{ ID, Type string }](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
	require.Len(t, sources, 1)
	require.Equal(t, "manual", sources[0].Type)

	// Act
	saved := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Bookmark", "link": "https://example.test/article"}, 201))

	// Assert
	require.Equal(t, sources[0].ID, saved.SourceID)
}

func TestSavingTheSameLinkRefreshesTheExistingBookmark(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := map[string]string{"title": "Original", "link": "https://example.test/article"}
	first := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", article, 201))
	article["title"] = "Updated bookmark"

	// Act
	refreshed := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", article, 201))

	// Assert
	require.Equal(t, first.ID, refreshed.ID)
	require.Equal(t, "Updated bookmark", refreshed.Title)
}

func TestTheSameLinkUnderManualAndRSSSourcesHasSeparateIdentities(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := map[string]string{"title": "Bookmark", "link": "https://example.test/shared"}
	manual := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", article, 201))
	article["sourceId"] = sourceID

	// Act
	rss := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))

	// Assert
	require.NotEqual(t, manual.ID, rss.ID)
	require.Equal(t, sourceID, rss.SourceID)
	require.NotEqual(t, manual.SourceID, rss.SourceID)
}

func TestSavingAnUnreadArticleLeavesItUnread(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))

	// Act
	saved := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/"+article.ID+"/save", nil, 200))

	// Assert
	require.NotNil(t, saved.SavedAt)
	require.Nil(t, saved.ReadAt)
}

func TestMarkingASavedArticleReadPreservesItsSavedChoice(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))
	path := "/api/v1/entries/" + article.ID
	saved := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/save", nil, 200))
	require.NotNil(t, saved.SavedAt)

	// Act
	read := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/read", nil, 200))

	// Assert
	require.NotNil(t, read.ReadAt)
	require.Equal(t, saved.SavedAt, read.SavedAt)
}

func TestClearingReadPreservesTheSavedChoice(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))
	path := "/api/v1/entries/" + article.ID
	testkit.Request(t, router, "POST", path+"/read", nil, 200)
	saved := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/save", nil, 200))
	require.NotNil(t, saved.ReadAt)
	require.NotNil(t, saved.SavedAt)

	// Act
	unread := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/clear-read", nil, 200))

	// Assert
	require.Nil(t, unread.ReadAt)
	require.Equal(t, saved.SavedAt, unread.SavedAt)
}

func TestClearingSavedPreservesTheReadChoice(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))
	path := "/api/v1/entries/" + article.ID
	testkit.Request(t, router, "POST", path+"/save", nil, 200)
	read := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/read", nil, 200))
	require.NotNil(t, read.ReadAt)
	require.NotNil(t, read.SavedAt)

	// Act
	cleared := testkit.Decode[entry](t, testkit.Request(t, router, "POST", path+"/clear-save", nil, 200))

	// Assert
	require.Nil(t, cleared.SavedAt)
	require.Equal(t, read.ReadAt, cleared.ReadAt)
}

func TestSavedCountIncludesOnlyCurrentlySavedArticles(t *testing.T) {
	for _, scenario := range []struct {
		name                    string
		hasArticle, save, clear bool
		count                   int
	}{
		{name: "empty library", count: 0}, {name: "unsaved article", hasArticle: true, count: 0},
		{name: "saved article", hasArticle: true, save: true, count: 1}, {name: "cleared bookmark", hasArticle: true, save: true, clear: true, count: 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			if scenario.hasArticle {
				article := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))
				path := "/api/v1/entries/" + article.ID
				if scenario.save {
					testkit.Request(t, router, "POST", path+"/save", nil, 200)
				}
				if scenario.clear {
					testkit.Request(t, router, "POST", path+"/clear-save", nil, 200)
				}
			}

			// Act
			count := testkit.Decode[struct{ Count int }](t, testkit.Request(t, router, "GET", "/api/v1/entries/saved-count", nil, 200))

			// Assert
			require.Equal(t, scenario.count, count.Count)
		})
	}
}

func TestEntriesWithTheSamePublicationTimeAreSortedByDescendingID(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	var ids []string
	for i := 0; i < 3; i++ {
		article := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": fmt.Sprintf("https://example.test/%d", i), "publishedAt": "2025-01-01T00:00:00Z"}, 201))
		ids = append(ids, article.ID)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))

	// Act
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))

	// Assert
	require.Len(t, page.Data, 3)
	require.Equal(t, ids, []string{page.Data[0].ID, page.Data[1].ID, page.Data[2].ID})
}

func TestPaginationContinuesPastTiedTimestampsWithoutRepeatingOrSkippingArticles(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	var ids []string
	for i := 0; i < 3; i++ {
		article := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": fmt.Sprintf("https://example.test/%d", i), "publishedAt": "2025-01-01T00:00:00Z"}, 201))
		ids = append(ids, article.ID)
	}
	first := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=2", nil, 200))
	require.Len(t, first.Data, 2)
	require.True(t, first.Page.HasNext)
	require.NotNil(t, first.Page.NextCursor)

	// Act
	second := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=2&cursor="+url.QueryEscape(*first.Page.NextCursor), nil, 200))

	// Assert
	require.Len(t, second.Data, 1)
	require.False(t, second.Page.HasNext)
	require.Nil(t, second.Page.NextCursor)
	require.ElementsMatch(t, ids, []string{first.Data[0].ID, first.Data[1].ID, second.Data[0].ID})
}

func TestANewerArrivalDoesNotShiftTheNextPage(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	older := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "Older", "link": "https://example.test/older", "publishedAt": "2024-01-01T00:00:00Z"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "Newer", "link": "https://example.test/newer", "publishedAt": "2025-01-01T00:00:00Z"}, 201)
	first := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=1", nil, 200))
	require.NotNil(t, first.Page.NextCursor)
	testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "Breaking", "link": "https://example.test/breaking", "publishedAt": "2026-01-01T00:00:00Z"}, 201)

	// Act
	second := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=1&cursor="+url.QueryEscape(*first.Page.NextCursor), nil, 200))

	// Assert
	require.Len(t, second.Data, 1)
	require.Equal(t, older.ID, second.Data[0].ID)
	require.False(t, second.Page.HasNext)
}

func TestACursorCannotBeReusedWithDifferentFilters(t *testing.T) {
	// Arrange
	router := api.New(t)
	for i := 0; i < 2; i++ {
		testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": fmt.Sprintf("https://example.test/%d", i)}, 201)
	}
	first := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=1", nil, 200))
	require.NotNil(t, first.Page.NextCursor)

	// Act
	failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "GET", "/api/v1/entries/?savedOnly=true&cursor="+url.QueryEscape(*first.Page.NextCursor), nil, 400))

	// Assert
	require.Equal(t, "validation", failure.Code)
	require.Equal(t, "cursor does not match criteria", failure.Error)
}

func TestInvalidEntryRequestsReturnUsefulBusinessErrors(t *testing.T) {
	// Arrange: immutable request data.
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
			// Arrange
			router := api.New(t)

			// Act: submit the request that should be rejected.
			failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, scenario.method, scenario.path, scenario.body, scenario.status))

			// Assert
			require.Equal(t, scenario.code, failure.Code)
			require.Equal(t, scenario.message, failure.Error)
		})
	}
}

func TestReaderActionsOnMissingArticlesReturnNotFound(t *testing.T) {
	// Arrange: immutable request data.
	missing := uuid.NewString()
	for _, action := range []string{"read", "save", "clear-read", "clear-save"} {
		t.Run("missing article cannot "+action, func(t *testing.T) {
			// Arrange
			router := api.New(t)

			// Act: submit the request that should be rejected.
			failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "POST", "/api/v1/entries/"+missing+"/"+action, nil, 404))

			// Assert
			require.Equal(t, "not_found", failure.Code)
		})
	}
}

func TestIngestionRejectsManualSources(t *testing.T) {
	// Arrange
	router := api.New(t)
	sources := testkit.Decode[[]struct{ ID string }](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
	require.Len(t, sources, 1)

	// Act
	failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sources[0].ID, "title": "News", "link": "https://example.test/a"}, 400))

	// Assert
	require.Equal(t, "validation", failure.Code)
	require.Equal(t, "ingest can only record entries into rss sources", failure.Error)
}

func TestIngestionRejectsBlankMetadata(t *testing.T) {
	for _, scenario := range []struct{ name, title, link string }{
		{"blank title", " ", "https://example.test/a"}, {"blank link", "News", " "},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			sourceID := rssSource(t, router)

			// Act
			failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": scenario.title, "link": scenario.link}, 400))

			// Assert
			require.Equal(t, "validation", failure.Code)
		})
	}
}

func TestIngestionRejectsPausedRSSSources(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	testkit.Request(t, router, "POST", "/api/v1/sources/"+sourceID+"/deactivate", nil, 200)

	// Act
	failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/a"}, 422))

	// Assert
	require.Equal(t, "failed_precondition", failure.Code)
	require.Equal(t, "cannot ingest into a deactivated source", failure.Error)
}

func TestPausingASourceDoesNotHideItsExistingArticles(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/a"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/sources/"+sourceID+"/deactivate", nil, 200)

	// Act
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))

	// Assert
	require.Len(t, page.Data, 1)
	require.Equal(t, article.ID, page.Data[0].ID)
}

func TestSourceFilterReturnsOnlyArticlesFromThatSource(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	rss := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/rss-article"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Bookmark", "link": "https://example.test/manual"}, 201)

	// Act
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?sourceId="+sourceID, nil, 200))

	// Assert
	require.Len(t, page.Data, 1)
	require.Equal(t, rss.ID, page.Data[0].ID)
}

func TestUnreadAndSavedFiltersCanBeCombinedWithASourceFilter(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	var articles []entry
	for i := 0; i < 3; i++ {
		article := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": fmt.Sprintf("https://example.test/%d", i)}, 201))
		articles = append(articles, article)
	}
	for _, article := range articles[:2] {
		testkit.Request(t, router, "POST", "/api/v1/entries/"+article.ID+"/save", nil, 200)
	}
	testkit.Request(t, router, "POST", "/api/v1/entries/"+articles[0].ID+"/read", nil, 200)
	manual := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Bookmark", "link": "https://example.test/manual"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/entries/"+manual.ID+"/save", nil, 200)

	// Act
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?sourceId="+sourceID+"&unread=true&savedOnly=true", nil, 200))

	// Assert
	require.Len(t, page.Data, 1)
	require.Equal(t, articles[1].ID, page.Data[0].ID)
}

func TestSavedFilterIncludesBookmarksAcrossSources(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	rss := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/rss-article"}, 201))
	manual := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Bookmark", "link": "https://example.test/manual"}, 201))
	for _, article := range []entry{rss, manual} {
		testkit.Request(t, router, "POST", "/api/v1/entries/"+article.ID+"/save", nil, 200)
	}
	testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Unsaved", "link": "https://example.test/unsaved"}, 201)

	// Act
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?savedOnly=true", nil, 200))

	// Assert
	require.Len(t, page.Data, 2)
	require.ElementsMatch(t, []string{rss.ID, manual.ID}, []string{page.Data[0].ID, page.Data[1].ID})
}

func TestFilteringByAnUnknownSourceReturnsAnEmptyArray(t *testing.T) {
	// Arrange
	router := api.New(t)
	testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201)

	// Act
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/?sourceId="+uuid.NewString(), nil, 200))

	// Assert
	require.NotNil(t, page.Data, "empty results must serialize as an array")
	require.Empty(t, page.Data)
	require.False(t, page.Page.HasNext)
}

func TestOmittedOrNonpositivePageSizeUsesTwentyArticles(t *testing.T) {
	for _, scenario := range []struct{ name, query string }{
		{"omitted", ""}, {"zero", "?pageSize=0"}, {"negative", "?pageSize=-1"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			for i := 0; i < 21; i++ {
				testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": fmt.Sprintf("https://example.test/%d", i)}, 201)
			}

			// Act
			page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/"+scenario.query, nil, 200))

			// Assert
			require.Len(t, page.Data, 20)
			require.True(t, page.Page.HasNext)
		})
	}
}

func TestEntriesWithoutPublicationDatesUseTheirCreationDateForOrdering(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	published := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{
		"sourceId": sourceID, "title": "Older news", "link": "https://example.test/old", "publishedAt": "2000-01-01T00:00:00Z",
	}, 201))
	undated := testkit.Decode[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Just added", "link": "https://example.test/new"}, 201))

	// Act: list articles.
	page := testkit.Decode[httpx.CursorResponse[entry]](t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))

	// Assert
	require.Len(t, page.Data, 2)
	require.Equal(t, undated.ID, page.Data[0].ID)
	require.Equal(t, published.ID, page.Data[1].ID)
}
