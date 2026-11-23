package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	authhttp "github.com/larb26656/my-feed/backend/internal/modules/auth/adapter/http"
	authsqlite "github.com/larb26656/my-feed/backend/internal/modules/auth/adapter/sqlite"
	authsvc "github.com/larb26656/my-feed/backend/internal/modules/auth/service"
	entryhttp "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/http"
	entrymcp "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/mcp"
	entrysqlite "github.com/larb26656/my-feed/backend/internal/modules/entry/adapter/sqlite"
	entrysvc "github.com/larb26656/my-feed/backend/internal/modules/entry/service"
	fetchhttp "github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/http"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/runtime"
	fetchsqlite "github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/sqlite"
	fetchsvc "github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	filestorage "github.com/larb26656/my-feed/backend/internal/modules/file/service"
	sourcehttp "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/http"
	sourcesqlite "github.com/larb26656/my-feed/backend/internal/modules/source/adapter/sqlite"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	sourcelifecycle "github.com/larb26656/my-feed/backend/internal/modules/sourcelifecycle/service"
	sourcedevto "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/devto"
	typehttp "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/http"
	sourcemedium "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/medium"
	sourcerss "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"
	sourceyoutube "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/youtube"
	typesvc "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apidocs"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/platform/http/middleware"
	"github.com/larb26656/my-feed/backend/internal/platform/outbound"
)

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

type App struct {
	Pool   *database.Pool
	Router *gin.Engine
}

var configureJSONValidator sync.Once

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
	sourceService.SetFileStorage(filestorage.New(cfg.UploadsDir))

	client := outbound.NewClient(http.DefaultClient)
	typeRegistry, err := typesvc.New(sourcerss.Definition(client), sourceyoutube.Definition(client), sourcedevto.Definition(client), sourcemedium.Definition(client))
	if err != nil {
		return nil, errors.Join(fmt.Errorf("register source types: %w", err), pool.Close())
	}
	sourceService.SetSourceTypes(typeRegistry)

	// init default source
	if _, err := sourceService.EnsureSavedLinks(ctx); err != nil {
		return nil, errors.Join(fmt.Errorf("ensure saved links source: %w", err), pool.Close())
	}

	entryRepo := entrysqlite.NewRepository(pool.DB)
	entryService := entrysvc.New(entryRepo, sourceService)

	obtainRegistry := typeRegistry
	sourceService.SetObtainer(obtainRegistry)

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

	configureJSONValidator.Do(func() {
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

	authService := authsvc.New(authsqlite.NewRepository(pool.DB), pool, authsvc.Config{
		Username:   cfg.Auth.Username,
		Password:   cfg.Auth.Password,
		JWTSecret:  cfg.Auth.JWTSecret,
		AccessTTL:  cfg.Auth.AccessTTL,
		RefreshTTL: cfg.Auth.RefreshTTL,
	})

	r := httpx.NewRouter()
	httpx.ServeWeb(r, cfg.WebRoot)
	apidocs.Register(r)

	r.GET("/healthz", health)

	api := r.Group("/api/v1")
	authhttp.NewHandler(authService).RegisterRoutes(api.Group("/auth"))

	protected := api.Group("")
	protected.Use(middleware.Authenticate(cfg.Auth.JWTSecret, cfg.APIKey))

	sourceLifecycle := sourcelifecycle.New(pool, sourceService, fetchService)
	sourceHandler := sourcehttp.NewHandler(sourceService, sourceLifecycle)
	sourceHandler.RegisterRoutes(protected.Group("/sources"))
	typehttp.RegisterRoutes(protected, typeRegistry)
	entryhttp.NewHandler(entryService).RegisterRoutes(protected.Group("/entries"))
	fetchhttp.NewHandler(fetchService).RegisterRoutes(protected.Group("/fetch-jobs"))

	mcpHandler := entrymcp.NewHandler(entryService)
	r.Any("/mcp", middleware.Authenticate(cfg.Auth.JWTSecret, cfg.APIKey), gin.WrapH(mcpHandler))

	return &App{Pool: pool, Router: r}, nil
}

func (a *App) Close() {
	if a.Pool != nil {
		if err := a.Pool.Close(); err != nil {
			slog.Error("close database pool", "error", err)
		}
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

// @Summary Health check
// @Tags Health
// @Produce json
// @Success 200 {object} healthResponse
// @Router /healthz [get]
func health(c *gin.Context) { c.JSON(http.StatusOK, healthResponse{Status: "ok"}) }

type healthResponse struct {
	Status string `json:"status"`
}
