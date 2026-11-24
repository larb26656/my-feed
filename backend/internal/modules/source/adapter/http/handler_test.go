package http_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/larb26656/my-feed/backend/internal/testkit/api"
	"github.com/larb26656/my-feed/backend/internal/testkit/customtype"
	"github.com/stretchr/testify/require"
)

type source struct {
	ID, Name, Type, CreatedAt, UpdatedAt string
	Config                               map[string]any
	Active                               bool
}

func createRSSSource(t *testing.T, router http.Handler, name, feedURL string) source {
	t.Helper()
	return decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{"name": name, "type": "rss", "config": map[string]any{"url": feedURL}}, 201))
}

func TestSubscribingToRSSReturnsNormalizedSourceDetails(t *testing.T) {
	// Arrange
	router := api.New(t)
	input := map[string]any{"name": " Daily news ", "type": "rss", "config": map[string]any{"url": " https://example.test/rss "}}

	// Act
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", input, 201))

	// Assert
	require.NotEmpty(t, created.ID)
	require.Equal(t, "Daily news", created.Name)
	require.Equal(t, "rss", created.Type)
	require.Contains(t, created.Config, "url")
	require.Equal(t, "https://example.test/rss", created.Config["url"])
	require.True(t, created.Active)
}

func TestEditingASubscriptionPersistsItsNewNameAndFeedURL(t *testing.T) {
	// Arrange
	router := api.New(t)
	original := createRSSSource(t, router, "Daily news", "https://example.test/rss")
	path := "/api/v1/sources/" + original.ID

	// Act
	updated := decodeSuccess[source](t, testkit.Request(t, router, "PATCH", path, map[string]any{"name": "Weekly news", "config": map[string]any{"url": "https://example.test/weekly"}}, 200))
	got := decodeSuccess[source](t, testkit.Request(t, router, "GET", path, nil, 200))

	// Assert
	require.Equal(t, original.ID, got.ID)
	require.Equal(t, original.CreatedAt, got.CreatedAt)
	require.Equal(t, "Weekly news", got.Name)
	require.Contains(t, got.Config, "url")
	require.Equal(t, "https://example.test/weekly", got.Config["url"])
	require.Equal(t, "rss", got.Type)
	require.Equal(t, updated, got)
}

func TestPausingAnRSSSourceIsIdempotent(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "News", "https://example.test/rss")
	path := "/api/v1/sources/" + subscription.ID + "/deactivate"
	first := decodeSuccess[source](t, testkit.Request(t, router, "POST", path, nil, 200))
	require.False(t, first.Active)

	// Act
	repeated := decodeSuccess[source](t, testkit.Request(t, router, "POST", path, nil, 200))

	// Assert
	require.Equal(t, first, repeated)
}

func TestDefaultSourceListIncludesPausedSubscriptions(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "News", "https://example.test/rss")
	testkit.Request(t, router, "POST", "/api/v1/sources/"+subscription.ID+"/deactivate", nil, 200)

	// Act
	all := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))

	// Assert
	require.Len(t, all, 2)
	require.Equal(t, subscription.ID, all[1].ID)
	require.False(t, all[1].Active)
}

func TestSourceListReturnsRequestedPageAndTotal(t *testing.T) {
	// Arrange
	router := api.New(t)
	createRSSSource(t, router, "First", "https://example.test/first")
	second := createRSSSource(t, router, "Second", "https://example.test/second")

	// Act
	response := testkit.Request(t, router, "GET", "/api/v1/sources/?page=2&pageSize=2", nil, 200)
	result := testkit.Decode[httpx.SuccessResponse[[]source]](t, response)

	// Assert
	require.Len(t, result.Data, 1)
	require.Equal(t, second.ID, result.Data[0].ID)
	require.Equal(t, map[string]any{"page": float64(2), "pageSize": float64(2), "total": float64(3), "totalPages": float64(2)}, result.Meta)
}

