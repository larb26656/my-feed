package bootstrap

import (
	"fmt"
	"os"
	"time"

	"github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/runtime"
	fetchsvc "github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
)

const (
	defaultDatabasePath = "data/my-feed.db"
	defaultHTTPAddr     = ":8080"
	defaultAccessTTL    = 15 * time.Minute
	defaultRefreshTTL   = 7 * 24 * time.Hour
)

// Config holds process settings loaded from the environment.
type Config struct {
	DatabasePath string
	HTTPAddr     string
	WebRoot      string
	// APIKey is an optional shared Bearer secret for /api/v1 and /mcp.
	// Empty disables the API-key auth path; JWT login remains required.
	APIKey string
	Auth   AuthConfig
	Fetch  FetchConfig
}

// AuthConfig holds Reader login and JWT Session settings.
type AuthConfig struct {
	Username   string
	Password   string
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// FetchConfig holds in-process Fetch Job scheduler/executor settings.
type FetchConfig struct {
	RetryBackoff     time.Duration
	ScheduleInterval time.Duration
	ExecutorPoll     time.Duration
}

// LoadConfig reads settings from the environment. Username, password, and
// JWT secret are required and must be non-empty.
func LoadConfig() (Config, error) {
	username := os.Getenv("MY_FEED_USERNAME")
	password := os.Getenv("MY_FEED_PASSWORD")
	jwtSecret := os.Getenv("MY_FEED_JWT_SECRET")
	if username == "" || password == "" || jwtSecret == "" {
		return Config{}, fmt.Errorf("MY_FEED_USERNAME, MY_FEED_PASSWORD, and MY_FEED_JWT_SECRET are required")
	}

	return Config{
		DatabasePath: envString("DATABASE_PATH", defaultDatabasePath),
		HTTPAddr:     envString("HTTP_ADDR", defaultHTTPAddr),
		WebRoot:      os.Getenv("WEB_ROOT"),
		APIKey:       os.Getenv("MY_FEED_API_KEY"),
		Auth: AuthConfig{
			Username:   username,
			Password:   password,
			JWTSecret:  jwtSecret,
			AccessTTL:  envDuration("MY_FEED_ACCESS_TTL", defaultAccessTTL),
			RefreshTTL: envDuration("MY_FEED_REFRESH_TTL", defaultRefreshTTL),
		},
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
