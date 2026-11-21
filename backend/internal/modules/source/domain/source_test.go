package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/stretchr/testify/require"
)

func TestNewRSSSourceNormalizesSubscriptionDetails(t *testing.T) {
	// Arrange
	name, feedURL := " News ", " https://example.test/rss "

	// Act
	source, err := domain.NewRSSSource(name, feedURL)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "News", source.Name)
	require.Equal(t, "https://example.test/rss", source.RSSURL)
}

func TestNewRSSSourceStartsActiveWithAnRSSIdentity(t *testing.T) {
	// Arrange
	name, feedURL := "News", "https://example.test/rss"

	// Act
	source, err := domain.NewRSSSource(name, feedURL)

	// Assert
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, source.ID)
	require.Equal(t, domain.TypeRSS, source.Type)
	require.True(t, source.Active)
	require.False(t, source.CreatedAt.IsZero())
	require.Equal(t, source.CreatedAt, source.UpdatedAt)
}

func TestNewManualSourceStartsActiveWithoutAFeedURL(t *testing.T) {
	// Arrange
	name := domain.SavedLinksName

	// Act
	source, err := domain.NewManualSource(name)

	// Assert
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, source.ID)
	require.Equal(t, domain.TypeManual, source.Type)
	require.True(t, source.Active)
	require.Empty(t, source.RSSURL)
}

func TestNewRSSSourceRejectsBlankNames(t *testing.T) {
	for _, scenario := range []struct{ name, value string }{{"empty", ""}, {"whitespace", " "}} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			feedURL := "https://example.test/rss"

			// Act
			_, err := domain.NewRSSSource(scenario.value, feedURL)

			// Assert
			var failure *apperr.AppError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, apperr.InvalidInput.Code, failure.Code)
		})
	}
}

func TestNewRSSSourceRejectsMissingFeedURLs(t *testing.T) {
	// Arrange
	name, feedURL := "News", " "

	// Act
	_, err := domain.NewRSSSource(name, feedURL)

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, apperr.InvalidInput.Code, failure.Code)
}

func TestPausingAnRSSSourceUpdatesItsStateAndTimestamp(t *testing.T) {
	// Arrange
	source, err := domain.NewRSSSource("News", "https://example.test/rss")
	require.NoError(t, err)
	previous := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	source.UpdatedAt = previous

	// Act
	err = source.Deactivate()

	// Assert
	require.NoError(t, err)
	require.False(t, source.Active)
	require.True(t, source.UpdatedAt.After(previous))
}

func TestManualSourcesCannotBePaused(t *testing.T) {
	// Arrange
	source, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)

	// Act
	err = source.Deactivate()

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, domain.ErrManualSourceImmutable.Code, failure.Code)
	require.True(t, source.Active)
}

func TestResumingAnRSSSourceUpdatesItsStateAndTimestamp(t *testing.T) {
	// Arrange
	source, err := domain.NewRSSSource("News", "https://example.test/rss")
	require.NoError(t, err)
	source.Active = false
	previous := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	source.UpdatedAt = previous

	// Act
	err = source.Reactivate()

	// Assert
	require.NoError(t, err)
	require.True(t, source.Active)
	require.True(t, source.UpdatedAt.After(previous))
}

func TestManualSourcesCannotBeResumed(t *testing.T) {
	// Arrange
	source, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)

	// Act
	err = source.Reactivate()

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, domain.ErrManualSourceImmutable.Code, failure.Code)
	require.True(t, source.Active)
}

func TestEditingRSSDetailsNormalizesFieldsAndPreservesTheSourceType(t *testing.T) {
	// Arrange
	source, err := domain.NewRSSSource("News", "https://example.test/rss")
	require.NoError(t, err)
	previous := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	source.UpdatedAt = previous

	// Act
	err = source.UpdateDetails(" Weekly news ", " https://example.test/weekly ")

	// Assert
	require.NoError(t, err)
	require.Equal(t, "Weekly news", source.Name)
	require.Equal(t, "https://example.test/weekly", source.RSSURL)
	require.Equal(t, domain.TypeRSS, source.Type)
	require.True(t, source.UpdatedAt.After(previous))
}

func TestInvalidSourceEditsLeaveTheSourceUnchanged(t *testing.T) {
	for _, scenario := range []struct{ name, title, url string }{
		{"blank name", " ", "https://example.test/new"}, {"blank feed URL", "Changed", " "},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			source, err := domain.NewRSSSource("News", "https://example.test/rss")
			require.NoError(t, err)
			original := *source

			// Act
			err = source.UpdateDetails(scenario.title, scenario.url)

			// Assert
			var failure *apperr.AppError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, apperr.InvalidInput.Code, failure.Code)
			require.Equal(t, original, *source)
		})
	}
}

func TestManualSourcesCannotBeEdited(t *testing.T) {
	// Arrange
	source, err := domain.NewManualSource(domain.SavedLinksName)
	require.NoError(t, err)
	original := *source

	// Act
	err = source.UpdateDetails("Bookmarks", "https://example.test/rss")

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, domain.ErrManualSourceImmutable.Code, failure.Code)
	require.Equal(t, original, *source)
}

func TestNewYouTubeSourceNormalizesIdentity(t *testing.T) {
	// Arrange
	name, channelID := " Channel ", " UC1234567890123456789012 "
	// Act
	src, err := domain.NewYouTubeSource(name, channelID)
	// Assert
	require.NoError(t, err)
	require.Equal(t, "Channel", src.Name)
	require.Equal(t, "UC1234567890123456789012", src.YouTubeChannelID)
}

func TestNewYouTubeSourceRejectsInvalidChannelID(t *testing.T) {
	// Arrange
	channelID := "@handle"
	// Act
	_, err := domain.NewYouTubeSource("Channel", channelID)
	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, apperr.InvalidInput.Code, failure.Code)
}