func TestSourceListCombinesNameTypeAndStatusFilters(t *testing.T) {
	// Arrange
	router := api.New(t)
	second := createRSSSource(t, router, "Tech 100% Zebra", "https://example.test/z")
	first := createRSSSource(t, router, "Tech 100% Alpha", "https://example.test/a")
	createRSSSource(t, router, "tech 100X Other", "https://example.test/x")
	// Act
	response := testkit.Request(t, router, "GET", "/api/v1/sources/?nameLike=100%25&typeEqual=rss&activeEqual=true", nil, 200)
	result := testkit.Decode[httpx.SuccessResponse[[]source]](t, response)
	// Assert
	require.Len(t, result.Data, 2)
	require.Equal(t, []string{second.ID, first.ID}, []string{result.Data[0].ID, result.Data[1].ID})
	require.Equal(t, float64(2), result.Meta.(map[string]any)["total"])
}

func TestSourceListFiltersDeactivatedSources(t *testing.T) {
	// Arrange
	router := api.New(t)
	paused := createRSSSource(t, router, "Paused", "https://example.test/paused")
	testkit.Request(t, router, "POST", "/api/v1/sources/"+paused.ID+"/deactivate", nil, 200)
	// Act
	got := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/?activeEqual=false", nil, 200))
	// Assert
	require.Len(t, got, 1)
	require.Equal(t, paused.ID, got[0].ID)
}

func TestSourceNameFilterMatchesUnicodeIgnoringCase(t *testing.T) {
	// Arrange
	router := api.New(t)
	want := createRSSSource(t, router, "ÉCOLE", "https://example.test/ecole")
	// Act
	got := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/?nameLike=%C3%A9cole", nil, 200))
	// Assert
	require.Len(t, got, 1)
	require.Equal(t, want.ID, got[0].ID)
}

func TestSourceListVeryLargePageIsEmpty(t *testing.T) {
	// Arrange
	router := api.New(t)
	// Act
	got := testkit.Decode[httpx.SuccessResponse[[]source]](t, testkit.Request(t, router, "GET", "/api/v1/sources/?page=9223372036854775807&pageSize=100", nil, 200))
	// Assert
	require.Empty(t, got.Data)
	require.Equal(t, float64(1), got.Meta.(map[string]any)["total"])
}

func TestSourceListSortsNamesDescending(t *testing.T) {
	// Arrange
	router := api.New(t)
	createRSSSource(t, router, "Alpha", "https://example.test/alpha")
	want := createRSSSource(t, router, "Zebra", "https://example.test/zebra")
	// Act
	got := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/?nameSort=desc", nil, 200))
	// Assert
	require.Equal(t, want.ID, got[0].ID)
}

func TestSourceListRejectsInvalidQueries(t *testing.T) {
	for _, query := range []string{"page=0", "pageSize=101", "typeEqual=unknown", "activeEqual=maybe", "nameSort=sideways", "createdAtSort=sideways"} {
		t.Run(query, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			// Act
			failure := decodeFailure(t, testkit.Request(t, router, "GET", "/api/v1/sources/?"+query, nil, 400))
			// Assert
			require.Equal(t, "VALIDATION", failure.Code)
		})
	}
}

func TestSourceListCanFilterActiveSubscriptions(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "News", "https://example.test/rss")
	testkit.Request(t, router, "POST", "/api/v1/sources/"+subscription.ID+"/deactivate", nil, 200)

	// Act
	active := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/?activeEqual=true", nil, 200))

	// Assert
	require.Len(t, active, 1)
	require.Equal(t, "manual", active[0].Type)
}

func TestPausedSourcesRemainAccessibleByID(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "News", "https://example.test/rss")
	path := "/api/v1/sources/" + subscription.ID
	paused := decodeSuccess[source](t, testkit.Request(t, router, "POST", path+"/deactivate", nil, 200))

	// Act
	got := decodeSuccess[source](t, testkit.Request(t, router, "GET", path, nil, 200))

	// Assert
	require.Equal(t, paused, got)
	require.False(t, got.Active)
}

func TestResumingAPausedSourceReturnsItToTheActiveList(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "News", "https://example.test/rss")
	path := "/api/v1/sources/" + subscription.ID
	testkit.Request(t, router, "POST", path+"/deactivate", nil, 200)

	// Act
	resumed := decodeSuccess[source](t, testkit.Request(t, router, "POST", path+"/reactivate", nil, 200))
	active := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))

	// Assert
	require.True(t, resumed.Active)
	require.Len(t, active, 2)
	require.Contains(t, active, resumed)
}

