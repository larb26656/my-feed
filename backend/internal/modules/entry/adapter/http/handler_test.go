package http_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"

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
		"name": "Daily news", "type": "rss", "config": map[string]any{"url": "https://example.test/rss"},
	}, 201)
	return decodeSuccess[struct{ ID string }](t, response).ID
}

func TestIngestingANewArticleReturnsItsIdentityAndMetadata(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/article"}

	// Act
	created := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))

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
	first := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))

	// Act
	repeated := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 200))
	page := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))

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
	first := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))
	article["title"] = "Updated headline"
	article["description"] = "Updated summary"
	article["thumbnail"] = "https://example.test/photo.png"
	article["publishedAt"] = "2025-01-02T03:04:05Z"

	// Act
	refreshed := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 200))
	got := decodeSuccess[entry](t, testkit.Request(t, router, "GET", "/api/v1/entries/"+first.ID, nil, 200))

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
	first := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))
	path := "/api/v1/entries/" + first.ID
	read := decodeSuccess[entry](t, testkit.Request(t, router, "POST", path+"/read", nil, 200))
	saved := decodeSuccess[entry](t, testkit.Request(t, router, "POST", path+"/save", nil, 200))
	require.NotNil(t, read.ReadAt)
	require.NotNil(t, saved.SavedAt)
	article["title"] = "Updated headline"

	// Act
	testkit.Request(t, router, "POST", "/api/v1/entries/", article, 200)
	got := decodeSuccess[entry](t, testkit.Request(t, router, "GET", path, nil, 200))

	// Assert
	require.Equal(t, read.ReadAt, got.ReadAt)
	require.Equal(t, saved.SavedAt, got.SavedAt)
}

func TestSavingALinkAssignsTheSystemManualSource(t *testing.T) {
	// Arrange
	router := api.New(t)
	sources := decodeSuccess[[]struct{ ID, Type string }](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
	require.Len(t, sources, 1)
	require.Equal(t, "manual", sources[0].Type)

	// Act
	saved := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Bookmark", "link": "https://example.test/article", "publishedAt": "2025-01-02T10:04:05.123+07:00"}, 201))

	// Assert
	require.Equal(t, sources[0].ID, saved.SourceID)
	require.NotNil(t, saved.PublishedAt)
	require.Equal(t, "2025-01-02T03:04:05.123Z", *saved.PublishedAt)
}

func TestSavingTheSameLinkRefreshesTheExistingBookmark(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := map[string]string{"title": "Original", "link": "https://example.test/article"}
	first := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", article, 201))
	article["title"] = "Updated bookmark"

	// Act
	refreshed := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", article, 201))

	// Assert
	require.Equal(t, first.ID, refreshed.ID)
	require.Equal(t, "Updated bookmark", refreshed.Title)
}

func TestTheSameLinkUnderManualAndRSSSourcesHasSeparateIdentities(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := map[string]string{"title": "Bookmark", "link": "https://example.test/shared"}
	manual := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", article, 201))
	article["sourceId"] = sourceID

	// Act
	rss := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", article, 201))

	// Assert
	require.NotEqual(t, manual.ID, rss.ID)
	require.Equal(t, sourceID, rss.SourceID)
	require.NotEqual(t, manual.SourceID, rss.SourceID)
}

func TestSavingAnUnreadArticleLeavesItUnread(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))

	// Act
	saved := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/"+article.ID+"/save", nil, 200))

	// Assert
	require.NotNil(t, saved.SavedAt)
	require.Nil(t, saved.ReadAt)
}

func TestMarkingASavedArticleReadPreservesItsSavedChoice(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))
	path := "/api/v1/entries/" + article.ID
	saved := decodeSuccess[entry](t, testkit.Request(t, router, "POST", path+"/save", nil, 200))
	require.NotNil(t, saved.SavedAt)

	// Act
	read := decodeSuccess[entry](t, testkit.Request(t, router, "POST", path+"/read", nil, 200))

	// Assert
	require.NotNil(t, read.ReadAt)
	require.Equal(t, saved.SavedAt, read.SavedAt)
}

