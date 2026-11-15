package bootstrap

import (
	"testing"
	"time"

	"github.com/larb26656/quick-feed/backend/internal/fetch/adapter/runtime"
	fetchsvc "github.com/larb26656/quick-feed/backend/internal/fetch/service"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("FETCH_RETRY_BACKOFF", "")
	t.Setenv("FETCH_SCHEDULE_INTERVAL", "")
	t.Setenv("FETCH_EXECUTOR_POLL", "")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, defaultDatabasePath, cfg.DatabasePath)
	require.Equal(t, defaultHTTPAddr, cfg.HTTPAddr)
	require.Equal(t, fetchsvc.DefaultRetryBackoff, cfg.Fetch.RetryBackoff)
	require.Equal(t, fetchsvc.DefaultScheduleInterval, cfg.Fetch.ScheduleInterval)
	require.Equal(t, runtime.DefaultExecutorPoll, cfg.Fetch.ExecutorPoll)
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("DATABASE_PATH", "var/library.db")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("FETCH_RETRY_BACKOFF", "30s")
	t.Setenv("FETCH_SCHEDULE_INTERVAL", "5m")
	t.Setenv("FETCH_EXECUTOR_POLL", "1s")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, "var/library.db", cfg.DatabasePath)
	require.Equal(t, ":9090", cfg.HTTPAddr)
	require.Equal(t, 30*time.Second, cfg.Fetch.RetryBackoff)
	require.Equal(t, 5*time.Minute, cfg.Fetch.ScheduleInterval)
	require.Equal(t, time.Second, cfg.Fetch.ExecutorPoll)
}

func TestLoadConfigInvalidDurationFallsBack(t *testing.T) {
	t.Setenv("FETCH_RETRY_BACKOFF", "not-a-duration")
	t.Setenv("FETCH_SCHEDULE_INTERVAL", "0s")
	t.Setenv("FETCH_EXECUTOR_POLL", "-1s")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, fetchsvc.DefaultRetryBackoff, cfg.Fetch.RetryBackoff)
	require.Equal(t, fetchsvc.DefaultScheduleInterval, cfg.Fetch.ScheduleInterval)
	require.Equal(t, runtime.DefaultExecutorPoll, cfg.Fetch.ExecutorPoll)
}