func TestResumingAnAlreadyActiveSourceIsIdempotent(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "News", "https://example.test/rss")

	// Act
	repeated := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/"+subscription.ID+"/reactivate", nil, 200))

	// Assert
	require.Equal(t, subscription, repeated)
}

func TestInvalidSubscriptionsReturnValidationErrorsWithoutCreatingSources(t *testing.T) {
	for _, scenario := range []struct{ name, body, message string }{
		{"blank name", `{"name":" ","type":"rss","config":{"url":"https://example.test/rss"}}`, "name is required"},
		{"blank feed URL", `{"name":"News","type":"rss","config":{"url":" "}}`, "invalid source config"},
		{"reader cannot create manual source", `{"name":"Bookmarks","type":"manual","config":{}}`, "manual sources cannot be created by the reader"},
		{"unsupported type", `{"name":"News","type":"unknown","config":{}}`, "unknown source type"},
		{"malformed JSON", `{`, "invalid json body"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			router := api.New(t)

			// Act
			failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", scenario.body, 400))
			sources := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))

			// Assert
			require.Equal(t, "VALIDATION", failure.Code)
			if scenario.name != "malformed JSON" {
				require.Equal(t, scenario.message, failure.Message)
			}
			require.Len(t, sources, 1, "invalid subscriptions must not create a source")
			require.Equal(t, "manual", sources[0].Type)
		})
	}
}

func TestSubscribingToAnExistingFeedURLReturnsConflict(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "Daily", "https://example.test/daily")

	// Act
	failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{"name": "Another name", "type": "rss", "config": map[string]any{"url": subscription.Config["url"]}}, 409))

	// Assert
	require.Equal(t, "CONFLICT", failure.Code)
}

func TestAnEditConflictingWithAnotherFeedLeavesTheSubscriptionUnchanged(t *testing.T) {
	// Arrange
	router := api.New(t)
	first := createRSSSource(t, router, "Daily", "https://example.test/daily")
	second := createRSSSource(t, router, "Weekly", "https://example.test/weekly")
	path := "/api/v1/sources/" + second.ID

	// Act
	failure := decodeFailure(t, testkit.Request(t, router, "PATCH", path, map[string]any{"name": "Changed", "config": map[string]any{"url": first.Config["url"]}}, 409))
	got := decodeSuccess[source](t, testkit.Request(t, router, "GET", path, nil, 200))

	// Assert
	require.Equal(t, "CONFLICT", failure.Code)
	require.Equal(t, second, got)
}

func TestRenamingASubscriptionAllowsKeepingItsFeedURL(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "News", "https://example.test/rss")

	// Act
	renamed := decodeSuccess[source](t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+subscription.ID, map[string]any{"name": "Renamed", "config": map[string]any{"url": subscription.Config["url"]}}, 200))

	// Assert
	require.Equal(t, "Renamed", renamed.Name)
	require.Equal(t, subscription.Config, renamed.Config)
	require.Equal(t, subscription.ID, renamed.ID)
}

func TestInvalidEditsLeaveTheSubscriptionUnchanged(t *testing.T) {
	for _, scenario := range []struct{ name, body string }{
		{"blank name", `{"name":" ","config":{"url":"https://example.test/new"}}`},
		{"blank feed URL", `{"name":"Changed","config":{"url":" "}}`},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			subscription := createRSSSource(t, router, "News", "https://example.test/rss")
			path := "/api/v1/sources/" + subscription.ID

			// Act
			failure := decodeFailure(t, testkit.Request(t, router, "PATCH", path, scenario.body, 400))
			got := decodeSuccess[source](t, testkit.Request(t, router, "GET", path, nil, 200))

			// Assert
			require.Equal(t, "VALIDATION", failure.Code)
			require.Equal(t, subscription, got)
		})
	}
}

func TestTheSystemSavedLinksSourceIsVisibleOnStartup(t *testing.T) {
	// Arrange
	router := api.New(t)

	// Act
	sources := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))

	// Assert
	require.Len(t, sources, 1)
	require.Equal(t, "Saved links", sources[0].Name)
	require.Equal(t, "manual", sources[0].Type)
	require.Empty(t, sources[0].Config)
}

