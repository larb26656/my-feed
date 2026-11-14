package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	entryhttp "github.com/larb26656/quick-feed/backend/internal/entry/adapter/http"
	entrypg "github.com/larb26656/quick-feed/backend/internal/entry/adapter/postgres"
	entrysvc "github.com/larb26656/quick-feed/backend/internal/entry/service"
	"github.com/larb26656/quick-feed/backend/internal/platform/database"
	sourcehttp "github.com/larb26656/quick-feed/backend/internal/source/adapter/http"
	sourcepg "github.com/larb26656/quick-feed/backend/internal/source/adapter/postgres"
	sourcedomain "github.com/larb26656/quick-feed/backend/internal/source/domain"
	sourcerepo "github.com/larb26656/quick-feed/backend/internal/source/repository"
	sourcesvc "github.com/larb26656/quick-feed/backend/internal/source/service"
)

type App struct {
	Pool   *database.Pool
	Router chi.Router
}

// sourceLookup adapts the Source repository port for Entry without importing
// source/adapter/postgres into the entry module.
type sourceLookup struct {
	repo sourcerepo.Repository
}

func (s sourceLookup) Get(ctx context.Context, id uuid.UUID) (*entrysvc.SourceRef, error) {
	src, err := s.repo.GetSourceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toEntrySourceRef(src), nil
}

func (s sourceLookup) GetManual(ctx context.Context) (*entrysvc.SourceRef, error) {
	src, err := s.repo.GetManualSource(ctx)
	if err != nil {
		return nil, err
	}
	return toEntrySourceRef(src), nil
}

func toEntrySourceRef(src *sourcedomain.Source) *entrysvc.SourceRef {
	return &entrysvc.SourceRef{
		ID:     src.ID,
		Type:   entrysvc.SourceType(src.Type),
		Active: src.Active,
	}
}

func New(ctx context.Context) (*App, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://quickfeed:quickfeed@localhost:54330/quick_feed?sslmode=disable"
	}

	pool, err := database.NewPool(ctx, dsn)
	if err != nil {
		return nil, err
	}

	sourceRepo := sourcepg.NewRepository(pool.Pool)
	sourceService := sourcesvc.New(sourceRepo)
	if _, err := sourceService.EnsureSavedLinks(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ensure saved links source: %w", err)
	}

	entryRepo := entrypg.NewRepository(pool.Pool)
	entryService := entrysvc.New(entryRepo, sourceLookup{repo: sourceRepo})

	r := chi.NewRouter()
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Mount("/api/v1/sources", sourcehttp.NewHandler(sourceService).Routes())
	r.Mount("/api/v1/entries", entryhttp.NewHandler(entryService).Routes())

	return &App{Pool: pool, Router: r}, nil
}

func (a *App) Close() {
	if a.Pool != nil {
		a.Pool.Close()
	}
}

func Run(ctx context.Context) error {
	app, err := New(ctx)
	if err != nil {
		return err
	}
	defer app.Close()

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	fmt.Println("listening on", addr)
	return http.ListenAndServe(addr, app.Router)
}
