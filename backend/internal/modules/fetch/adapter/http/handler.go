package http

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"net/http"
)

type Handler struct{ svc *service.Service }

func NewHandler(svc *service.Service) *Handler { return &Handler{svc: svc} }
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/", h.createManual)
	r.POST("", h.createManual)
	r.GET("/", h.listJobs)
	r.GET("", h.listJobs)
	r.GET("/:id", h.getJob)
}
func (h *Handler) createManual(c *gin.Context) {
	var req createFetchJobRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	job, err := h.svc.CreateManualFetchJob(c.Request.Context(), req.SourceID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, httpx.Success(toFetchJobResponse(job)))
}
func (h *Handler) getJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		_ = c.Error(apperr.InvalidInput.Wrap(err, "invalid fetch job id"))
		return
	}
	job, err := h.svc.GetFetchJob(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toFetchJobResponse(job)))
}
func (h *Handler) listJobs(c *gin.Context) {
	req := listFetchJobsQuery{CursorPageOption: httpx.NewCursorPageOption()}
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(apperr.InvalidInput.Wrap(err, "invalid fetch job query"))
		return
	}
	page, err := h.svc.ListFetchJobs(c.Request.Context(), req.input())
	if err != nil {
		_ = c.Error(err)
		return
	}
	items := make([]fetchJobResponse, 0, len(page.Jobs))
	for _, job := range page.Jobs {
		items = append(items, toFetchJobResponse(job))
	}
	var next string
	if page.NextCursor != nil {
		next = *page.NextCursor
	}
	c.JSON(http.StatusOK, httpx.Cursor(items, req.PageSize, next))
}