func TestReaderCannotEditOrToggleTheManualSource(t *testing.T) {
	for _, action := range []struct{ name, method, suffix string }{{"edit", "PATCH", ""}, {"pause", "POST", "/deactivate"}, {"resume", "POST", "/reactivate"}} {
		t.Run(action.name, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			sources := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
			require.Len(t, sources, 1)
			path := "/api/v1/sources/" + sources[0].ID

			// Act
			failure := decodeFailure(t, testkit.Request(t, router, action.method, path+action.suffix, map[string]any{"name": "Changed", "config": map[string]any{"url": "https://example.test/rss"}}, 409))
			got := decodeSuccess[source](t, testkit.Request(t, router, "GET", path, nil, 200))

			// Assert
			require.Equal(t, "MANUAL_SOURCE_IMMUTABLE", failure.Code)
			require.Equal(t, sources[0], got)
		})
	}
}

func TestUnknownSourceRequestsReturnUsefulErrors(t *testing.T) {
	for _, scenario := range []struct {
		name, id, code string
		status         int
	}{
		{"invalid ID", "bad", "VALIDATION", 400}, {"missing source", uuid.NewString(), "NOT_FOUND", 404},
	} {
		for _, action := range []struct{ method, suffix string }{{"GET", ""}, {"PATCH", ""}, {"POST", "/deactivate"}, {"POST", "/reactivate"}} {
			t.Run(fmt.Sprintf("%s/%s%s", scenario.name, action.method, action.suffix), func(t *testing.T) {
				// Arrange
				router := api.New(t)

				// Act
				failure := decodeFailure(t, testkit.Request(t, router, action.method, "/api/v1/sources/"+scenario.id+action.suffix, map[string]any{"name": "News", "config": map[string]any{"url": "https://example.test/rss"}}, scenario.status))

				// Assert
				require.Equal(t, scenario.code, failure.Code)
			})
		}
	}
}

func TestYouTubeChannelURLNormalizesToChannelID(t *testing.T) {
	// Arrange
	router := api.New(t)
	// Act
	created := decodeSuccess[struct{ Config map[string]any }](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{"name": "Channel", "type": "youtube", "config": map[string]any{"channelId": " https://www.youtube.com/channel/UC1234567890123456789012 "}}, 201))
	// Assert
	require.Equal(t, "UC1234567890123456789012", created.Config["channelId"])
}

func TestRenamingYouTubeSourcePreservesChannel(t *testing.T) {
	// Arrange
	router := api.New(t)
	created := decodeSuccess[struct {
		ID     string
		Config map[string]any
	}](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{"name": "Channel", "type": "youtube", "config": map[string]any{"channelId": "UC1234567890123456789012"}}, 201))
	// Act
	got := decodeSuccess[struct {
		ID, Name string
		Config   map[string]any
	}](t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]string{"name": " Renamed "}, 200))
	// Assert
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.Config["channelId"], got.Config["channelId"])
	require.Equal(t, "Renamed", got.Name)
}

func TestInvalidYouTubeChannelInputsAreRejected(t *testing.T) {
	for _, input := range []string{"@handle", "https://www.youtube.com/@handle", "https://www.youtube.com/c/custom", "https://www.youtube.com/user/legacy", "http://www.youtube.com/channel/UC1234567890123456789012", "https://www.youtube.com/channel/UC1234567890123456789012?x=1"} {
		t.Run(input, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			// Act
			failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{"name": "Channel", "type": "youtube", "config": map[string]any{"channelId": input}}, 400))
			// Assert
			require.Equal(t, "VALIDATION", failure.Code)
			require.Equal(t, "invalid source config", failure.Message)
			require.Len(t, decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200)), 1)
		})
	}
}

func TestDuplicateYouTubeChannelConflictsEvenWhenDeactivated(t *testing.T) {
	// Arrange
	router := api.New(t)
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{"name": "Channel", "type": "youtube", "config": map[string]any{"channelId": "UC1234567890123456789012"}}, 201))
	testkit.Request(t, router, "POST", "/api/v1/sources/"+created.ID+"/deactivate", nil, 200)
	// Act
	failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{"name": "Duplicate", "type": "youtube", "config": map[string]any{"channelId": "https://www.youtube.com/channel/UC1234567890123456789012"}}, 409))
	// Assert
	require.Equal(t, "CONFLICT", failure.Code)
	got := decodeSuccess[source](t, testkit.Request(t, router, "GET", "/api/v1/sources/"+created.ID, nil, 200))
	require.False(t, got.Active)
}