func TestClearingReadPreservesTheSavedChoice(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))
	path := "/api/v1/entries/" + article.ID
	testkit.Request(t, router, "POST", path+"/read", nil, 200)
	saved := decodeSuccess[entry](t, testkit.Request(t, router, "POST", path+"/save", nil, 200))
	require.NotNil(t, saved.ReadAt)
	require.NotNil(t, saved.SavedAt)

	// Act
	unread := decodeSuccess[entry](t, testkit.Request(t, router, "POST", path+"/clear-read", nil, 200))

	// Assert
	require.Nil(t, unread.ReadAt)
	require.Equal(t, saved.SavedAt, unread.SavedAt)
}

func TestClearingSavedPreservesTheReadChoice(t *testing.T) {
	// Arrange
	router := api.New(t)
	article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))
	path := "/api/v1/entries/" + article.ID
	testkit.Request(t, router, "POST", path+"/save", nil, 200)
	read := decodeSuccess[entry](t, testkit.Request(t, router, "POST", path+"/read", nil, 200))
	require.NotNil(t, read.ReadAt)
	require.NotNil(t, read.SavedAt)

	// Act
	cleared := decodeSuccess[entry](t, testkit.Request(t, router, "POST", path+"/clear-save", nil, 200))

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
		{name: "empty library", count: 0},
		{name: "unsaved article", hasArticle: true, count: 0},
		{name: "saved article", hasArticle: true, save: true, count: 1},
		{name: "cleared bookmark", hasArticle: true, save: true, clear: true, count: 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			if scenario.hasArticle {
				article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201))
				path := "/api/v1/entries/" + article.ID
				if scenario.save {
					testkit.Request(t, router, "POST", path+"/save", nil, 200)
				}
				if scenario.clear {
					testkit.Request(t, router, "POST", path+"/clear-save", nil, 200)
				}
			}

			// Act
			count := decodeSuccess[struct{ Count int }](t, testkit.Request(t, router, "GET", "/api/v1/entries/saved-count", nil, 200))

			// Assert
			require.Equal(t, scenario.count, count.Count)
		})
	}
}

func TestEntriesWithTheSamePublicationTimeAreSortedByAscendingID(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	var ids []string
	for i := 0; i < 3; i++ {
		article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": fmt.Sprintf("https://example.test/%d", i), "publishedAt": "2025-01-01T00:00:00Z"}, 201))
		ids = append(ids, article.ID)
	}
	sort.Strings(ids)

	// Act
	page := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))

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
		article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": fmt.Sprintf("https://example.test/%d", i), "publishedAt": "2025-01-01T00:00:00Z"}, 201))
		ids = append(ids, article.ID)
	}
	first := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=2", nil, 200))
	require.Len(t, first.Data, 2)
	require.NotEmpty(t, first.Meta.NextCursor)

	// Act
	second := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=2&cursor="+url.QueryEscape(first.Meta.NextCursor), nil, 200))

	// Assert
	require.Len(t, second.Data, 1)
	require.Empty(t, second.Meta.NextCursor)
	require.ElementsMatch(t, ids, []string{first.Data[0].ID, first.Data[1].ID, second.Data[0].ID})
}

func TestANewerArrivalDoesNotShiftTheNextPage(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	older := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "Older", "link": "https://example.test/older", "publishedAt": "2024-01-01T00:00:00Z"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "Newer", "link": "https://example.test/newer", "publishedAt": "2025-01-01T00:00:00Z"}, 201)
	first := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=1", nil, 200))
	require.NotEmpty(t, first.Meta.NextCursor)
	testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "Breaking", "link": "https://example.test/breaking", "publishedAt": "2026-01-01T00:00:00Z"}, 201)

	// Act
	second := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=1&cursor="+url.QueryEscape(first.Meta.NextCursor), nil, 200))

	// Assert
	require.Len(t, second.Data, 1)
	require.Equal(t, older.ID, second.Data[0].ID)
	require.Empty(t, second.Meta.NextCursor)
}

