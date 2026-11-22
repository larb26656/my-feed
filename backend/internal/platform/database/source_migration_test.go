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

func TestSourceConfigMigrationPreservesLibrary(t *testing.T) {
	ctx := context.Background()
	db, err := database.NewPool(ctx, filepath.Join(t.TempDir(), "library.db"))
	require.NoError(t, err)
	defer func() { require.NoError(t, db.Close()) }()
	provider, err := goose.NewProvider(goose.DialectSQLite3, db.DB, os.DirFS("../../../migrations"))
	require.NoError(t, err)
	_, err = provider.UpTo(ctx, 5)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO sources(id,name,type,rss_url,active,created_at,updated_at,profile_image_id) VALUES('r','RSS','rss',' legacy url ',0,10,20,'image');
 INSERT INTO sources(id,name,type,youtube_channel_id,active,created_at,updated_at) VALUES('y','Video','youtube','UCabcdefghijklmnopqrstuv',1,11,21);
 INSERT INTO sources(id,name,type,active,created_at,updated_at) VALUES('m','Saved links','manual',1,12,22);
 INSERT INTO entries(id,source_id,title,link,created_at,updated_at,read_at,saved_at) VALUES('e','r','Title','link',1,2,3,4);
 INSERT INTO fetch_jobs(id,source_id,status) VALUES('j','r','succeeded');`)
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)
	var config, key, image string
	var active, created, updated int
	err = db.QueryRowContext(ctx, `SELECT config,key,profile_image_id,active,created_at,updated_at FROM sources WHERE id='r'`).Scan(&config, &key, &image, &active, &created, &updated)
	require.NoError(t, err)
	require.JSONEq(t, `{"url":" legacy url "}`, config)
	require.Equal(t, " legacy url ", key)
	require.Equal(t, "image", image)
	require.Equal(t, 0, active)
	require.Equal(t, 10, created)
	require.Equal(t, 20, updated)
	var youtubeConfig, youtubeKey, manualConfig, manualKey string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT config,key FROM sources WHERE id='y'`).Scan(&youtubeConfig, &youtubeKey))
	require.JSONEq(t, `{"channelId":"UCabcdefghijklmnopqrstuv"}`, youtubeConfig)
	require.Equal(t, "UCabcdefghijklmnopqrstuv", youtubeKey)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT config,key FROM sources WHERE id='m'`).Scan(&manualConfig, &manualKey))
	require.JSONEq(t, `{}`, manualConfig)
	require.Equal(t, "singleton", manualKey)
	_, err = db.ExecContext(ctx, `INSERT INTO sources(id,name,type,config,key,active,created_at,updated_at) VALUES('another-manual','Saved','manual','{}','different',1,1,1)`)
	require.Error(t, err)
	var read, saved int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT read_at,saved_at FROM entries WHERE source_id='r'`).Scan(&read, &saved))
	require.Equal(t, 3, read)
	require.Equal(t, 4, saved)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM fetch_jobs WHERE source_id='r'`).Scan(&count))
	require.Equal(t, 1, count)
	_, err = db.ExecContext(ctx, `INSERT INTO sources(id,name,type,config,key,active,created_at,updated_at) VALUES('duplicate','Dup','rss','{}',' legacy url ',1,1,1)`)
	require.Error(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO sources(id,name,type,config,key,active,created_at,updated_at) VALUES('new','New','custom','{}',' legacy url ',1,1,1)`)
	require.NoError(t, err)
}
