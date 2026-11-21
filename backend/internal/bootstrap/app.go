package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	prototypehttp "github.com/larb26656/my-feed/backend/internal/apiprototype/adapter/http"
	"github.com/larb26656/my-feed/backend/internal/apiprototype/adapter/memory"
	prototypesvc "github.com/larb26656/my-feed/backend/internal/apiprototype/service"
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
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	sourcehttp "github.com/larb26656/my-feed/backend/internal/source/adapter/http"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/source/adapter/sqlite"
	sourceapi "github.com/larb26656/my-feed/backend/internal/source/api"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/source/service"
)

type App struct {
	Pool   *database.Pool
	Router *gin.Engine
}

var configurePrototypeValidator sync.Once

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
		sourceService,
		obtainRegistry,
		entrysvc.NewIngester(entryService),
		realClock{},
		cfg.Fetch.RetryBackoff,
		cfg.Fetch.ScheduleInterval,
	)

	runtime.NewLoops(
		fetchService,
		cfg.Fetch.ExecutorPoll,
	).Start(ctx)

	configurePrototypeValidator.Do(func() {
		if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
			v.RegisterTagNameFunc(func(field reflect.StructField) string {
				name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
				if name == "-" {
					return ""
				}
				return name
			})
		}
	})

	r := httpx.NewRouter()

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	sourcehttp.NewHandler(sourceAPIWithFetch{
		db:         pool,
		sourceRepo: sourceRepo,
		fetchRepo:  fetchRepo,
		inner:      sourceService,
		fetch:      fetchService,
	}).RegisterRoutes(r.Group("/api/v1/sources"))
	entryhttp.NewHandler(entryService).RegisterRoutes(r.Group("/api/v1/entries"))
	fetchhttp.NewHandler(fetchService).RegisterRoutes(r.Group("/api/v1/fetch-jobs"))
	prototypehttp.New(prototypesvc.New(memory.New())).RegisterRoutes(r.Group("/api/prototype"))

	return &App{Pool: pool, Router: r}, nil
}

func newObtainRegistry(client *http.Client) *obtain.Registry {
	registry := obtain.NewRegistry()
	registry.Register(sourceapi.SourceTypeRSS, obtainrss.New(client))
	registry.Register(sourceapi.SourceTypeYouTube, obtainyoutube.New(client))
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