func TestYouTubeRetargetingIsRejected(t *testing.T) {
	for _, changes := range []map[string]string{{"name": "Changed", "youtubeChannelId": "UCabcdefghijklmnopqrstuv"}, {"name": "Changed", "type": "rss"}, {"name": "Changed", "rssUrl": "https://example.test"}} {
		t.Run(fmt.Sprint(changes), func(t *testing.T) {
			// Arrange
			router := api.New(t)
			created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{"name": "Channel", "type": "youtube", "config": map[string]any{"channelId": "UC1234567890123456789012"}}, 201))
			// Act
			failure := decodeFailure(t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, changes, 400))
			// Assert
			require.Equal(t, "VALIDATION", failure.Code)
			require.NotEmpty(t, failure.Message)
			require.Equal(t, created, decodeSuccess[source](t, testkit.Request(t, router, "GET", "/api/v1/sources/"+created.ID, nil, 200)))
		})
	}
}

func decodeSuccess[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	result := testkit.Decode[httpx.SuccessResponse[T]](t, response)
	require.True(t, result.Success)
	return result.Data
}

func decodeFailure(t *testing.T, response *httptest.ResponseRecorder) httpx.ErrorBody {
	t.Helper()
	result := testkit.Decode[httpx.ErrorResponse](t, response)
	require.False(t, result.Success)
	return result.Error
}

func TestPreviewRSSReturnsFirstEntryWithoutPersisting(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<rss version="2.0"><channel><title>News</title><link>https://example.test</link><description>News</description><item><title>First</title><link>https://example.test/first</link></item><item><title>Second</title><link>https://example.test/second</link></item></channel></rss>`))
	}))
	defer origin.Close()
	router := api.New(t)
	response := testkit.Request(t, router, "POST", "/api/v1/sources/preview", map[string]any{"type": "rss", "config": map[string]any{"url": origin.URL}}, 200)
	result := testkit.Decode[httpx.SuccessResponse[map[string]any]](t, response)
	require.Equal(t, "First", result.Data["entry"].(map[string]any)["title"])
	require.NotContains(t, result.Data["entry"].(map[string]any), "publishedAt")
	sources := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))
	require.Len(t, sources, 1)
}

func TestPreviewAtomAndEmptyAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		wantTitle  string
		wantStatus int
	}{
		{"atom", `<feed xmlns="http://www.w3.org/2005/Atom"><title>Feed</title><id>urn:feed</id><updated>2024-01-01T00:00:00Z</updated><entry><title>Atom first</title><id>urn:entry</id><link href="https://example.test/a"/><updated>2024-01-01T00:00:00Z</updated></entry></feed>`, 200, "Atom first", 200},
		{"empty", `<rss version="2.0"><channel><title>Empty</title><link>https://example.test</link><description>Empty</description></channel></rss>`, 200, "", 200},
		{"invalid document", `not a feed`, 200, "", 400},
		{"upstream failure", `unavailable`, 503, "", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer origin.Close()
			router := api.New(t)
			response := testkit.Request(t, router, "POST", "/api/v1/sources/preview", map[string]any{"type": "rss", "config": map[string]any{"url": origin.URL}}, tc.wantStatus)
			if tc.wantStatus != 200 {
				return
			}
			result := testkit.Decode[httpx.SuccessResponse[map[string]any]](t, response)
			if tc.wantTitle == "" {
				require.Nil(t, result.Data["entry"])
			} else {
				require.Equal(t, tc.wantTitle, result.Data["entry"].(map[string]any)["title"])
			}
		})
	}
}

func TestReaderDiscoversRSSConfiguration(t *testing.T) {
	router := api.New(t)
	response := testkit.Request(t, router, "GET", "/api/v1/source-types", nil, 200)
	result := testkit.Decode[httpx.SuccessResponse[struct {
		Types []struct {
			Code   string         `json:"code"`
			Label  string         `json:"label"`
			Schema map[string]any `json:"schema"`
			UI     map[string]any `json:"ui"`
		} `json:"types"`
	}]](t, response)
	require.NotEmpty(t, result.Data.Types)
	require.Equal(t, "rss", result.Data.Types[0].Code)
	require.NotEmpty(t, result.Data.Types[0].Label)
	require.Equal(t, "https://json-schema.org/draft/2020-12/schema", result.Data.Types[0].Schema["$schema"])
	require.NotEmpty(t, result.Data.Types[0].UI)
}