func TestACursorCannotBeReusedWithDifferentFilters(t *testing.T) {
	// Arrange
	router := api.New(t)
	for i := 0; i < 2; i++ {
		testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": fmt.Sprintf("https://example.test/%d", i)}, 201)
	}
	first := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=1", nil, 200))
	require.NotEmpty(t, first.Meta.NextCursor)

	// Act
	failure := decodeFailure(t, testkit.Request(t, router, "GET", "/api/v1/entries/?savedOnly=true&cursor="+url.QueryEscape(first.Meta.NextCursor), nil, 400))

	// Assert
	require.Equal(t, "VALIDATION", failure.Code)
	require.Equal(t, "cursor does not match criteria", failure.Message)
}

func TestInvalidEntryRequestsReturnUsefulBusinessErrors(t *testing.T) {
	// Arrange: immutable request data.
	missing := uuid.NewString()
	for _, scenario := range []struct {
		name, method, path, body string
		status                   int
		code, message            string
	}{
		{"malformed JSON", "POST", "/api/v1/entries/", "{", 400, "VALIDATION", "Invalid JSON body"},
		{"invalid source ID", "POST", "/api/v1/entries/", `{"sourceId":"bad"}`, 400, "VALIDATION", "Invalid JSON body"},
		{"omitted source ID", "POST", "/api/v1/entries/", `{"title":"News","link":"https://example.test/a"}`, 400, "VALIDATION", "Validation failed"},
		{"nil UUID", "POST", "/api/v1/entries/", `{"sourceId":"00000000-0000-0000-0000-000000000000","title":"News","link":"https://example.test/a"}`, 400, "VALIDATION", "Validation failed"},
		{"missing title and link", "POST", "/api/v1/entries/", fmt.Sprintf(`{"sourceId":%q}`, missing), 400, "VALIDATION", "Validation failed"},
		{"invalid publication time", "POST", "/api/v1/entries/save-link", `{"title":"News","link":"https://example.test/a","publishedAt":"yesterday"}`, 400, "VALIDATION", "Invalid JSON body"},
		{"missing source", "POST", "/api/v1/entries/", fmt.Sprintf(`{"sourceId":%q,"title":"A","link":"https://example.test/a"}`, missing), 404, "NOT_FOUND", "source not found"},
		{"invalid entry ID", "GET", "/api/v1/entries/bad", "", 400, "VALIDATION", "invalid entry id"},
		{"missing entry", "GET", "/api/v1/entries/" + missing, "", 404, "NOT_FOUND", "entry not found"},
		{"invalid cursor", "GET", "/api/v1/entries/?cursor=bad", "", 400, "VALIDATION", "invalid cursor"},
		{"invalid page size", "GET", "/api/v1/entries/?pageSize=many", "", 400, "VALIDATION", "Invalid Entry query"},
		{"invalid unread filter", "GET", "/api/v1/entries/?unread=yes", "", 400, "VALIDATION", "Invalid Entry query"},
		{"invalid saved filter", "GET", "/api/v1/entries/?savedOnly=yes", "", 400, "VALIDATION", "Invalid Entry query"},
		{"invalid source filter", "GET", "/api/v1/entries/?sourceId=bad", "", 400, "VALIDATION", "Invalid Entry query"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			router := api.New(t)

			// Act: submit the request that should be rejected.
			failure := decodeFailure(t, testkit.Request(t, router, scenario.method, scenario.path, scenario.body, scenario.status))

			// Assert
			require.Equal(t, scenario.code, failure.Code)
			require.Equal(t, scenario.message, failure.Message)
		})
	}
}

