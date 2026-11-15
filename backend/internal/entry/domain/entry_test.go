package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/stretchr/testify/require"
)

func newEntry(t *testing.T) *domain.Entry {
	t.Helper()
	entry, err := domain.NewEntry(uuid.New(), domain.EntryMetadata{Title: "Original", Link: "https://example.test/a", Description: "Old summary"})
	require.NoError(t, err)
	return entry
}

func TestNewEntryTrimsMetadata(t *testing.T) {
	// Arrange
	sourceID := uuid.New()
	thumbnail := "https://example.test/photo.png"
	published := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	metadata := domain.EntryMetadata{Title: " News ", Link: " https://example.test/a ", Description: " Summary ", Thumbnail: &thumbnail, PublishedAt: &published}

	// Act
	entry, err := domain.NewEntry(sourceID, metadata)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "News", entry.Title)
	require.Equal(t, "https://example.test/a", entry.Link)
	require.Equal(t, "Summary", entry.Description)
	require.Equal(t, &thumbnail, entry.Thumbnail)
	require.Equal(t, &published, entry.PublishedAt)
}

func TestNewEntryHasAnIdentityAndStartsUnreadAndUnsaved(t *testing.T) {
	// Arrange
	sourceID := uuid.New()

	// Act
	entry, err := domain.NewEntry(sourceID, domain.EntryMetadata{Title: "News", Link: "https://example.test/a"})

	// Assert
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, entry.ID)
	require.Equal(t, sourceID, entry.SourceID)
	require.False(t, entry.CreatedAt.IsZero())
	require.Equal(t, entry.CreatedAt, entry.UpdatedAt)
	require.Nil(t, entry.ReadAt)
	require.Nil(t, entry.SavedAt)
}

func TestNewEntryRejectsMissingRequiredFields(t *testing.T) {
	for _, scenario := range []struct {
		name, title, link string
		sourceID          uuid.UUID
	}{
		{"blank title", " ", "https://example.test/a", uuid.New()},
		{"blank link", "News", "", uuid.New()},
		{"missing source", "News", "https://example.test/a", uuid.Nil},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			metadata := domain.EntryMetadata{Title: scenario.title, Link: scenario.link}

			// Act
			_, err := domain.NewEntry(scenario.sourceID, metadata)

			// Assert
			var failure *apperr.Error
			require.ErrorAs(t, err, &failure)
			require.Equal(t, apperr.CodeValidation, failure.Code)
		})
	}
}

func TestEntrySortAtPrefersThePublicationDate(t *testing.T) {
	// Arrange
	published := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	entry := &domain.Entry{PublishedAt: &published, CreatedAt: published.Add(-time.Hour)}

	// Act
	sortAt := entry.SortAt()

	// Assert
	require.Equal(t, published, sortAt)
}

func TestEntrySortAtFallsBackToTheCreationDate(t *testing.T) {
	// Arrange
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entry := &domain.Entry{CreatedAt: created}

	// Act
	sortAt := entry.SortAt()

	// Assert
	require.Equal(t, created, sortAt)
}

func TestMarkReadRecordsTheReadTimeAndUpdatesTheEntryTimestamp(t *testing.T) {
	// Arrange
	entry := newEntry(t)
	previous := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	entry.UpdatedAt = previous
	readAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Act
	entry.MarkRead(readAt)

	// Assert
	require.Equal(t, &readAt, entry.ReadAt)
	require.True(t, entry.UpdatedAt.After(previous))
}

func TestClearReadRemovesTheReadTimeAndUpdatesTheEntryTimestamp(t *testing.T) {
	// Arrange
	entry := newEntry(t)
	previous := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	entry.ReadAt = &previous
	entry.UpdatedAt = previous

	// Act
	entry.ClearRead()

	// Assert
	require.Nil(t, entry.ReadAt)
	require.True(t, entry.UpdatedAt.After(previous))
}

func TestMarkSavedRecordsTheSavedTimeAndUpdatesTheEntryTimestamp(t *testing.T) {
	// Arrange
	entry := newEntry(t)
	previous := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	entry.UpdatedAt = previous
	savedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Act
	entry.MarkSaved(savedAt)

	// Assert
	require.Equal(t, &savedAt, entry.SavedAt)
	require.True(t, entry.UpdatedAt.After(previous))
}

func TestClearSavedRemovesTheSavedTimeAndUpdatesTheEntryTimestamp(t *testing.T) {
	// Arrange
	entry := newEntry(t)
	previous := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	entry.SavedAt = &previous
	entry.UpdatedAt = previous

	// Act
	entry.ClearSaved()

	// Assert
	require.Nil(t, entry.SavedAt)
	require.True(t, entry.UpdatedAt.After(previous))
}

func TestRefreshMetadataUpdatesMutableFieldsWithoutChangingIdentity(t *testing.T) {
	// Arrange
	entry := newEntry(t)
	createdAt, originalLink, originalID := entry.CreatedAt, entry.Link, entry.ID
	previous := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	entry.UpdatedAt = previous
	thumbnail := "https://example.test/new.png"
	published := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	metadata := domain.EntryMetadata{Title: " Updated ", Link: " https://example.test/a ", Description: " Refreshed ", Thumbnail: &thumbnail, PublishedAt: &published}

	// Act
	err := entry.RefreshMetadata(metadata)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "Updated", entry.Title)
	require.Equal(t, "Refreshed", entry.Description)
	require.Equal(t, &thumbnail, entry.Thumbnail)
	require.Equal(t, &published, entry.PublishedAt)
	require.Equal(t, originalID, entry.ID)
	require.Equal(t, originalLink, entry.Link)
	require.Equal(t, createdAt, entry.CreatedAt)
	require.True(t, entry.UpdatedAt.After(previous))
}

func TestRefreshMetadataRejectsBlankRequiredFields(t *testing.T) {
	for _, scenario := range []struct{ name, title, link string }{
		{"blank title", " ", "https://example.test/a"}, {"blank link", "Updated", ""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// Arrange
			entry := newEntry(t)
			original := *entry

			// Act
			err := entry.RefreshMetadata(domain.EntryMetadata{Title: scenario.title, Link: scenario.link})

			// Assert
			var failure *apperr.Error
			require.ErrorAs(t, err, &failure)
			require.Equal(t, apperr.CodeValidation, failure.Code)
			require.Equal(t, original, *entry, "invalid metadata must not partially update the entry")
		})
	}
}
