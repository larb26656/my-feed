package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

type typesResponse struct {
	Types []api.Definition `json:"types"`
}

type Handler struct{ catalog api.Catalog }

func RegisterRoutes(r *gin.RouterGroup, catalog api.Catalog) {
	h := &Handler{catalog: catalog}
	r.GET("/source-types", h.list)
}

// @Summary List Source Types
// @Tags Sources
// @Produce json
// @Success 200 {object} httpx.SuccessResponse[typesResponse]
// @Router /api/v1/source-types [get]
func (h *Handler) list(c *gin.Context) {
	c.JSON(http.StatusOK, httpx.Success(typesResponse{Types: h.catalog.Definitions()}))
}
