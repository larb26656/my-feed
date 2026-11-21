package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPoolPersistsDataAndEnablesForeignKeys(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "nested", "my-feed.db")

	db, err := NewPool(ctx, path)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (parent_id INTEGER REFERENCES parent(id))`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO child(parent_id) VALUES (1)`)
	require.Error(t, err, "foreign keys must be enforced by the production DSN")
	_, err = db.ExecContext(ctx, `INSERT INTO parent(id) VALUES (1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO child(parent_id) VALUES (1)`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	db, err = NewPool(ctx, path)
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM child`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestWithinTransactionCommitsAndRollsBack(t *testing.T) {
	ctx := context.Background()
	db, err := NewPool(ctx, filepath.Join(t.TempDir(), "my-feed.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	_, err = db.ExecContext(ctx, `CREATE TABLE values_table (value INTEGER)`)
	require.NoError(t, err)

	require.NoError(t, db.WithinTransaction(ctx, func(ctx context.Context, raw Transaction) error {
		tx := raw.(*sql.Tx)
		_, err := tx.ExecContext(ctx, `INSERT INTO values_table VALUES (1)`)
		return err
	}))

	rollback := db.WithinTransaction(ctx, func(ctx context.Context, raw Transaction) error {
		tx := raw.(*sql.Tx)
		if _, err := tx.ExecContext(ctx, `INSERT INTO values_table VALUES (2)`); err != nil {
			return err
		}
		return context.Canceled
	})
	require.ErrorIs(t, rollback, context.Canceled)

	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM values_table`).Scan(&count))
	require.Equal(t, 1, count)
}
