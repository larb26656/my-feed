package http_test

import (
	"fmt"
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

func TestReaderCanSubscribeEditPauseAndResumeAnRSSSource(t *testing.T) {
	// Arrange
	router := api.New(t)

	// Act: subscribe to the RSS source.
	created := testkit.Decode[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": " Daily news ", "type": "rss", "rssUrl": " https://example.test/rss "}, 201))

	// Assert
	require.Equal(t, "Daily news", created.Name)
	require.Equal(t, "rss", created.Type)
	require.Equal(t, "https://example.test/rss", *created.RSSURL)
	require.True(t, created.Active)
	path := "/api/v1/sources/" + created.ID

	// Act: edit the subscription.
	updated := testkit.Decode[source](t, testkit.Request(t, router, "PATCH", path, map[string]string{"name": "Weekly news", "rssUrl": "https://example.test/weekly"}, 200))

	// Assert
	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, created.CreatedAt, updated.CreatedAt)
	require.Equal(t, "Weekly news", updated.Name)
	require.Equal(t, "https://example.test/weekly", *updated.RSSURL)
	require.Equal(t, "rss", updated.Type)
	for i := 0; i < 2; i++ {
		// Act: pause the subscription again.
		paused := testkit.Decode[source](t, testkit.Request(t, router, "POST", path+"/deactivate", nil, 200))

		// Assert
		require.False(t, paused.Active, "pausing is idempotent")
	}

	// Act: list active sources.
	active := testkit.Decode[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))

	// Assert
	require.Len(t, active, 1)
	require.Equal(t, "manual", active[0].Type)

	// Act: include paused sources in the list.
	all := testkit.Decode[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/?includeDeactivated=true", nil, 200))

	// Assert
	require.Len(t, all, 2)

	// Act: fetch the stored subscription.
	got := testkit.Decode[source](t, testkit.Request(t, router, "GET", path, nil, 200))

	// Assert
	require.False(t, got.Active, "paused sources remain directly accessible")
	require.Equal(t, updated.Name, got.Name)
	for i := 0; i < 2; i++ {
		// Act: resume the subscription again.
		resumed := testkit.Decode[source](t, testkit.Request(t, router, "POST", path+"/reactivate", nil, 200))

		// Assert
		require.True(t, resumed.Active, "resuming is idempotent")
	}

	// Act: list active sources after resuming.
	active = testkit.Decode[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))

	// Assert
	require.Len(t, active, 2)
}

func TestSourceValidationReturnsActionableErrorsWithoutCreatingSubscriptions(t *testing.T) {
	// Arrange
	router := api.New(t)
	for _, scenario := range []struct{ name, body, message string }{
		{"blank name", `{"name":" ","type":"rss","rssUrl":"https://example.test/rss"}`, "name is required"},
		{"blank feed URL", `{"name":"News","type":"rss","rssUrl":" "}`, "rss url is required"},
		{"reader cannot create manual source", `{"name":"Bookmarks","type":"manual"}`, "manual sources cannot be created by the reader"},
		{"unsupported type", `{"name":"News","type":"unknown"}`, "source type must be rss"},
		{"malformed JSON", `{`, "invalid json body"},
		{"unknown field", `{"extra":true}`, "invalid json body"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Act: submit the request that should be rejected.
			failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "POST", "/api/v1/sources/", scenario.body, 400))

			// Assert
			require.Equal(t, "validation", failure.Code)
			require.Equal(t, scenario.message, failure.Error)
		})
	}

	// Act: list the system sources.
	sources := testkit.Decode[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))

	// Assert
	require.Len(t, sources, 1, "only the system Saved links source should exist")
}

func TestDuplicateFeedURLsAreRejectedAndFailedEditsLeaveTheSubscriptionUnchanged(t *testing.T) {
	// Arrange
	router := api.New(t)
	first := map[string]string{"name": "Daily", "type": "rss", "rssUrl": "https://example.test/daily"}
	testkit.Request(t, router, "POST", "/api/v1/sources/", first, 201)

	// Act: subscribe to an already registered feed URL.
	conflict := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "POST", "/api/v1/sources/", first, 409))

	// Assert
	require.Equal(t, "conflict", conflict.Code)

	// Arrange: a second subscription with a different feed URL.
	second := testkit.Decode[source](t, testkit.Request(t, router, "POST", "/api/v1/sources/", map[string]string{"name": "Weekly", "type": "rss", "rssUrl": "https://example.test/weekly"}, 201))
	path := "/api/v1/sources/" + second.ID

	// Act: edit the second subscription to use the first feed URL.
	conflict = testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "PATCH", path, map[string]string{"name": "Changed", "rssUrl": first["rssUrl"]}, 409))

	// Assert
	require.Equal(t, "conflict", conflict.Code)

	// Act: fetch the stored subscription.
	got := testkit.Decode[source](t, testkit.Request(t, router, "GET", path, nil, 200))

	// Assert
	require.Equal(t, second, got, "a conflicting edit must not persist any fields")

	// Act: keep the existing URL while renaming the subscription (status checked by Request).
	testkit.Request(t, router, "PATCH", path, map[string]string{"name": "Renamed", "rssUrl": *second.RSSURL}, 200)
	for _, body := range []map[string]string{{"name": " ", "rssUrl": *second.RSSURL}, {"name": "Weekly", "rssUrl": " "}} {
		// Act: submit the request that should be rejected.
		failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, "PATCH", path, body, 400))

		// Assert
		require.Equal(t, "validation", failure.Code)
	}
}

func TestSavedLinksIsProtectedAndUnknownSourcesReturnNotFound(t *testing.T) {
	// Arrange
	router := api.New(t)

	// Act: list the system sources.
	sources := testkit.Decode[[]source](t, testkit.Request(t, router, "GET", "/api/v1/sources/", nil, 200))

	// Assert
	require.Len(t, sources, 1)
	require.Equal(t, "Saved links", sources[0].Name)
	require.Nil(t, sources[0].RSSURL)
	manualPath := "/api/v1/sources/" + sources[0].ID
	for _, action := range []struct{ method, suffix string }{{"PATCH", ""}, {"POST", "/deactivate"}, {"POST", "/reactivate"}} {
		t.Run(action.method+action.suffix, func(t *testing.T) {
			// Act: submit the request that should be rejected.
			failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, action.method, manualPath+action.suffix, map[string]string{"name": "Changed", "rssUrl": "https://example.test/rss"}, 422))

			// Assert
			require.Equal(t, "failed_precondition", failure.Code)
		})
	}
	for _, id := range []string{"bad", uuid.NewString()} {
		status, code := 404, "not_found"
		if id == "bad" {
			status, code = 400, "validation"
		}
		for _, action := range []struct{ method, suffix string }{{"GET", ""}, {"PATCH", ""}, {"POST", "/deactivate"}, {"POST", "/reactivate"}} {
			t.Run(fmt.Sprintf("%s %s%s", action.method, id, action.suffix), func(t *testing.T) {
				// Act: submit the request that should be rejected.
				failure := testkit.Decode[httpx.ErrorBody](t, testkit.Request(t, router, action.method, "/api/v1/sources/"+id+action.suffix, map[string]string{"name": "News", "rssUrl": "https://example.test/rss"}, status))

				// Assert
				require.Equal(t, code, failure.Code)
			})
		}
	}
}
