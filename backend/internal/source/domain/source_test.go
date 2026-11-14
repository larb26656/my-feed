package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
	"github.com/larb26656/quick-feed/backend/internal/source/domain"
)

func TestNewRSSSourceHappyPath(t *testing.T) {
	s, err := domain.NewRSSSource("  HN RSS  ", "  https://hn.example/rss  ")
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, s.ID)
	require.Equal(t, "HN RSS", s.Name)
	require.Equal(t, domain.TypeRSS, s.Type)
	require.Equal(t, "https://hn.example/rss", s.RSSURL)
	require.False(t, s.CreatedAt.IsZero())
	require.Equal(t, s.CreatedAt, s.UpdatedAt)
}

func TestNewRSSSourceRejectsBlankName(t *testing.T) {
	_, err := domain.NewRSSSource("   ", "https://hn.example/rss")
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	_, err = domain.NewRSSSource("", "https://hn.example/rss")
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestNewRSSSourceRejectsMissingURL(t *testing.T) {
	_, err := domain.NewRSSSource("HN RSS", "   ")
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestSourceTypeFixedAtConstruction(t *testing.T) {
	rss, err := domain.NewRSSSource("HN", "https://hn.example/rss")
	require.NoError(t, err)
	require.Equal(t, domain.TypeRSS, rss.Type)

	manual, err := domain.NewManualSource("Saved links")
	require.NoError(t, err)
	require.Equal(t, domain.TypeManual, manual.Type)
	require.Empty(t, manual.RSSURL)

	// Type is set only by constructors; no domain operation changes it.
	require.Equal(t, domain.TypeRSS, rss.Type)
	require.Equal(t, domain.TypeManual, manual.Type)
}

func TestNewSourcesStartActive(t *testing.T) {
	rss, err := domain.NewRSSSource("HN", "https://hn.example/rss")
	require.NoError(t, err)
	require.True(t, rss.Active)

	manual, err := domain.NewManualSource("Saved links")
	require.NoError(t, err)
	require.True(t, manual.Active)
}

func TestDeactivateRSSSource(t *testing.T) {
	s, err := domain.NewRSSSource("HN", "https://hn.example/rss")
	require.NoError(t, err)
	before := s.UpdatedAt

	time.Sleep(2 * time.Millisecond)
	require.NoError(t, s.Deactivate())
	require.False(t, s.Active)
	require.True(t, s.UpdatedAt.After(before))
}

func TestDeactivateManualSourceRejected(t *testing.T) {
	s, err := domain.NewManualSource("Saved links")
	require.NoError(t, err)

	err = s.Deactivate()
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
	require.True(t, s.Active)
}

func TestReactivateRSSSource(t *testing.T) {
	s, err := domain.NewRSSSource("HN", "https://hn.example/rss")
	require.NoError(t, err)
	require.NoError(t, s.Deactivate())
	require.False(t, s.Active)

	before := s.UpdatedAt
	time.Sleep(2 * time.Millisecond)
	require.NoError(t, s.Reactivate())
	require.True(t, s.Active)
	require.True(t, s.UpdatedAt.After(before))
}

func TestReactivateManualSourceRejected(t *testing.T) {
	s, err := domain.NewManualSource("Saved links")
	require.NoError(t, err)

	err = s.Reactivate()
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
	require.True(t, s.Active)
}