func TestCreatingRSSFromConfigNormalizesAndRejectsInvalidInput(t *testing.T) {
	router := api.New(t)
	response := testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"type": "rss", "name": "News", "config": map[string]any{"url": " https://example.org/feed "}}, 201)
	result := testkit.Decode[httpx.SuccessResponse[struct {
		Config map[string]any `json:"config"`
	}]](t, response)
	require.Equal(t, "https://example.org/feed", result.Data.Config["url"])
	testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"type": "unknown", "name": "Bad", "config": map[string]any{}}, 400)
	for _, config := range []map[string]any{{}, {"url": "ftp://example.org"}, {"url": "https://example.org", "extra": true}} {
		testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"type": "rss", "name": "Bad", "config": config}, 400)
	}
}

func TestUnknownConfigPropertyIdentifiesTheField(t *testing.T) {
	router := api.New(t)
	response := testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"name": "News", "type": "rss", "config": map[string]any{"url": "https://example.test", "extra": true}}, 400)
	result := testkit.Decode[struct {
		Error struct {
			Detail struct {
				Fields map[string]string `json:"fields"`
			} `json:"detail"`
		} `json:"error"`
	}](t, response)
	require.Contains(t, result.Error.Detail.Fields, "extra")
}

func TestInvalidRSSConfigIdentifiesTheField(t *testing.T) {
	router := api.New(t)
	response := testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"name": "News", "type": "rss", "config": map[string]any{}}, 400)
	result := testkit.Decode[struct {
		Error struct {
			Code   string `json:"code"`
			Detail struct {
				Fields map[string]string `json:"fields"`
			} `json:"detail"`
		} `json:"error"`
	}](t, response)
	require.Equal(t, "VALIDATION", result.Error.Code)
	require.Contains(t, result.Error.Detail.Fields, "url")
}

func TestCreatingYouTubeFromConfigNormalizesAndConflicts(t *testing.T) {
	router := api.New(t)
	response := testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"type": "youtube", "name": "Channel", "config": map[string]any{"channelId": " https://www.youtube.com/channel/UC1234567890123456789012 "}}, 201)
	result := testkit.Decode[httpx.SuccessResponse[struct {
		Config map[string]any `json:"config"`
	}]](t, response)
	require.Equal(t, "UC1234567890123456789012", result.Data.Config["channelId"])
	testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"type": "youtube", "name": "Duplicate", "config": map[string]any{"channelId": "UC1234567890123456789012"}}, 409)
	for _, config := range []map[string]any{{}, {"channelId": "invalid"}, {"channelId": "UC1234567890123456789012", "extra": true}} {
		testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"type": "youtube", "name": "Bad", "config": config}, 400)
	}
}

func TestEditSourceConfigNormalizesAndReservesOrigin(t *testing.T) {
	router := api.New(t)
	first := createRSSSource(t, router, "First", "https://example.test/first")
	second := createRSSSource(t, router, "Second", "https://example.test/second")
	path := "/api/v1/sources/" + first.ID
	testkit.Request(t, router, "PATCH", path, map[string]any{"name": "Changed", "config": map[string]any{"url": " https://example.test/new "}}, 200)
	got := decodeSuccess[source](t, testkit.Request(t, router, "GET", path, nil, 200))
	require.Equal(t, "https://example.test/new", got.Config["url"])
	testkit.Request(t, router, "PATCH", path, map[string]any{"name": "Conflict", "config": map[string]any{"url": second.Config["url"]}}, 409)
	got = decodeSuccess[source](t, testkit.Request(t, router, "GET", path, nil, 200))
	require.Equal(t, "Changed", got.Name)
	require.Equal(t, "https://example.test/new", got.Config["url"])
}