func TestSaveLinkReturnsValidationDetails(t *testing.T) {
	router := api.New(t)
	response := testkit.Request(t, router, "POST", "/api/v1/entries/save-link",
		map[string]string{"title": "Example"}, http.StatusBadRequest)
	failure := decodeFailure(t, response)
	require.Equal(t, "VALIDATION", failure.Code)
	require.Contains(t, response.Body.String(), `"rule":"required"`)
	require.Contains(t, response.Body.String(), `"detail":{"fields":`)
}

func TestReaderActionsOnMissingArticlesReturnNotFound(t *testing.T) {
	// Arrange: immutable request data.
	missing := uuid.NewString()
	for _, action := range []string{"read", "save", "clear-read", "clear-save"} {
		t.Run("missing article cannot "+action, func(t *testing.T) {
			// Arrange
			router := api.New(t)

			// Act: submit the request that should be rejected.
			failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/entries/"+missing+"/"+action, nil, 404))

			// Assert
			require.Equal(t, "NOT_FOUND", failure.Code)
		})
	}
}

func TestIngestionRejectsManualSources(t *testing.T) {
	// Arrange
	router := api.New(t)
	sources := decodeSuccess[[]struct{ ID string }](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
	require.Len(t, sources, 1)

	// Act
	failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sources[0].ID, "title": "News", "link": "https://example.test/a"}, 400))

	// Assert
	require.Equal(t, "VALIDATION", failure.Code)
	require.Equal(t, "ingest can only record Entries into fetchable Sources", failure.Message)
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
			failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": scenario.title, "link": scenario.link}, 400))

			// Assert
			require.Equal(t, "VALIDATION", failure.Code)
		})
	}
}

func TestIngestionRejectsPausedRSSSources(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	testkit.Request(t, router, "POST", "/api/v1/sources/"+sourceID+"/deactivate", nil, 200)

	// Act
	failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/a"}, 409))

	// Assert
	require.Equal(t, "SOURCE_DEACTIVATED", failure.Code)
	require.Equal(t, "cannot ingest into a deactivated source", failure.Message)
}

func TestPausingASourceDoesNotHideItsExistingArticles(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/a"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/sources/"+sourceID+"/deactivate", nil, 200)

	// Act
	page := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))

	// Assert
	require.Len(t, page.Data, 1)
	require.Equal(t, article.ID, page.Data[0].ID)
}

func TestSourceFilterReturnsOnlyArticlesFromThatSource(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	rss := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/rss-article"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Bookmark", "link": "https://example.test/manual"}, 201)

	// Act
	page := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?sourceId="+sourceID, nil, 200))

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
		article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": fmt.Sprintf("https://example.test/%d", i)}, 201))
		articles = append(articles, article)
	}
	for _, article := range articles[:2] {
		testkit.Request(t, router, "POST", "/api/v1/entries/"+article.ID+"/save", nil, 200)
	}
	testkit.Request(t, router, "POST", "/api/v1/entries/"+articles[0].ID+"/read", nil, 200)
	manual := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Bookmark", "link": "https://example.test/manual"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/entries/"+manual.ID+"/save", nil, 200)

	// Act
	page := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?sourceId="+sourceID+"&unread=true&savedOnly=true", nil, 200))

	// Assert
	require.Len(t, page.Data, 1)
	require.Equal(t, articles[1].ID, page.Data[0].ID)
}

func TestSavedFilterIncludesBookmarksAcrossSources(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	rss := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": "https://example.test/rss-article"}, 201))
	manual := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Bookmark", "link": "https://example.test/manual"}, 201))
	for _, article := range []entry{rss, manual} {
		testkit.Request(t, router, "POST", "/api/v1/entries/"+article.ID+"/save", nil, 200)
	}
	testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Unsaved", "link": "https://example.test/unsaved"}, 201)

	// Act
	page := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?savedOnly=true", nil, 200))

	// Assert
	require.Len(t, page.Data, 2)
	require.ElementsMatch(t, []string{rss.ID, manual.ID}, []string{page.Data[0].ID, page.Data[1].ID})
}

