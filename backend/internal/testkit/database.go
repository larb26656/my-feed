// Package testkit contains small fixtures shared by backend integration tests.
package testkit

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/platform/database"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

// NewDatabase opens an isolated SQLite file with production connection settings
// and all real migrations. It owns cleanup; callers may safely run in parallel.
func NewDatabase(t testing.TB) *database.Pool {
	t.Helper()
	pool, err := database.NewPool(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	Migrate(t, pool.DB)
	return pool
}

// Migrate applies every production migration to an already-open database.
func Migrate(t testing.TB, db *sql.DB) {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS(filepath.Join(filepath.Dir(file), "../../migrations")))
	require.NoError(t, err)
	_, err = provider.Up(context.Background())
	require.NoError(t, err)
}
