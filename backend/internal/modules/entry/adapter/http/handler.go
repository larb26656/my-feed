package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/save-link", h.saveLink)
	r.POST("/", h.ingestEntry)
	r.POST("", h.ingestEntry)
	r.GET("/", h.listEntries)
	r.GET("", h.listEntries)
	r.GET("/saved-count", h.savedCount)
	r.GET("/:id", h.getEntry)
	r.POST("/:id/read", h.markEntryRead)
	r.POST("/:id/clear-read", h.clearEntryRead)
	r.POST("/:id/save", h.markEntrySaved)
	r.POST("/:id/clear-save", h.clearEntrySaved)
}

// @Summary Count saved Entries
// @Tags Entries
// @Produce json
// @Success 200 {object} httpx.SuccessResponse[savedCountResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/entries/saved-count [get]
func (h *Handler) savedCount(c *gin.Context) {
	count, err := h.svc.CountSavedEntries(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(savedCountResponse{Count: count}))
}

// @Summary Save link
// @Tags Entries
// @Produce json
// @Accept json
// @Param request body saveLinkRequest true "Request body"
// @Success 201 {object} httpx.SuccessResponse[entryResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/entries/save-link [post]
func (h *Handler) saveLink(c *gin.Context) {
	var req saveLinkRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	result, err := h.svc.SaveLink(c.Request.Context(), req.input())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, httpx.Success(toEntryResponse(result.Entry)))
}

// @Summary Ingest Entry
// @Tags Entries
// @Produce json
// @Accept json
// @Param request body ingestEntryRequest true "Request body"
// @Success 201 {object} httpx.SuccessResponse[entryResponse]
// @Success 200 {object} httpx.SuccessResponse[entryResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/entries [post]
func (h *Handler) ingestEntry(c *gin.Context) {
	var req ingestEntryRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	result, err := h.svc.IngestEntry(c.Request.Context(), req.input())
	if err != nil {
		_ = c.Error(err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	c.JSON(status, httpx.Success(toEntryResponse(result.Entry)))
}

// @Summary Get Entry
// @Tags Entries
// @Produce json
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[entryResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/entries/{id} [get]
func (h *Handler) getEntry(c *gin.Context) {
	id, err := parseID(c, "invalid entry id")
	if err != nil {
		_ = c.Error(err)
		return
	}
	entry, err := h.svc.GetEntry(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toEntryResponse(entry)))
}

// @Summary List Entries
// @Tags Entries
// @Produce json
// @Param cursor query string false "Continuation cursor"
// @Param pageSize query int false "Page size" minimum(1) maximum(100) default(10)
// @Param sourceId query string false "Source ID" format(uuid)
// @Param unread query bool false "Only unread Entries"
// @Param savedOnly query bool false "Only saved Entries"
// @Param sortDirection query string false "Newest or oldest. Saved lists use the time the Entry was saved. The Saved links Source uses the time the link was recorded. Other lists use the publication time, or the recorded time when publication time is absent" Enums(asc, desc) default(desc)
// @Success 200 {object} CursorResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/entries [get]
func (h *Handler) listEntries(c *gin.Context) {
	req := listEntriesQuery{CursorPageOption: httpx.NewCursorPageOption()}
	if err := c.ShouldBindQuery(&req); err != nil {
		_ = c.Error(apperr.InvalidInput.Wrap(err, "Invalid Entry query"))
		return
	}
	page, err := h.svc.ListEntries(c.Request.Context(), req.input())
	if err != nil {
		_ = c.Error(err)
		return
	}
	items := make([]entryResponse, 0, len(page.Entries))
	for _, entry := range page.Entries {
		items = append(items, toEntryResponse(entry))
	}
	var nextCursor string
	if page.NextCursor != nil {
		nextCursor = *page.NextCursor
	}
	c.JSON(http.StatusOK, httpx.Cursor(items, req.PageSize, nextCursor))
}

type entryMutator func(ctx context.Context, id uuid.UUID) (*domain.Entry, error)

// @Summary Mark Entry read
// @Tags Entries
// @Produce json
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[entryResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/entries/{id}/read [post]
func (h *Handler) markEntryRead(c *gin.Context) {
	h.mutateEntry(c, h.svc.MarkEntryRead)
}

// @Summary Clear Entry read
// @Tags Entries
// @Produce json
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[entryResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/entries/{id}/clear-read [post]
func (h *Handler) clearEntryRead(c *gin.Context) {
	h.mutateEntry(c, h.svc.ClearEntryRead)
}

// @Summary Save Entry
// @Tags Entries
// @Produce json
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[entryResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/entries/{id}/save [post]
func (h *Handler) markEntrySaved(c *gin.Context) {
	h.mutateEntry(c, h.svc.MarkEntrySaved)
}

// @Summary Clear saved Entry
// @Tags Entries
// @Produce json
// @Param id path string true "Resource ID" format(uuid)
// @Success 200 {object} httpx.SuccessResponse[entryResponse]
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/entries/{id}/clear-save [post]
func (h *Handler) clearEntrySaved(c *gin.Context) {
	h.mutateEntry(c, h.svc.ClearEntrySaved)
}

func (h *Handler) mutateEntry(c *gin.Context, fn entryMutator) {
	id, err := parseID(c, "invalid entry id")
	if err != nil {
		_ = c.Error(err)
		return
	}
	entry, err := fn(c.Request.Context(), id)
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(toEntryResponse(entry)))
}

func parseID(c *gin.Context, invalidMsg string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, apperr.InvalidInput.Wrap(err, invalidMsg)
	}
	return id, nil
}