func TestEditYouTubeConfigCanonicalizesAndRejectsInvalidInput(t *testing.T) {
	router := api.New(t)
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]any{"name": "Channel", "type": "youtube", "config": map[string]any{"channelId": "UC1234567890123456789012"}}, 201))
	path := "/api/v1/sources/" + created.ID
	testkit.Request(t, router, "PATCH", path, map[string]any{"name": "Channel", "config": map[string]any{"channelId": " https://www.youtube.com/channel/UCabcdefghijklmnopqrstuv "}}, 200)
	got := decodeSuccess[struct{ Config map[string]any }](t, testkit.Request(t, router, "GET", path, nil, 200))
	require.Equal(t, "UCabcdefghijklmnopqrstuv", got.Config["channelId"])
	testkit.Request(t, router, "PATCH", path, map[string]any{"name": "Channel", "config": map[string]any{"channelId": "bad"}}, 400)
}

func TestRegisteredTypeCreateEditAndPreview(t *testing.T) {
	definition := customtype.Definition(t)
	router := api.NewWithDefinitions(t, t.TempDir(), definition)
	discovered := testkit.Decode[httpx.SuccessResponse[struct {
		Types []struct {
			Code string `json:"code"`
		} `json:"types"`
	}]](t, testkit.Request(t, router, "GET", "/api/v1/source-types", nil, 200))
	require.Equal(t, "custom", discovered.Data.Types[2].Code)
	preview := testkit.Request(t, router, "POST", "/api/v1/sources/preview", map[string]any{"type": "custom", "config": customtype.Input("ok-alpha")}, 200)
	result := testkit.Decode[httpx.SuccessResponse[map[string]any]](t, preview)
	require.Equal(t, "ok-alpha", result.Data["entry"].(map[string]any)["title"])
	created := testkit.Decode[httpx.SuccessResponse[struct {
		ID     string         `json:"id"`
		Config map[string]any `json:"config"`
	}]](t, testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"name": "Custom", "type": "custom", "config": customtype.Input("ok-alpha")}, 201))
	require.Equal(t, "ok-alpha", created.Data.Config["text"])
	updated := testkit.Decode[httpx.SuccessResponse[struct {
		Config map[string]any `json:"config"`
	}]](t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.Data.ID, map[string]any{"name": "Custom", "config": customtype.Input("ok-beta")}, 200))
	require.Equal(t, "ok-beta", updated.Data.Config["text"])
	response := testkit.Request(t, router, "GET", "/api/v1/sources?typeEqual=custom", nil, 200)
	listed := testkit.Decode[httpx.SuccessResponse[[]map[string]any]](t, response)
	require.Len(t, listed.Data, 1)
	require.Equal(t, "custom", listed.Data[0]["type"])
}

func TestCreateAndPreviewRequireConfig(t *testing.T) {
	router := api.New(t)
	testkit.Request(t, router, "POST", "/api/v1/sources", map[string]any{"name": "News", "type": "rss", "rssUrl": "https://example.test/feed"}, 400)
	testkit.Request(t, router, "POST", "/api/v1/sources/preview", map[string]any{"type": "rss", "rssUrl": "https://example.test/feed"}, 400)
}

func TestSourceListMultiFieldSortAndPagination(t *testing.T) {
	router := api.New(t)
	first := createRSSSource(t, router, "Alpha", "https://example.test/multi-a")
	second := createRSSSource(t, router, "Alpha", "https://example.test/multi-b")
	createRSSSource(t, router, "Zebra", "https://example.test/multi-c")
	testkit.Request(t, router, "POST", "/api/v1/sources/"+second.ID+"/deactivate", nil, 200)
	// Query order must not change the fixed name-then-active priority.
	query := "activeSort=asc&nameSort=asc&nameLike=a&pageSize=1"
	page1 := testkit.Decode[httpx.SuccessResponse[[]source]](t, testkit.Request(t, router, "GET", "/api/v1/sources/?"+query+"&page=1", nil, 200))
	page2 := testkit.Decode[httpx.SuccessResponse[[]source]](t, testkit.Request(t, router, "GET", "/api/v1/sources/?"+query+"&page=2", nil, 200))
	require.Equal(t, second.ID, page1.Data[0].ID)
	require.Equal(t, first.ID, page2.Data[0].ID)
	require.Equal(t, float64(4), page1.Meta.(map[string]any)["total"])
}
