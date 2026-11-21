package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/source/domain"
	"github.com/larb26656/my-feed/backend/internal/source/service"
)

// SourceAPI is the application surface the HTTP adapter needs.
// Kept as an interface so bootstrap can wrap cross-module hooks (e.g. fetch on deactivate).
type SourceAPI interface {
	CreateSource(ctx context.Context, in service.CreateSourceInput) (*domain.Source, error)
	ListSources(ctx context.Context, in service.ListSourcesInput) ([]*domain.Source, error)
	GetSource(ctx context.Context, id uuid.UUID) (*domain.Source, error)
	UpdateSource(ctx context.Context, id uuid.UUID, in service.UpdateSourceInput) (*domain.Source, error)
	DeactivateSource(ctx context.Context, id uuid.UUID) (*domain.Source, error)
	ReactivateSource(ctx context.Context, id uuid.UUID) (*domain.Source, error)
}

type Handler struct {
	svc SourceAPI
}

func NewHandler(svc SourceAPI) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/", h.createSource)
	r.POST("", h.createSource)
	r.GET("/", h.listSources)
	r.GET("", h.listSources)
	r.GET("/:id", h.getSource)
	r.PATCH("/:id", h.updateSource)
	r.POST("/:id/deactivate", h.deactivateSource)
	r.POST("/:id/reactivate", h.reactivateSource)
}

func (h *Handler) createSource(c *gin.Context) {
	r := c.Request
	var req createSourceRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	s, err := h.svc.CreateSource(r.Context(), service.CreateSourceInput{
		Name:             req.Name,
		Type:             req.Type,
		RSSURL:           req.RSSURL,
		YouTubeChannelID: req.YouTubeChannelID,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, httpx.Success(toSourceResponse(s)))
}

func (h *Handler) listSources(c *gin.Context) {
	r := c.Request
	var req listSourcesQuery
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(apperr.InvalidInput.Wrap(err, "Invalid Source query"))
		return
	}
	list, err := h.svc.ListSources(r.Context(), service.ListSourcesInput{
		IncludeDeactivated: req.IncludeDeactivated,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	out := make([]sourceResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toSourceResponse(s))
	}
	c.JSON(http.StatusOK, httpx.Success(out))
}

func (h *Handler) getSource(c *gin.Context) {
	r := c.Request
	id, err := parseID(c, "invalid source id")
	if err != nil {
		_ = c.Error(err)
		return
	}
	s, err := h.svc.GetSource(r.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toSourceResponse(s)))
}

func (h *Handler) updateSource(c *gin.Context) {
	r := c.Request
	id, err := parseID(c, "invalid source id")
	if err != nil {
		_ = c.Error(err)
		return
	}
	var req updateSourceRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	s, err := h.svc.UpdateSource(r.Context(), id, service.UpdateSourceInput{
		Name:   req.Name,
		RSSURL: req.RSSURL,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toSourceResponse(s)))
}

func (h *Handler) deactivateSource(c *gin.Context) {
	r := c.Request
	id, err := parseID(c, "invalid source id")
	if err != nil {
		_ = c.Error(err)
		return
	}
	s, err := h.svc.DeactivateSource(r.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toSourceResponse(s)))
}

func (h *Handler) reactivateSource(c *gin.Context) {
	r := c.Request
	id, err := parseID(c, "invalid source id")
	if err != nil {
		_ = c.Error(err)
		return
	}
	s, err := h.svc.ReactivateSource(r.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toSourceResponse(s)))
}

func parseID(c *gin.Context, invalidMsg string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, apperr.InvalidInput.Wrap(err, invalidMsg)
	}
	return id, nil
}
