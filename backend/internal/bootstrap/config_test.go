package bootstrap

import (
	"testing"
	"time"

	"github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/runtime"
	fetchsvc "github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	"github.com/stretchr/testify/require"
)

func setRequiredAuthEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MY_FEED_USERNAME", "admin")
	t.Setenv("MY_FEED_PASSWORD", "secret")
	t.Setenv("MY_FEED_JWT_SECRET", "jwt-test-secret")
}

func TestLoadConfigDefaults(t *testing.T) {
	setRequiredAuthEnv(t)
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("FETCH_RETRY_BACKOFF", "")
	t.Setenv("FETCH_SCHEDULE_INTERVAL", "")
	t.Setenv("FETCH_EXECUTOR_POLL", "")
	t.Setenv("MY_FEED_ACCESS_TTL", "")
	t.Setenv("MY_FEED_REFRESH_TTL", "")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, defaultDatabasePath, cfg.DatabasePath)
	require.Equal(t, defaultHTTPAddr, cfg.HTTPAddr)
	require.Equal(t, "admin", cfg.Auth.Username)
	require.Equal(t, "secret", cfg.Auth.Password)
	require.Equal(t, "jwt-test-secret", cfg.Auth.JWTSecret)
	require.Equal(t, defaultAccessTTL, cfg.Auth.AccessTTL)
	require.Equal(t, defaultRefreshTTL, cfg.Auth.RefreshTTL)
	require.Equal(t, fetchsvc.DefaultRetryBackoff, cfg.Fetch.RetryBackoff)
	require.Equal(t, fetchsvc.DefaultScheduleInterval, cfg.Fetch.ScheduleInterval)
	require.Equal(t, runtime.DefaultExecutorPoll, cfg.Fetch.ExecutorPoll)
}

func TestLoadConfigFromEnv(t *testing.T) {
	setRequiredAuthEnv(t)
	t.Setenv("DATABASE_PATH", "var/library.db")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("MY_FEED_API_KEY", "dev-secret")
	t.Setenv("MY_FEED_USERNAME", "reader")
	t.Setenv("MY_FEED_PASSWORD", "pass")
	t.Setenv("MY_FEED_JWT_SECRET", "super-secret")
	t.Setenv("MY_FEED_ACCESS_TTL", "5m")
	t.Setenv("MY_FEED_REFRESH_TTL", "48h")
	t.Setenv("FETCH_RETRY_BACKOFF", "30s")
	t.Setenv("FETCH_SCHEDULE_INTERVAL", "5m")
	t.Setenv("FETCH_EXECUTOR_POLL", "1s")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, "var/library.db", cfg.DatabasePath)
	require.Equal(t, ":9090", cfg.HTTPAddr)
	require.Equal(t, "dev-secret", cfg.APIKey)
	require.Equal(t, "reader", cfg.Auth.Username)
	require.Equal(t, "pass", cfg.Auth.Password)
	require.Equal(t, "super-secret", cfg.Auth.JWTSecret)
	require.Equal(t, 5*time.Minute, cfg.Auth.AccessTTL)
	require.Equal(t, 48*time.Hour, cfg.Auth.RefreshTTL)
	require.Equal(t, 30*time.Second, cfg.Fetch.RetryBackoff)
	require.Equal(t, 5*time.Minute, cfg.Fetch.ScheduleInterval)
	require.Equal(t, time.Second, cfg.Fetch.ExecutorPoll)
}

func TestLoadConfigAPIKeyEmptyWhenUnset(t *testing.T) {
	setRequiredAuthEnv(t)
	t.Setenv("MY_FEED_API_KEY", "")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Empty(t, cfg.APIKey)
}

func TestLoadConfigRequiresAuthCredentials(t *testing.T) {
	t.Setenv("MY_FEED_USERNAME", "")
	t.Setenv("MY_FEED_PASSWORD", "secret")
	t.Setenv("MY_FEED_JWT_SECRET", "jwt")

	_, err := LoadConfig()
	require.Error(t, err)

	t.Setenv("MY_FEED_USERNAME", "admin")
	t.Setenv("MY_FEED_PASSWORD", "")
	_, err = LoadConfig()
	require.Error(t, err)

	t.Setenv("MY_FEED_PASSWORD", "secret")
	t.Setenv("MY_FEED_JWT_SECRET", "")
	_, err = LoadConfig()
	require.Error(t, err)
}

func TestLoadConfigInvalidDurationFallsBack(t *testing.T) {
	setRequiredAuthEnv(t)
	t.Setenv("FETCH_RETRY_BACKOFF", "not-a-duration")
	t.Setenv("FETCH_SCHEDULE_INTERVAL", "0s")
	t.Setenv("FETCH_EXECUTOR_POLL", "-1s")
	t.Setenv("MY_FEED_ACCESS_TTL", "nope")
	t.Setenv("MY_FEED_REFRESH_TTL", "0s")

	cfg, err := LoadConfig()
	require.NoError(t, err)
	require.Equal(t, fetchsvc.DefaultRetryBackoff, cfg.Fetch.RetryBackoff)
	require.Equal(t, fetchsvc.DefaultScheduleInterval, cfg.Fetch.ScheduleInterval)
	require.Equal(t, runtime.DefaultExecutorPoll, cfg.Fetch.ExecutorPoll)
	require.Equal(t, defaultAccessTTL, cfg.Auth.AccessTTL)
	require.Equal(t, defaultRefreshTTL, cfg.Auth.RefreshTTL)
}
