package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

func TestNewEntryTrimsMetadataAndStartsUnreadAndUnsaved(t *testing.T) {
	// Arrange
	sourceID := uuid.New()
	thumb := "https://example.com/t.png"
	published := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	// Act: create an entry from padded metadata.
	e, err := domain.NewEntry(sourceID, domain.EntryMetadata{
		Title:       "  Show HN  ",
		Link:        "  https://example.com/a  ",
		Description: "  desc  ",
		Thumbnail:   &thumb,
		PublishedAt: &published,
	})

	// Assert
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, e.ID)
	require.Equal(t, sourceID, e.SourceID)
	require.Equal(t, "Show HN", e.Title)
	require.Equal(t, "https://example.com/a", e.Link)
	require.Equal(t, "desc", e.Description)
	require.NotNil(t, e.Thumbnail)
	require.Equal(t, thumb, *e.Thumbnail)
	require.NotNil(t, e.PublishedAt)
	require.True(t, published.Equal(e.PublishedAt.UTC()))
	require.False(t, e.CreatedAt.IsZero())
	require.Equal(t, e.CreatedAt, e.UpdatedAt)
	require.Nil(t, e.ReadAt)
	require.Nil(t, e.SavedAt)
}

func TestNewEntryRequiresTitleLinkAndSource(t *testing.T) {
	// Arrange
	sourceID := uuid.New()

	// Act: create an entry without a required field.
	_, err := domain.NewEntry(sourceID, domain.EntryMetadata{Title: "  ", Link: "https://example.com/a"})

	// Assert
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	// Act: create an entry without another required field.
	_, err = domain.NewEntry(sourceID, domain.EntryMetadata{Title: "Show HN", Link: ""})

	// Assert
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	// Act: create an entry without another required field.
	_, err = domain.NewEntry(uuid.Nil, domain.EntryMetadata{Title: "Show HN", Link: "https://example.com/a"})

	// Assert
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestEntrySortAtPrefersPublishedAt(t *testing.T) {
	// Arrange
	published := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	e := &domain.Entry{PublishedAt: &published, CreatedAt: created}

	// Act & Assert: prefer the publication date when available.
	require.True(t, published.Equal(e.SortAt()))

	// Arrange: remove the publication date.
	e.PublishedAt = nil

	// Act & Assert: fall back to the creation date.
	require.True(t, created.Equal(e.SortAt()))
}

func TestMarkReadAndClearReadUpdateUpdatedAt(t *testing.T) {
	// Arrange
	e, err := domain.NewEntry(uuid.New(), domain.EntryMetadata{
		Title: "Show HN",
		Link:  "https://example.com/a",
	})
	require.NoError(t, err)
	e.UpdatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	originalUpdated := e.UpdatedAt

	at := time.Now().UTC()

	// Act: mark the entry read.
	e.MarkRead(at)

	// Assert
	require.NotNil(t, e.ReadAt)
	require.True(t, at.Equal(e.ReadAt.UTC()))
	require.True(t, e.UpdatedAt.After(originalUpdated))

	// Arrange: reset the historical timestamp before clearing the choice.
	e.UpdatedAt = originalUpdated
	afterMark := e.UpdatedAt

	// Act: clear the read choice.
	e.ClearRead()

	// Assert
	require.Nil(t, e.ReadAt)
	require.True(t, e.UpdatedAt.After(afterMark))
}

func TestMarkSavedAndClearSavedUpdateUpdatedAt(t *testing.T) {
	// Arrange
	e, err := domain.NewEntry(uuid.New(), domain.EntryMetadata{
		Title: "Show HN",
		Link:  "https://example.com/a",
	})
	require.NoError(t, err)
	e.UpdatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	originalUpdated := e.UpdatedAt

	at := time.Now().UTC()

	// Act: mark the entry saved.
	e.MarkSaved(at)

	// Assert
	require.NotNil(t, e.SavedAt)
	require.True(t, at.Equal(e.SavedAt.UTC()))
	require.True(t, e.UpdatedAt.After(originalUpdated))

	// Arrange: reset the historical timestamp before clearing the choice.
	e.UpdatedAt = originalUpdated
	afterMark := e.UpdatedAt

	// Act: clear the saved choice.
	e.ClearSaved()

	// Assert
	require.Nil(t, e.SavedAt)
	require.True(t, e.UpdatedAt.After(afterMark))
}

func TestRefreshMetadataUpdatesFieldsAndUpdatedAt(t *testing.T) {
	// Arrange
	e, err := domain.NewEntry(uuid.New(), domain.EntryMetadata{
		Title:       "Original",
		Link:        "https://example.com/a",
		Description: "old",
	})
	require.NoError(t, err)
	createdAt := e.CreatedAt
	e.UpdatedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	originalUpdated := e.UpdatedAt
	originalLink := e.Link

	thumb := "https://example.com/new.png"
	published := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	// Act: refresh entry metadata.
	err = e.RefreshMetadata(domain.EntryMetadata{
		Title:       "  Updated  ",
		Link:        "  https://example.com/a  ",
		Description: "  refreshed  ",
		Thumbnail:   &thumb,
		PublishedAt: &published,
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, "Updated", e.Title)
	require.Equal(t, originalLink, e.Link)
	require.Equal(t, "refreshed", e.Description)
	require.NotNil(t, e.Thumbnail)
	require.Equal(t, thumb, *e.Thumbnail)
	require.NotNil(t, e.PublishedAt)
	require.True(t, published.Equal(e.PublishedAt.UTC()))
	require.Equal(t, createdAt, e.CreatedAt)
	require.True(t, e.UpdatedAt.After(originalUpdated))
}

func TestRefreshMetadataRejectsBlankTitleOrLink(t *testing.T) {
	// Arrange
	e, err := domain.NewEntry(uuid.New(), domain.EntryMetadata{
		Title: "Original",
		Link:  "https://example.com/a",
	})
	require.NoError(t, err)

	// Act: refresh entry metadata.
	err = e.RefreshMetadata(domain.EntryMetadata{Title: "  ", Link: "https://example.com/a"})

	// Assert
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	// Act: refresh entry metadata.
	err = e.RefreshMetadata(domain.EntryMetadata{Title: "Updated", Link: ""})

	// Assert
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}