func TestFilteringByAnUnknownSourceReturnsAnEmptyArray(t *testing.T) {
	// Arrange
	router := api.New(t)
	testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": "https://example.test/a"}, 201)

	// Act
	page := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?sourceId="+uuid.NewString(), nil, 200))

	// Assert
	require.NotNil(t, page.Data, "empty results must serialize as an array")
	require.Empty(t, page.Data)
	require.Empty(t, page.Meta.NextCursor)
}

func TestEntryPageUsesSharedPageSizeDefaultsAndBounds(t *testing.T) {
	router := api.New(t)
	for i := 0; i < 11; i++ {
		testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Article", "link": fmt.Sprintf("https://example.test/%d", i)}, 201)
	}

	page := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))
	require.Len(t, page.Data, 10)
	require.Equal(t, 10, page.Meta.PageSize)
	require.NotEmpty(t, page.Meta.NextCursor)

	for _, size := range []string{"0", "-1", "101"} {
		failure := decodeFailure(t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize="+size, nil, 400))
		require.Equal(t, "VALIDATION", failure.Code)
	}
}

func TestEntriesWithoutPublicationDatesAppearLastByDefault(t *testing.T) {
	// Arrange
	router := api.New(t)
	sourceID := rssSource(t, router)
	published := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{
		"sourceId": sourceID, "title": "Older news", "link": "https://example.test/old", "publishedAt": "2000-01-01T00:00:00Z",
	}, 201))
	undated := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Just added", "link": "https://example.test/new"}, 201))

	// Act: list articles.
	page := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/", nil, 200))

	// Assert
	require.Len(t, page.Data, 2)
	require.Equal(t, published.ID, page.Data[0].ID)
	require.Equal(t, undated.ID, page.Data[1].ID)
}

func decodeSuccess[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	body := testkit.Decode[httpx.SuccessResponse[T]](t, response)
	require.True(t, body.Success)
	return body.Data
}

type entryPage struct {
	Success bool
	Data    []entry
	Meta    httpx.CursorMeta
}

func decodePage(t *testing.T, response *httptest.ResponseRecorder) entryPage {
	t.Helper()
	body := testkit.Decode[entryPage](t, response)
	require.True(t, body.Success)
	require.Positive(t, body.Meta.PageSize)
	return body
}

func decodeFailure(t *testing.T, response *httptest.ResponseRecorder) httpx.ErrorBody {
	t.Helper()
	body := testkit.Decode[httpx.ErrorResponse](t, response)
	require.False(t, body.Success)
	return body.Error
}

