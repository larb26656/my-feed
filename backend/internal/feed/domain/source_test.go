package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/quick-feed/backend/internal/feed/domain"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

func TestNewSourceHappyPath(t *testing.T) {
	s, err := domain.NewSource("  HN RSS  ")
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, s.ID)
	require.Equal(t, "HN RSS", s.Name)
	require.False(t, s.CreatedAt.IsZero())
	require.Equal(t, s.CreatedAt, s.UpdatedAt)
}

func TestNewSourceRejectsBlankName(t *testing.T) {
	_, err := domain.NewSource("   ")
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	_, err = domain.NewSource("")
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}
