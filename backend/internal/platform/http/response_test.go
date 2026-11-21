package httpx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/stretchr/testify/require"
)

func TestWriteErrorReturnsValidationDetails(t *testing.T) {
	// Arrange
	recorder := httptest.NewRecorder()
	err := fmt.Errorf("entry: %w", apperr.InvalidInput.New("title is required"))

	// Act
	httpx.WriteError(recorder, err)

	// Assert
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t, `{"error":"title is required","code":"validation"}`, recorder.Body.String())
}

func TestWriteErrorReturnsEntryNotFound(t *testing.T) {
	// Arrange
	recorder := httptest.NewRecorder()
	err := fmt.Errorf("lookup: %w", domain.ErrEntryNotFound.New("entry not found"))

	// Act
	httpx.WriteError(recorder, err)

	// Assert
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.JSONEq(t, `{"error":"entry not found","code":"ENTRY_NOT_FOUND"}`, recorder.Body.String())
}
