package apperr_test

import (
	"errors"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/stretchr/testify/require"
)

func TestDefinitionNewCreatesAppErrorWithoutCause(t *testing.T) {
	// Arrange
	def := apperr.Definition{Kind: apperr.KindNotFound, Code: "NOT_FOUND"}

	// Act
	err := def.New("Source not found")

	// Assert
	require.Equal(t, apperr.KindNotFound, err.Kind)
	require.Equal(t, "NOT_FOUND", err.Code)
	require.Equal(t, "Source not found", err.Message)
	require.Nil(t, err.Cause)
	require.Nil(t, err.Detail)
}

func TestDefinitionWrapPreservesUnderlyingCause(t *testing.T) {
	// Arrange
	def := apperr.Definition{Kind: apperr.KindInternal, Code: "INTERNAL"}
	cause := errors.New("sqlite: no such table")

	// Act
	err := def.Wrap(cause, "Failed to load source")

	// Assert
	require.Equal(t, apperr.KindInternal, err.Kind)
	require.Equal(t, "INTERNAL", err.Code)
	require.Equal(t, "Failed to load source", err.Message)
	require.Equal(t, cause, err.Cause)
	require.ErrorIs(t, err, cause)
}

func TestWithDetailAttachesStructuredDetail(t *testing.T) {
	// Arrange
	detail := struct {
		Field string `json:"field"`
	}{Field: "url"}
	err := apperr.InvalidInput.New("Invalid input")

	// Act
	got := err.WithDetail(detail)

	// Assert
	require.Same(t, err, got)
	require.Equal(t, detail, got.Detail)
}

func TestAppErrorImplementsErrorContract(t *testing.T) {
	// Arrange
	cause := errors.New("connection refused")
	err := apperr.Definition{Kind: apperr.KindUnavailable, Code: "UNAVAILABLE"}.
		Wrap(cause, "Service unavailable")

	// Act & Assert
	require.Equal(t, "Service unavailable", err.Error())
	require.Equal(t, cause, errors.Unwrap(err))
}
