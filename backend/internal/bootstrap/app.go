package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	entryhttp "github.com/larb26656/my-feed/backend/internal/entry/adapter/http"
	entrysqlite "github.com/larb26656/my-feed/backend/internal/entry/adapter/sqlite"
	entrysvc "github.com/larb26656/my-feed/backend/internal/entry/service"
	fetchhttp "github.com/larb26656/my-feed/backend/internal/fetch/adapter/http"
	"github.com/larb26656/my-feed/backend/internal/fetch/adapter/obtain"
	obtainrss "github.com/larb26656/my-feed/backend/internal/fetch/adapter/obtain/rss"
	obtainyoutube "github.com/larb26656/my-feed/backend/internal/fetch/adapter/obtain/youtube"
	"github.com/larb26656/my-feed/backend/internal/fetch/adapter/runtime"
	fetchsqlite "github.com/larb26656/my-feed/backend/internal/fetch/adapter/sqlite"
	fetchsvc "github.com/larb26656/my-feed/backend/internal/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
	sourcehttp "github.com/larb26656/my-feed/backend/internal/source/adapter/http"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/source/adapter/sqlite"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/source/service"
)

type App struct {
	Pool   *database.Pool
	Router chi.Router
}

func New(ctx context.Context) (*App, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	return NewWithConfig(ctx, cfg)
}

func NewWithConfig(ctx context.Context, cfg Config) (*App, error) {
	pool, err := database.NewPool(ctx, cfg.DatabasePath)
	if err != nil {
		return nil, err
	}

	sourceRepo := sourcesqlite.NewRepository(pool.DB)
	sourceService := sourcesvc.New(sourceRepo)
	if _, err := sourceService.EnsureSavedLinks(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ensure saved links source: %w", err)
	}

	entryRepo := entrysqlite.NewRepository(pool.DB)
	entryService := entrysvc.New(entryRepo, sourceLookup{repo: sourceRepo})

	obtainRegistry := newObtainRegistry(http.DefaultClient)

	fetchRepo := fetchsqlite.NewRepository(pool.DB)
	fetchService := fetchsvc.New(
		fetchRepo,
		fetchSources{repo: sourceRepo},
		obtainRegistry,
		entryIngest{svc: entryService},
		realClock{},
		cfg.Fetch.RetryBackoff,
		cfg.Fetch.ScheduleInterval,
	)

	runtime.NewLoops(
		fetchService,
		cfg.Fetch.ExecutorPoll,
	).Start(ctx)

	r := chi.NewRouter()
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Mount("/api/v1/sources", sourcehttp.NewHandler(sourceAPIWithFetch{
		db:         pool,
		sourceRepo: sourceRepo,
		fetchRepo:  fetchRepo,
		inner:      sourceService,
		fetch:      fetchService,
	}).Routes())
	r.Mount("/api/v1/entries", entryhttp.NewHandler(entryService).Routes())
	r.Mount("/api/v1/fetch-jobs", fetchhttp.NewHandler(fetchService).Routes())

	return &App{Pool: pool, Router: r}, nil
}

func newObtainRegistry(client *http.Client) *obtain.Registry {
	registry := obtain.NewRegistry()
	registry.Register(fetchsvc.SourceTypeRSS, obtainrss.New(client))
	registry.Register(fetchsvc.SourceTypeYouTube, obtainyoutube.New(client))
	return registry
}

func (a *App) Close() {
	if a.Pool != nil {
		a.Pool.Close()
	}
}

func Run(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	app, err := NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer app.Close()

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: app.Router}
	errCh := make(chan error, 1)
	go func() {
		fmt.Println("listening on", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		err := <-errCh
		if err == nil || err == http.ErrServerClosed {
			return nil
		}
		return err
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
