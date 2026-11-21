package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	entrydomain "github.com/larb26656/my-feed/backend/internal/modules/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	sourcedomain "github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func entryRepository(t *testing.T) (*Repository, *sourcedomain.Source) {
	t.Helper()
	pool := testkit.NewDatabase(t)
	source, err := sourcedomain.NewRSSSource("News", "https://example.test/rss")
	require.NoError(t, err)
	require.NoError(t, sourcesqlite.NewRepository(pool.DB).CreateSource(context.Background(), source))
	return NewRepository(pool.DB), source
}

func TestSQLiteRejectsAnEntryWhoseSourceDoesNotExist(t *testing.T) {
	// Arrange
	pool := testkit.NewDatabase(t)
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()

	// Act
	_, err := pool.Exec(`INSERT INTO entries (id, source_id, title, link, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), uuid.NewString(), "Orphan", "https://example.test/orphan", now, now)

	// Assert
	require.Error(t, err, "SQLite must enforce the Source foreign key")
}

func TestSQLiteRejectsDuplicateEntrySourceAndLink(t *testing.T) {
	// Arrange
	repo, source := entryRepository(t)
	first, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "First", Link: "https://example.test/a"})
	require.NoError(t, err)
	_, err = repo.CreateEntry(context.Background(), first)
	require.NoError(t, err)
	duplicate, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "Duplicate", Link: first.Link})
	require.NoError(t, err)

	// Act
	_, err = repo.CreateEntry(context.Background(), duplicate)

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, apperr.ResourceConflict.Code, failure.Code)
}

func TestSQLitePreservesPublicationTimestampPrecision(t *testing.T) {
	// Arrange
	repo, source := entryRepository(t)
	published := time.Date(2025, 4, 3, 2, 1, 0, 123456789, time.UTC)
	article, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "News", Link: "https://example.test/a", PublishedAt: &published})
	require.NoError(t, err)

	// Act
	created, err := repo.CreateEntry(context.Background(), article)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, created.PublishedAt)
	require.Equal(t, published.UnixNano(), created.PublishedAt.UnixNano())
}

func TestSQLitePersistsRefreshedMetadataAndReaderState(t *testing.T) {
	// Arrange
	repo, source := entryRepository(t)
	article, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "Original", Link: "https://example.test/a"})
	require.NoError(t, err)
	_, err = repo.CreateEntry(context.Background(), article)
	require.NoError(t, err)
	published := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, article.RefreshMetadata(entrydomain.EntryMetadata{Title: "Updated", Link: article.Link, Description: "New summary", PublishedAt: &published}))
	readAt := published.Add(time.Minute)
	savedAt := published.Add(2 * time.Minute)
	article.MarkRead(readAt)
	article.MarkSaved(savedAt)

	// Act
	updated, err := repo.UpdateEntry(context.Background(), article)
	require.NoError(t, err)
	got, err := repo.GetEntryByID(context.Background(), article.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "Updated", got.Title)
	require.Equal(t, "New summary", got.Description)
	require.NotNil(t, got.ReadAt)
	require.NotNil(t, got.SavedAt)
	require.Equal(t, readAt.UnixNano(), got.ReadAt.UnixNano())
	require.Equal(t, savedAt.UnixNano(), got.SavedAt.UnixNano())
	require.Equal(t, updated, got)
}

func TestSQLiteListsEntriesNewestFirst(t *testing.T) {
	// Arrange
	repo, source := entryRepository(t)
	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := older.Add(time.Hour)
	var ids []uuid.UUID
	for i, at := range []time.Time{older, newer} {
		article, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "News", Link: []string{"https://example.test/old", "https://example.test/new"}[i], PublishedAt: &at})
		require.NoError(t, err)
		_, err = repo.CreateEntry(context.Background(), article)
		require.NoError(t, err)
		ids = append(ids, article.ID)
	}

	// Act
	entries, err := repo.ListEntries(context.Background(), nil, repository.EntryListCriteria{}, 10)

	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, ids[1], entries[0].ID)
	require.Equal(t, ids[0], entries[1].ID)
}

func TestSQLiteCursorReturnsOnlyEntriesAfterItsPosition(t *testing.T) {
	// Arrange
	repo, source := entryRepository(t)
	olderAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newerAt := olderAt.Add(time.Hour)
	older, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "Older", Link: "https://example.test/old", PublishedAt: &olderAt})
	require.NoError(t, err)
	newer, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "Newer", Link: "https://example.test/new", PublishedAt: &newerAt})
	require.NoError(t, err)
	for _, article := range []*entrydomain.Entry{older, newer} {
		_, err = repo.CreateEntry(context.Background(), article)
		require.NoError(t, err)
	}
	cursor := &repository.EntryListCursor{SortAt: newer.SortAt(), ID: newer.ID}

	// Act
	entries, err := repo.ListEntries(context.Background(), cursor, repository.EntryListCriteria{}, 10)

	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, older.ID, entries[0].ID)
}

func TestSQLiteEntrySurvivesClosingAndReopeningTheDatabase(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "entries.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	testkit.Migrate(t, db)
	source, err := sourcedomain.NewRSSSource("News", "https://example.test/rss")
	require.NoError(t, err)
	require.NoError(t, sourcesqlite.NewRepository(db).CreateSource(context.Background(), source))
	article, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "Persisted", Link: "https://example.test/a"})
	require.NoError(t, err)
	savedAt := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	article.MarkSaved(savedAt)
	_, err = NewRepository(db).CreateEntry(context.Background(), article)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	// Act
	db, err = sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	got, err := NewRepository(db).GetEntryByID(context.Background(), article.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, article.ID, got.ID)
	require.Equal(t, "Persisted", got.Title)
	require.NotNil(t, got.SavedAt)
	require.Equal(t, savedAt.UnixNano(), got.SavedAt.UnixNano())
}

func TestSQLiteFindsAnEntryByItsSourceAndLink(t *testing.T) {
	// Arrange
	repo, source := entryRepository(t)
	article, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "News", Link: "https://example.test/a"})
	require.NoError(t, err)
	_, err = repo.CreateEntry(context.Background(), article)
	require.NoError(t, err)

	// Act
	got, err := repo.GetEntryBySourceAndLink(context.Background(), source.ID, article.Link)

	// Assert
	require.NoError(t, err)
	require.Equal(t, article.ID, got.ID)
}

func TestSQLiteReturnsNotFoundForAMissingEntry(t *testing.T) {
	// Arrange
	repo, _ := entryRepository(t)

	// Act
	_, err := repo.GetEntryByID(context.Background(), uuid.New())

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, apperr.KindNotFound, failure.Kind)
	require.Equal(t, "NOT_FOUND", failure.Code)
}
