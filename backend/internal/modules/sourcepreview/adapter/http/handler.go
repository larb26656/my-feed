package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	previewapi "github.com/larb26656/my-feed/backend/internal/modules/sourcepreview/api"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

type Handler struct{ preview previewapi.Previewer }

func NewHandler(preview previewapi.Previewer) *Handler { return &Handler{preview: preview} }
func (h *Handler) RegisterRoutes(r *gin.RouterGroup)   { r.POST("/preview", h.previewSource) }

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
	entry, err := h.preview.Preview(c.Request.Context(), previewapi.Input{Type: req.Type, RSSURL: req.RSSURL, YouTubeChannelID: req.YouTubeChannelID})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toPreviewResponse(entry)))
}
