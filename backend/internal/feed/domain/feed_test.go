package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/larb26656/quick-feed/backend/internal/feed/domain"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

func TestFeedLifecycle(t *testing.T) {
	f, err := domain.NewFeed("morning", "Morning Brief", "daily headlines")
	require.NoError(t, err)
	require.Equal(t, domain.StatusDraft, f.Status)

	require.Error(t, f.Archive())
	require.NoError(t, f.Publish())
	require.Equal(t, domain.StatusPublished, f.Status)

	require.Error(t, f.Publish())
	require.NoError(t, f.Archive())
	require.Equal(t, domain.StatusArchived, f.Status)
	require.Error(t, f.Archive())
}

func TestNewFeedValidation(t *testing.T) {
	_, err := domain.NewFeed("", "Title", "")
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	_, err = domain.NewFeed("slug", "  ", "")
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}
