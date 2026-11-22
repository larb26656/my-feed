package http

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/modules/source/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

type Handler struct {
	svc       *service.Service
	lifecycle sourceapi.Lifecycle
}

func NewHandler(svc *service.Service, lifecycle sourceapi.Lifecycle) *Handler {
	return &Handler{svc: svc, lifecycle: lifecycle}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/", h.createSource)
	r.POST("/preview", h.previewSource)
	r.POST("", h.createSource)
	r.GET("/", h.listSources)
	r.GET("", h.listSources)
	r.GET("/:id", h.getSource)
	r.PATCH("/:id", h.updateSource)
	r.PUT("/:id/profile-image", h.replaceProfileImage)
	r.GET("/:id/profile-image", h.getProfileImage)
	r.DELETE("/:id/profile-image", h.removeProfileImage)
	r.POST("/:id/deactivate", h.deactivateSource)
	r.POST("/:id/reactivate", h.reactivateSource)
}

// @Summary Create Source
// @Tags Sources
// @Produce json
// @Accept json
// @Param request body createSourceRequest true "Request body"
// @Success 201 {object} httpx.SuccessResponse[sourceResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/sources [post]
func (h *Handler) createSource(c *gin.Context) {
	r := c.Request
	var req createSourceRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	s, err := h.lifecycle.CreateSource(r.Context(), sourceapi.CreateSourceInput{
		Name:   req.Name,
		Config: req.Config,
		Type:   req.Type,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, httpx.Success(toLifecycleResponse(s)))
}

// @Summary List Sources
// @Tags Sources
// @Produce json
// @Param page query int false "Page number" minimum(1) default(1)
// @Param pageSize query int false "Page size" minimum(1) maximum(100) default(10)
// @Param nameLike query string false "Case-insensitive substring match on Source name"
// @Param typeEqual query string false "Exact Source Type"
// @Param activeEqual query bool false "Filter by active status; omit to include both"
// @Param sortBy query string false "Sort field" Enums(name, type, active, createdAt) default(createdAt)
// @Param sortDirection query string false "Sort direction" Enums(asc, desc) default(asc)
// @Success 200 {object} PageResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/sources [get]
func (h *Handler) listSources(c *gin.Context) {
	r := c.Request
	req := listSourcesQuery{PageOption: httpx.NewPageOption(), SortBy: "createdAt", SortDirection: "asc"}
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(apperr.InvalidInput.Wrap(err, "Invalid Source query"))
		return
	}
	list, total, err := h.svc.ListSourcesPage(r.Context(), service.ListSourcesPageInput{
		NameLike: req.NameLike, TypeEqual: req.TypeEqual, ActiveEqual: req.ActiveEqual,
		SortBy: req.SortBy, SortDirection: req.SortDirection, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	out := make([]sourceResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toSourceResponse(s))
	}
	c.JSON(http.StatusOK, httpx.Page(out, req.Page, req.PageSize, total))
}

// @Summary Get Source
// @Tags Sources
// @Produce json
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[sourceResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/sources/{id} [get]
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

// @Summary Update Source
// @Description Source type cannot be changed.
// @Tags Sources
// @Produce json
// @Accept json
// @Param request body updateSourceRequest true "Request body"
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[sourceResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/sources/{id} [patch]
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
		Config: req.Config,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toSourceResponse(s)))
}

// @Summary Deactivate Source
// @Tags Sources
// @Produce json
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[sourceResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/sources/{id}/deactivate [post]
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

// @Summary Reactivate Source
// @Tags Sources
// @Produce json
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[sourceResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/sources/{id}/reactivate [post]
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

// @Summary Replace Source profile image
// @Tags Sources
// @Accept application/octet-stream
// @Produce json
// @Param id path string true "Source ID"
// @Param image body []byte true "JPEG or PNG, up to 2 MB"
// @Success 200 {object} httpx.SuccessResponse[sourceResponse]
// @Router /api/v1/sources/{id}/profile-image [put]
func (h *Handler) replaceProfileImage(c *gin.Context) {
	id, err := parseID(c, "invalid source id")
	if err != nil {
		_ = c.Error(err)
		return
	}
	data, err := io.ReadAll(io.LimitReader(c.Request.Body, 2*1024*1024+1))
	if err != nil {
		_ = c.Error(apperr.InvalidInput.Wrap(err, "invalid image"))
		return
	}
	src, err := h.svc.ReplaceProfileImage(c.Request.Context(), id, data)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toSourceResponse(src)))
}

// @Summary Get Source profile image
// @Tags Sources
// @Produce image/jpeg,image/png
// @Param id path string true "Source ID"
// @Success 200 {string} string "Image bytes"
// @Router /api/v1/sources/{id}/profile-image [get]
func (h *Handler) getProfileImage(c *gin.Context) {
	id, err := parseID(c, "invalid source id")
	if err != nil {
		_ = c.Error(err)
		return
	}
	data, err := h.svc.GetProfileImage(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Data(http.StatusOK, http.DetectContentType(data), data)
}

// @Summary Remove Source profile image
// @Tags Sources
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {object} httpx.SuccessResponse[sourceResponse]
// @Router /api/v1/sources/{id}/profile-image [delete]
func (h *Handler) removeProfileImage(c *gin.Context) {
	id, err := parseID(c, "invalid source id")
	if err != nil {
		_ = c.Error(err)
		return
	}
	src, err := h.svc.RemoveProfileImage(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toSourceResponse(src)))
}

// @Summary Test fetch a Source
// @Tags Sources
// @Accept json
// @Produce json
// @Param request body previewSourceRequest true "Source configuration"
// @Success 200 {object} httpx.SuccessResponse[previewResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Router /api/v1/sources/preview [post]
func (h *Handler) previewSource(c *gin.Context) {
	var req previewSourceRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	entry, err := h.svc.Preview(c.Request.Context(), sourceapi.PreviewInput{Type: req.Type, Config: req.Config})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toPreviewResponse(entry)))
}
