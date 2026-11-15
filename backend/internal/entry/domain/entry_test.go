package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

func TestNewEntryHappyPath(t *testing.T) {
	sourceID := uuid.New()
	thumb := "https://example.com/t.png"
	published := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	e, err := domain.NewEntry(sourceID, domain.EntryMetadata{
		Title:       "  Show HN  ",
		Link:        "  https://example.com/a  ",
		Description: "  desc  ",
		Thumbnail:   &thumb,
		PublishedAt: &published,
	})
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

func TestNewEntryValidation(t *testing.T) {
	sourceID := uuid.New()

	_, err := domain.NewEntry(sourceID, domain.EntryMetadata{Title: "  ", Link: "https://example.com/a"})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	_, err = domain.NewEntry(sourceID, domain.EntryMetadata{Title: "Show HN", Link: ""})
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	_, err = domain.NewEntry(uuid.Nil, domain.EntryMetadata{Title: "Show HN", Link: "https://example.com/a"})
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestEntrySortAtPrefersPublishedAt(t *testing.T) {
	published := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	e := &domain.Entry{PublishedAt: &published, CreatedAt: created}
	require.True(t, published.Equal(e.SortAt()))

	e.PublishedAt = nil
	require.True(t, created.Equal(e.SortAt()))
}

func TestMarkReadAndClearReadUpdateUpdatedAt(t *testing.T) {
	e, err := domain.NewEntry(uuid.New(), domain.EntryMetadata{
		Title: "Show HN",
		Link:  "https://example.com/a",
	})
	require.NoError(t, err)
	originalUpdated := e.UpdatedAt

	time.Sleep(2 * time.Millisecond)
	at := time.Now().UTC()
	e.MarkRead(at)
	require.NotNil(t, e.ReadAt)
	require.True(t, at.Equal(e.ReadAt.UTC()))
	require.True(t, e.UpdatedAt.After(originalUpdated))

	afterMark := e.UpdatedAt
	time.Sleep(2 * time.Millisecond)
	e.ClearRead()
	require.Nil(t, e.ReadAt)
	require.True(t, e.UpdatedAt.After(afterMark))
}

func TestMarkSavedAndClearSavedUpdateUpdatedAt(t *testing.T) {
	e, err := domain.NewEntry(uuid.New(), domain.EntryMetadata{
		Title: "Show HN",
		Link:  "https://example.com/a",
	})
	require.NoError(t, err)
	originalUpdated := e.UpdatedAt

	time.Sleep(2 * time.Millisecond)
	at := time.Now().UTC()
	e.MarkSaved(at)
	require.NotNil(t, e.SavedAt)
	require.True(t, at.Equal(e.SavedAt.UTC()))
	require.True(t, e.UpdatedAt.After(originalUpdated))

	afterMark := e.UpdatedAt
	time.Sleep(2 * time.Millisecond)
	e.ClearSaved()
	require.Nil(t, e.SavedAt)
	require.True(t, e.UpdatedAt.After(afterMark))
}

func TestRefreshMetadataUpdatesFieldsAndUpdatedAt(t *testing.T) {
	e, err := domain.NewEntry(uuid.New(), domain.EntryMetadata{
		Title:       "Original",
		Link:        "https://example.com/a",
		Description: "old",
	})
	require.NoError(t, err)
	createdAt := e.CreatedAt
	originalUpdated := e.UpdatedAt
	originalLink := e.Link

	time.Sleep(2 * time.Millisecond)
	thumb := "https://example.com/new.png"
	published := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	err = e.RefreshMetadata(domain.EntryMetadata{
		Title:       "  Updated  ",
		Link:        "  https://example.com/a  ",
		Description: "  refreshed  ",
		Thumbnail:   &thumb,
		PublishedAt: &published,
	})
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
	e, err := domain.NewEntry(uuid.New(), domain.EntryMetadata{
		Title: "Original",
		Link:  "https://example.com/a",
	})
	require.NoError(t, err)

	err = e.RefreshMetadata(domain.EntryMetadata{Title: "  ", Link: "https://example.com/a"})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	err = e.RefreshMetadata(domain.EntryMetadata{Title: "Updated", Link: ""})
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}