func TestSavedEntriesAreOrderedByTheTimeTheyWereSaved(t *testing.T) {
	// Arrange
	fixture := arrangeIndependentEntryClocks(t)

	// Act
	newest := decodePage(t, testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?savedOnly=true&savedAtSort=desc", nil, 200))
	oldest := decodePage(t, testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?savedOnly=true&savedAtSort=asc", nil, 200))

	// Assert
	require.Equal(t, []string{fixture.recordedEarlier.ID, fixture.earlyPublish.ID, fixture.latePublish.ID}, idsOf(newest.Data))
	require.Equal(t, []string{fixture.latePublish.ID, fixture.earlyPublish.ID, fixture.recordedEarlier.ID}, idsOf(oldest.Data))
}

func TestSavedPaginationFollowsSavedTime(t *testing.T) {
	// Arrange
	fixture := arrangeIndependentEntryClocks(t)
	first := decodePage(t, testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?savedOnly=true&savedAtSort=desc&pageSize=2", nil, 200))
	require.Len(t, first.Data, 2)
	require.NotEmpty(t, first.Meta.NextCursor)

	// Act
	second := decodePage(t, testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?savedOnly=true&savedAtSort=desc&pageSize=2&cursor="+url.QueryEscape(first.Meta.NextCursor), nil, 200))

	// Assert
	require.Equal(t, []string{fixture.recordedEarlier.ID, fixture.earlyPublish.ID}, idsOf(first.Data))
	require.Equal(t, []string{fixture.latePublish.ID}, idsOf(second.Data))
	require.Empty(t, second.Meta.NextCursor)
	testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?cursor="+url.QueryEscape(first.Meta.NextCursor), nil, 400)
}

func TestTheMainFeedStaysOrderedByPublicationTime(t *testing.T) {
	// Arrange
	fixture := arrangeIndependentEntryClocks(t)

	// Act
	page := decodePage(t, testkit.Request(t, fixture.router, "GET", "/api/v1/entries/", nil, 200))

	// Assert
	require.Equal(t, []string{fixture.latePublish.ID, fixture.recordedEarlier.ID, fixture.earlyPublish.ID, fixture.recordedLater.ID}, idsOf(page.Data))
}

func TestSavedLinksAreOrderedByTheTimeTheyWereRecorded(t *testing.T) {
	// Arrange
	fixture := arrangeIndependentEntryClocks(t)

	// Act
	newest := decodePage(t, testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?createdAtSort=desc&sourceId="+fixture.manualID, nil, 200))
	oldest := decodePage(t, testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?createdAtSort=asc&sourceId="+fixture.manualID, nil, 200))

	// Assert
	require.Equal(t, []string{fixture.recordedLater.ID, fixture.recordedEarlier.ID}, idsOf(newest.Data))
	require.Equal(t, []string{fixture.recordedEarlier.ID, fixture.recordedLater.ID}, idsOf(oldest.Data))
}

func TestSavedLinksPaginationFollowsRecordedTime(t *testing.T) {
	// Arrange
	fixture := arrangeIndependentEntryClocks(t)
	first := decodePage(t, testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?createdAtSort=desc&sourceId="+fixture.manualID+"&pageSize=1", nil, 200))
	require.NotEmpty(t, first.Meta.NextCursor)

	// Act
	second := decodePage(t, testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?createdAtSort=desc&sourceId="+fixture.manualID+"&pageSize=1&cursor="+url.QueryEscape(first.Meta.NextCursor), nil, 200))

	// Assert
	require.Equal(t, fixture.recordedLater.ID, first.Data[0].ID)
	require.Equal(t, fixture.recordedEarlier.ID, second.Data[0].ID)
	require.Empty(t, second.Meta.NextCursor)
	testkit.Request(t, fixture.router, "GET", "/api/v1/entries/?createdAtSort=asc&sourceId="+fixture.manualID+"&cursor="+url.QueryEscape(first.Meta.NextCursor), nil, 400)
}

type clockedEntries struct {
	router                         http.Handler
	manualID                       string
	earlyPublish, latePublish      entry
	recordedLater, recordedEarlier entry
}

func arrangeIndependentEntryClocks(t *testing.T) clockedEntries {
	t.Helper()
	router, db := api.NewWithDB(t)
	sources := decodeSuccess[[]struct {
		ID, Type string
	}](t, testkit.Request(t, router, "GET", "/api/v1/sources/?pageSize=100", nil, 200))
	var manualID string
	for _, source := range sources {
		if source.Type == "manual" {
			manualID = source.ID
		}
	}
	require.NotEmpty(t, manualID)
	sourceID := rssSource(t, router)
	earlyPublish := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "Early publish", "link": "https://example.test/early"}, 201))
	latePublish := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "Late publish", "link": "https://example.test/late"}, 201))
	recordedLater := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Recorded later", "link": "https://example.test/later"}, 201))
	recordedEarlier := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/save-link", map[string]string{"title": "Recorded earlier", "link": "https://example.test/earlier"}, 201))

	setEntryClock(t, db, earlyPublish.ID, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), timePtr(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)))
	setEntryClock(t, db, latePublish.ID, time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)))
	setEntryClock(t, db, recordedLater.ID, time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC), nil)
	setEntryClock(t, db, recordedEarlier.ID, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), timePtr(time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)))
	return clockedEntries{router: router, manualID: manualID, earlyPublish: earlyPublish, latePublish: latePublish, recordedLater: recordedLater, recordedEarlier: recordedEarlier}
}

