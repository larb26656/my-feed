package database_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/platform/database"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestGlobalEntryLinkMigrationPreservesWinnerAndState(t *testing.T) {
	ctx := context.Background()
	db, err := database.NewPool(ctx, filepath.Join(t.TempDir(), "library.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db.DB, os.DirFS("../../../migrations"))
	require.NoError(t, err)
	_, err = provider.UpTo(ctx, 6)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO sources(id,name,type,config,key,active,created_at,updated_at) VALUES
 ('a','A','rss','{}','a',1,1,1),('b','B','rss','{}','b',1,1,1),('c','C','rss','{}','c',1,1,1);
 INSERT INTO entries(id,source_id,title,link,created_at,updated_at,read_at,saved_at) VALUES
 ('unsaved','a','Older',' link ',1,1,2,NULL),('saved-z','b','Z','link',3,3,5,7),('saved-a','c','Winner','link',3,3,4,8),
 ('no-saved-z','a','Z',' other ',4,4,NULL,NULL),('no-saved-a','b','A','other',4,4,NULL,NULL);`)
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	var source, title string
	var read, saved int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT source_id,title,read_at,saved_at FROM entries WHERE id='saved-a' AND link='link'`).Scan(&source, &title, &read, &saved))
	require.Equal(t, "c", source)
	require.Equal(t, "Winner", title)
	require.Equal(t, 2, read)
	require.Equal(t, 7, saved)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM entries`).Scan(&count))
	require.Equal(t, 2, count)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM entries WHERE id='no-saved-a' AND source_id='b'`).Scan(&count))
	require.Equal(t, 1, count)
	_, err = db.ExecContext(ctx, `INSERT INTO entries(id,source_id,title,link,created_at,updated_at) VALUES('dup','a','Dup','link',1,1)`)
	require.Error(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO entries(id,source_id,title,link,created_at,updated_at) VALUES('n1','a','Note',NULL,1,1),('n2','a','Note',NULL,1,1)`)
	require.NoError(t, err)
	rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	require.NoError(t, err)
	defer func() { require.NoError(t, rows.Close()) }()
	require.False(t, rows.Next())
}
