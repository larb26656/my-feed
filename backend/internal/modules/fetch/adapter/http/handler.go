package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
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

// @Summary Create Fetch Job
// @Tags Fetch Jobs
// @Produce json
// @Accept json
// @Param request body createFetchJobRequest true "Request body"
// @Success 201 {object} httpx.SuccessResponse[fetchJobResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/fetch-jobs [post]
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

// @Summary Get Fetch Job
// @Tags Fetch Jobs
// @Produce json
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[fetchJobResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/fetch-jobs/{id} [get]
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

// @Summary List Fetch Jobs
// @Tags Fetch Jobs
// @Produce json
// @Param cursor query string false "Continuation cursor"
// @Param pageSize query int false "Page size" minimum(1) maximum(100) default(10)
// @Param sourceId query string false "Source ID" format(uuid)
// @Param status query string false "Fetch Job status" Enums(pending,running,succeeded,failed)
// @Success 200 {object} CursorResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/fetch-jobs [get]
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
