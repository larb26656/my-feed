package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	feedhttp "github.com/larb26656/quick-feed/backend/internal/feed/adapter/http"
	feedpg "github.com/larb26656/quick-feed/backend/internal/feed/adapter/postgres"
	feedsvc "github.com/larb26656/quick-feed/backend/internal/feed/service"
	"github.com/larb26656/quick-feed/backend/internal/platform/database"
)

type App struct {
	Pool   *database.Pool
	Router chi.Router
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

	feedRepo := feedpg.NewRepository(pool.Pool)
	feedService := feedsvc.New(feedRepo)

	r := chi.NewRouter()
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	h := feedhttp.NewHandler(feedService)
	r.Mount("/api/v1/sources", h.SourceRoutes())
	r.Mount("/api/v1/entries", h.EntryRoutes())

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
