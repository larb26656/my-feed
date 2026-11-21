package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/source/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

// SourceAPI exposes operations that only need the Source service.
type SourceAPI interface {
	ListSources(ctx context.Context, in service.ListSourcesInput) ([]*domain.Source, error)
	GetSource(ctx context.Context, id uuid.UUID) (*domain.Source, error)
	UpdateSource(ctx context.Context, id uuid.UUID, in service.UpdateSourceInput) (*domain.Source, error)
	ReactivateSource(ctx context.Context, id uuid.UUID) (*domain.Source, error)
}

type Handler struct {
	svc       SourceAPI
	lifecycle sourceapi.Lifecycle
}

func NewHandler(svc SourceAPI, lifecycle sourceapi.Lifecycle) *Handler {
	return &Handler{svc: svc, lifecycle: lifecycle}
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
	s, err := h.lifecycle.CreateSource(r.Context(), sourceapi.CreateSourceInput{
		Name:             req.Name,
		Type:             req.Type,
		RSSURL:           req.RSSURL,
		YouTubeChannelID: req.YouTubeChannelID,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, httpx.Success(toLifecycleResponse(s)))
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
	s, err := h.lifecycle.DeactivateSource(r.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toLifecycleResponse(s)))
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
