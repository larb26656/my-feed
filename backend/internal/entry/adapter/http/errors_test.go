package http_test

import (
	"context"
	"errors"
	"testing"

	entryhttp "github.com/larb26656/my-feed/backend/internal/entry/adapter/http"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

// A focused stub forces an infrastructure failure without corrupting SQLite.
type unavailableSavedCount struct{ entryhttp.EntryAPI }

func (unavailableSavedCount) CountSavedEntries(context.Context) (int, error) {
	return 0, errors.New("private database details")
}

func TestStorageFailuresReturnAnInternalErrorWithoutLeakingDatabaseDetails(t *testing.T) {
	// Arrange
	router := httpx.NewRouter()
	entryhttp.NewHandler(unavailableSavedCount{}).RegisterRoutes(router.Group("/entries"))

	// Act: request the saved count while storage is unavailable.
	response := testkit.Request(t, router, "GET", "/entries/saved-count", nil, 500)
	failure := decodeFailure(t, response)

	// Assert
	require.Equal(t, "INTERNAL_SERVER_ERROR", failure.Code)
	require.Equal(t, "Internal server error", failure.Message)
	require.NotContains(t, response.Body.String(), "private database details")
}
