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
	"github.com/stretchr/testify/require"
)

type source struct {
	ID, Name, Type, CreatedAt, UpdatedAt string
	RSSURL                               *string
	Active                               bool
}

func createRSSSource(t *testing.T, router http.Handler, name, feedURL string) source {
	t.Helper()
	return decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": name, "type": "rss", "rssUrl": feedURL}, 201))
}

func TestSubscribingToRSSReturnsNormalizedSourceDetails(t *testing.T) {
	// Arrange
	router := api.New(t)
	input := map[string]string{"name": " Daily news ", "type": "rss", "rssUrl": " https://example.test/rss "}

	// Act
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", input, 201))

	// Assert
	require.NotEmpty(t, created.ID)
	require.Equal(t, "Daily news", created.Name)
	require.Equal(t, "rss", created.Type)
	require.NotNil(t, created.RSSURL)
	require.Equal(t, "https://example.test/rss", *created.RSSURL)
	require.True(t, created.Active)
}

func TestEditingASubscriptionPersistsItsNewNameAndFeedURL(t *testing.T) {
	// Arrange
	router := api.New(t)
	original := createRSSSource(t, router, "Daily news", "https://example.test/rss")
	path := "/api/v1/sources/" + original.ID

	// Act
	updated := decodeSuccess[source](t, testkit.Request(t, router, "PATCH", path, map[string]string{"name": "Weekly news", "rssUrl": "https://example.test/weekly"}, 200))
	got := decodeSuccess[source](t, testkit.Request(t, router, "GET", path, nil, 200))

	// Assert
	require.Equal(t, original.ID, got.ID)
	require.Equal(t, original.CreatedAt, got.CreatedAt)
	require.Equal(t, "Weekly news", got.Name)
	require.NotNil(t, got.RSSURL)
	require.Equal(t, "https://example.test/weekly", *got.RSSURL)
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

func TestDefaultSourceListOmitsPausedSubscriptions(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "News", "https://example.test/rss")
	testkit.Request(t, router, "POST", "/api/v1/sources/"+subscription.ID+"/deactivate", nil, 200)

	// Act
	active := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))

	// Assert
	require.Len(t, active, 1)
	require.Equal(t, "manual", active[0].Type)
}

func TestSourceListCanIncludePausedSubscriptions(t *testing.T) {
	// Arrange
	router := api.New(t)
	subscription := createRSSSource(t, router, "News", "https://example.test/rss")
	testkit.Request(t, router, "POST", "/api/v1/sources/"+subscription.ID+"/deactivate", nil, 200)

	// Act
	all := decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/?includeDeactivated=true", nil, 200))

	// Assert
	require.Len(t, all, 2)
	for _, item := range all {
		if item.ID == subscription.ID {
			require.False(t, item.Active)
			return
		}
	}
	t.Fatal("paused subscription must be included")
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
		{"blank name", `{"name":" ","type":"rss","rssUrl":"https://example.test/rss"}`, "name is required"},
		{"blank feed URL", `{"name":"News","type":"rss","rssUrl":" "}`, "rss url is required"},
		{"reader cannot create manual source", `{"name":"Bookmarks","type":"manual"}`, "manual sources cannot be created by the reader"},
		{"unsupported type", `{"name":"News","type":"unknown"}`, "source type must be rss or youtube"},
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
	failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": "Another name", "type": "rss", "rssUrl": *subscription.RSSURL}, 409))

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
	failure := decodeFailure(t, testkit.Request(t, router, "PATCH", path, map[string]string{"name": "Changed", "rssUrl": *first.RSSURL}, 409))
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
	renamed := decodeSuccess[source](t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+subscription.ID, map[string]string{"name": "Renamed", "rssUrl": *subscription.RSSURL}, 200))

	// Assert
	require.Equal(t, "Renamed", renamed.Name)
	require.Equal(t, subscription.RSSURL, renamed.RSSURL)
	require.Equal(t, subscription.ID, renamed.ID)
}

func TestInvalidEditsLeaveTheSubscriptionUnchanged(t *testing.T) {
	for _, scenario := range []struct{ name, body string }{
		{"blank name", `{"name":" ","rssUrl":"https://example.test/new"}`},
		{"blank feed URL", `{"name":"Changed","rssUrl":" "}`},
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
	require.Nil(t, sources[0].RSSURL)
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
			failure := decodeFailure(t, testkit.Request(t, router, action.method, path+action.suffix, map[string]string{"name": "Changed", "rssUrl": "https://example.test/rss"}, 409))
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
				failure := decodeFailure(t, testkit.Request(t, router, action.method, "/api/v1/sources/"+scenario.id+action.suffix, map[string]string{"name": "News", "rssUrl": "https://example.test/rss"}, scenario.status))

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
	created := decodeSuccess[struct{ YouTubeChannelID string }](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": "Channel", "type": "youtube", "youtubeChannelId": " https://www.youtube.com/channel/UC1234567890123456789012 "}, 201))
	// Assert
	require.Equal(t, "UC1234567890123456789012", created.YouTubeChannelID)
}

func TestRenamingYouTubeSourcePreservesChannel(t *testing.T) {
	// Arrange
	router := api.New(t)
	created := decodeSuccess[struct{ ID, YouTubeChannelID string }](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": "Channel", "type": "youtube", "youtubeChannelId": "UC1234567890123456789012"}, 201))
	// Act
	got := decodeSuccess[struct{ ID, Name, YouTubeChannelID string }](t, testkit.Request(t, router, "PATCH", "/api/v1/sources/"+created.ID, map[string]string{"name": " Renamed "}, 200))
	// Assert
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, created.YouTubeChannelID, got.YouTubeChannelID)
	require.Equal(t, "Renamed", got.Name)
}

func TestInvalidYouTubeChannelInputsAreRejected(t *testing.T) {
	for _, input := range []string{"@handle", "https://www.youtube.com/@handle", "https://www.youtube.com/c/custom", "https://www.youtube.com/user/legacy", "http://www.youtube.com/channel/UC1234567890123456789012", "https://www.youtube.com/channel/UC1234567890123456789012?x=1"} {
		t.Run(input, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			// Act
			failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": "Channel", "type": "youtube", "youtubeChannelId": input}, 400))
			// Assert
			require.Equal(t, "VALIDATION", failure.Code)
			require.Contains(t, failure.Message, "channel ID")
			require.Len(t, decodeSuccess[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200)), 1)
		})
	}
}

func TestDuplicateYouTubeChannelConflictsEvenWhenDeactivated(t *testing.T) {
	// Arrange
	router := api.New(t)
	created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": "Channel", "type": "youtube", "youtubeChannelId": "UC1234567890123456789012"}, 201))
	testkit.Request(t, router, "POST", "/api/v1/sources/"+created.ID+"/deactivate", nil, 200)
	// Act
	failure := decodeFailure(t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": "Duplicate", "type": "youtube", "youtubeChannelId": "https://www.youtube.com/channel/UC1234567890123456789012"}, 409))
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
			created := decodeSuccess[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": "Channel", "type": "youtube", "youtubeChannelId": "UC1234567890123456789012"}, 201))
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
