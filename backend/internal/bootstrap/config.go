package bootstrap

import (
	"os"
	"time"

	"github.com/larb26656/my-feed/backend/internal/fetch/adapter/runtime"
	fetchsvc "github.com/larb26656/my-feed/backend/internal/fetch/service"
)

const (
	defaultDatabasePath = "data/my-feed.db"
	defaultHTTPAddr     = ":8080"
)

// Config holds process settings loaded from the environment.
type Config struct {
	DatabasePath string
	HTTPAddr     string
	Fetch        FetchConfig
}

// FetchConfig holds in-process Fetch Job scheduler/executor settings.
type FetchConfig struct {
	RetryBackoff     time.Duration
	ScheduleInterval time.Duration
	ExecutorPoll     time.Duration
}

// LoadConfig reads settings from the environment, applying defaults when a
// value is missing or invalid. All fields currently have defaults.
func LoadConfig() (Config, error) {
	return Config{
		DatabasePath: envString("DATABASE_PATH", defaultDatabasePath),
		HTTPAddr:     envString("HTTP_ADDR", defaultHTTPAddr),
		Fetch: FetchConfig{
			RetryBackoff:     envDuration("FETCH_RETRY_BACKOFF", fetchsvc.DefaultRetryBackoff),
			ScheduleInterval: envDuration("FETCH_SCHEDULE_INTERVAL", fetchsvc.DefaultScheduleInterval),
			ExecutorPoll:     envDuration("FETCH_EXECUTOR_POLL", runtime.DefaultExecutorPoll),
		},
	}, nil
}

func envString(key, fallback string) string {
	if raw := os.Getenv(key); raw != "" {
		return raw
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
