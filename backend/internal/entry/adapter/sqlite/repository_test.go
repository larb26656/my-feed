package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	entrydomain "github.com/larb26656/my-feed/backend/internal/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/entry/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	sourcerepo "github.com/larb26656/my-feed/backend/internal/source/adapter/sqlite"
	sourcedomain "github.com/larb26656/my-feed/backend/internal/source/domain"
)

func TestEntryRepositoryPersistsMetadataAndPaginatesInStableOrder(t *testing.T) {
	// Arrange
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "entries.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	testkit.Migrate(t, db)
	sources := sourcerepo.NewRepository(db)
	source, err := sourcedomain.NewRSSSource("Example", "https://example.test/feed")
	require.NoError(t, err)
	require.NoError(t, sources.CreateSource(ctx, source))
	repo := NewRepository(db)

	// Act: insert an entry whose source does not exist.
	_, err = db.Exec(`INSERT INTO entries (id, source_id, title, link, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, uuid.NewString(), uuid.NewString(), "Orphan", "https://example.test/orphan", time.Now().UnixNano(), time.Now().UnixNano())

	// Assert
	require.Error(t, err, "SQLite must enforce the Source foreign key")

	// Arrange: an article with a precise publication timestamp.
	sortTime := time.Date(2025, 4, 3, 2, 1, 0, 123456789, time.UTC)
	e1, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "First", Link: "https://example.test/1", Description: "original", PublishedAt: &sortTime})
	require.NoError(t, err)

	// Act: persist entry metadata.
	created, err := repo.CreateEntry(ctx, e1)

	// Assert
	require.NoError(t, err)
	require.Equal(t, sortTime.UnixNano(), created.PublishedAt.UnixNano())

	// Arrange: another entry with the same source/link identity.
	duplicate, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "Duplicate", Link: e1.Link})
	require.NoError(t, err)

	// Act: persist the same source/link identity again.
	_, err = repo.CreateEntry(ctx, duplicate)

	// Assert
	require.Error(t, err)
	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeConflict, appErr.Code)

	// Arrange: refreshed metadata and independent reader timestamps.
	refreshedAt := sortTime.Add(time.Hour)
	require.NoError(t, created.RefreshMetadata(entrydomain.EntryMetadata{Title: "Refreshed", Link: e1.Link, Description: "new", PublishedAt: &refreshedAt}))
	readAt := refreshedAt.Add(time.Minute)
	created.MarkRead(readAt)
	savedAt := refreshedAt.Add(2 * time.Minute)
	created.MarkSaved(savedAt)

	// Act: persist refreshed metadata and reader choices.
	updated, err := repo.UpdateEntry(ctx, created)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "Refreshed", updated.Title)
	require.Equal(t, readAt.UnixNano(), updated.ReadAt.UnixNano())
	require.Equal(t, savedAt.UnixNano(), updated.SavedAt.UnixNano())

	// Arrange: a second, newer article for pagination.
	newerAt := refreshedAt.Add(time.Hour)
	e2, err := entrydomain.NewEntry(source.ID, entrydomain.EntryMetadata{Title: "Second", Link: "https://example.test/2", PublishedAt: &newerAt})
	require.NoError(t, err)
	_, err = repo.CreateEntry(ctx, e2)
	require.NoError(t, err)

	// Act: read the first page.
	firstPage, err := repo.ListEntries(ctx, nil, repository.EntryListCriteria{}, 1)

	// Assert
	require.NoError(t, err)
	require.Len(t, firstPage, 1)
	require.Equal(t, e2.ID, firstPage[0].ID)

	// Act: continue from the first page cursor.
	secondPage, err := repo.ListEntries(ctx, &repository.EntryListCursor{SortAt: e2.SortAt(), ID: e2.ID}, repository.EntryListCriteria{}, 10)

	// Assert
	require.NoError(t, err)
	require.Len(t, secondPage, 1)
	require.Equal(t, e1.ID, secondPage[0].ID)

	// Arrange: reopen the same persisted database.
	require.NoError(t, db.Close())
	db, err = sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	defer db.Close()

	// Act: retrieve the entry after reopening.
	got, err := NewRepository(db).GetEntryByID(ctx, e1.ID)

	// Assert
	require.NoError(t, err)
	require.Equal(t, "Refreshed", got.Title)
	require.Equal(t, savedAt.UnixNano(), got.SavedAt.UnixNano())

	// Act: retrieve the same entry by its source/link identity.
	_, err = NewRepository(db).GetEntryBySourceAndLink(ctx, source.ID, e1.Link)

	// Assert
	require.NoError(t, err)

	// Act: look up a missing entry.
	_, err = NewRepository(db).GetEntryByID(ctx, uuid.New())

	// Assert
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperr.CodeNotFound, appErr.Code)
}
