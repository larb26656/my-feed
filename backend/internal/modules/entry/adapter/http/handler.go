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

// EntryAPI is the application surface the HTTP adapter needs.
type EntryAPI interface {
	SaveLink(ctx context.Context, in service.SaveLinkInput) (*domain.Entry, error)
	IngestEntry(ctx context.Context, in service.IngestEntryInput) (*service.IngestEntryResult, error)
	GetEntry(ctx context.Context, id uuid.UUID) (*domain.Entry, error)
	ListEntries(ctx context.Context, in service.ListEntriesInput) (*service.ListEntriesResult, error)
	CountSavedEntries(ctx context.Context) (int, error)
	MarkEntryRead(ctx context.Context, id uuid.UUID) (*domain.Entry, error)
	ClearEntryRead(ctx context.Context, id uuid.UUID) (*domain.Entry, error)
	MarkEntrySaved(ctx context.Context, id uuid.UUID) (*domain.Entry, error)
	ClearEntrySaved(ctx context.Context, id uuid.UUID) (*domain.Entry, error)
}

type Handler struct {
	svc EntryAPI
}

func NewHandler(svc EntryAPI) *Handler {
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

func (h *Handler) savedCount(c *gin.Context) {
	count, err := h.svc.CountSavedEntries(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, httpx.Success(savedCountResponse{Count: count}))
}

func (h *Handler) saveLink(c *gin.Context) {
	var req saveLinkRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	entry, err := h.svc.SaveLink(c.Request.Context(), req.input())
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, httpx.Success(toEntryResponse(entry)))
}

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

func (h *Handler) markEntryRead(c *gin.Context) {
	h.mutateEntry(c, h.svc.MarkEntryRead)
}

func (h *Handler) clearEntryRead(c *gin.Context) {
	h.mutateEntry(c, h.svc.ClearEntryRead)
}

func (h *Handler) markEntrySaved(c *gin.Context) {
	h.mutateEntry(c, h.svc.MarkEntrySaved)
}

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
