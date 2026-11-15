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
	router := entryhttp.NewHandler(unavailableSavedCount{}).Routes()

	// Act: request the saved count while storage is unavailable.
	response := testkit.Request(t, router, "GET", "/saved-count", nil, 500)
	failure := testkit.Decode[httpx.ErrorBody](t, response)

	// Assert
	require.Equal(t, "internal", failure.Code)
	require.Equal(t, "internal server error", failure.Error)
	require.NotContains(t, response.Body.String(), "private database details")
}