func setEntryClock(t *testing.T, db *sql.DB, id string, published, created time.Time, saved *time.Time) {
	t.Helper()
	var savedAt any
	if saved != nil {
		savedAt = saved.UTC().UnixNano()
	}
	result, err := db.Exec(`UPDATE entries SET published_at = ?, created_at = ?, saved_at = ? WHERE id = ?`, published.UTC().UnixNano(), created.UTC().UnixNano(), savedAt, id)
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
}

func timePtr(value time.Time) *time.Time { return &value }

func idsOf(entries []entry) []string {
	ids := make([]string, 0, len(entries))
	for _, item := range entries {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestOldestFirstPaginationAndCursorDirection(t *testing.T) {
	router := api.New(t)
	sourceID := rssSource(t, router)
	var ids []string
	for i := 0; i < 3; i++ {
		article := decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", map[string]string{"sourceId": sourceID, "title": "News", "link": fmt.Sprintf("https://example.test/oldest-%d", i), "publishedAt": "2025-01-01T00:00:00Z"}, 201))
		ids = append(ids, article.ID)
	}
	sort.Strings(ids)
	first := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=2&publishedAtSort=asc", nil, 200))
	require.Len(t, first.Data, 2)
	require.Equal(t, ids[:2], []string{first.Data[0].ID, first.Data[1].ID})
	require.NotEmpty(t, first.Meta.NextCursor)
	cursor := url.QueryEscape(first.Meta.NextCursor)
	second := decodePage(t, testkit.Request(t, router, "GET", "/api/v1/entries/?pageSize=2&publishedAtSort=asc&cursor="+cursor, nil, 200))
	require.Len(t, second.Data, 1)
	require.Equal(t, ids[2], second.Data[0].ID)
	testkit.Request(t, router, "GET", "/api/v1/entries/?publishedAtSort=desc&cursor="+cursor, nil, 400)
	testkit.Request(t, router, "GET", "/api/v1/entries/?publishedAtSort=invalid", nil, 400)
}

func TestMultipleSortFieldsContinueAcrossNulls(t *testing.T) {
	router := api.New(t)
	sourceID := rssSource(t, router)
	makeEntry := func(name, published string) entry {
		body := map[string]string{"sourceId": sourceID, "title": name, "link": "https://example.test/" + name}
		if published != "" {
			body["publishedAt"] = published
		}
		return decodeSuccess[entry](t, testkit.Request(t, router, "POST", "/api/v1/entries/", body, 201))
	}
	a := makeEntry("a", "2025-01-01T00:00:00Z")
	b := makeEntry("b", "2025-01-01T00:00:00Z")
	c := makeEntry("c", "")
	testkit.Request(t, router, "POST", "/api/v1/entries/"+b.ID+"/save", nil, 200)
	query := "/api/v1/entries/?pageSize=1&publishedAtSort=desc&savedAtSort=asc"
	var got []string
	for {
		page := decodePage(t, testkit.Request(t, router, "GET", query, nil, 200))
		require.Len(t, page.Data, 1)
		got = append(got, page.Data[0].ID)
		if page.Meta.NextCursor == "" {
			break
		}
		query = "/api/v1/entries/?pageSize=1&publishedAtSort=desc&savedAtSort=asc&cursor=" + url.QueryEscape(page.Meta.NextCursor)
	}
	require.Equal(t, []string{b.ID, a.ID, c.ID}, got)
}
