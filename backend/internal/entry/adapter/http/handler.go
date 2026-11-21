package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/entry/service"
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
	w, r := c.Writer, c.Request
	count, err := h.svc.CountSavedEntries(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, struct {
		Count int `json:"count"`
	}{Count: count})
}

type saveLinkRequest struct {
	Title       string  `json:"title" binding:"required,min=3,max=255"`
	Link        string  `json:"link" binding:"required,url"`
	Description string  `json:"description" binding:"max=1000"`
	Thumbnail   *string `json:"thumbnail" binding:"omitempty,url"`
	PublishedAt *string `json:"publishedAt" binding:"omitempty,datetime=2006-01-02"`
}
type ingestEntryRequest struct {
	SourceID    string  `json:"sourceId"`
	Title       string  `json:"title"`
	Link        string  `json:"link"`
	Description string  `json:"description"`
	Thumbnail   *string `json:"thumbnail"`
	PublishedAt *string `json:"publishedAt"`
}

type entryResponse struct {
	ID          string  `json:"id"`
	SourceID    string  `json:"sourceId"`
	Title       string  `json:"title"`
	Link        string  `json:"link"`
	Description string  `json:"description"`
	Thumbnail   *string `json:"thumbnail,omitempty"`
	PublishedAt *string `json:"publishedAt,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	ReadAt      *string `json:"readAt,omitempty"`
	SavedAt     *string `json:"savedAt,omitempty"`
}

func toEntryResponse(e *domain.Entry) entryResponse {
	return entryResponse{
		ID:          e.ID.String(),
		SourceID:    e.SourceID.String(),
		Title:       e.Title,
		Link:        e.Link,
		Description: e.Description,
		Thumbnail:   e.Thumbnail,
		PublishedAt: formatTimePtr(e.PublishedAt),
		CreatedAt:   e.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   e.UpdatedAt.UTC().Format(time.RFC3339Nano),
		ReadAt:      formatTimePtr(e.ReadAt),
		SavedAt:     formatTimePtr(e.SavedAt),
	}
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

func parseOptionalTime(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, *raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, *raw)
	}
	if err != nil {
		return nil, apperr.Validation("invalid publishedAt")
	}
	utc := t.UTC()
	return &utc, nil
}

func (h *Handler) saveLink(c *gin.Context) {
	var req saveLinkRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErrors validator.ValidationErrors

		if errors.As(err, &validationErrors) {
			fields := make(map[string]string)

			for _, e := range validationErrors {
				fields[e.Field()] = e.Tag()
			}

			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Validation failed",
				"errors":  fields,
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid JSON",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
	// if err := httpx.DecodeJSON(r, &req); err != nil {
	// 	httpx.WriteError(w, err)
	// 	return
	// }
	// publishedAt, err := parseOptionalTime(req.PublishedAt)
	// if err != nil {
	// 	httpx.WriteError(w, err)
	// 	return
	// }
	// entry, err := h.svc.SaveLink(r.Context(), service.SaveLinkInput{
	// 	Title:       req.Title,
	// 	Link:        req.Link,
	// 	Description: req.Description,
	// 	Thumbnail:   req.Thumbnail,
	// 	PublishedAt: publishedAt,
	// })
	// if err != nil {
	// 	httpx.WriteError(w, err)
	// 	return
	// }
	// httpx.WriteJSON(w, http.StatusCreated, toEntryResponse(entry))
}

func (h *Handler) ingestEntry(c *gin.Context) {
	w, r := c.Writer, c.Request
	var req ingestEntryRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	sourceID, err := uuid.Parse(req.SourceID)
	if err != nil {
		httpx.WriteError(w, apperr.Validation("invalid source id"))
		return
	}
	publishedAt, err := parseOptionalTime(req.PublishedAt)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	result, err := h.svc.IngestEntry(r.Context(), service.IngestEntryInput{
		SourceID:    sourceID,
		Title:       req.Title,
		Link:        req.Link,
		Description: req.Description,
		Thumbnail:   req.Thumbnail,
		PublishedAt: publishedAt,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	httpx.WriteJSON(w, status, toEntryResponse(result.Entry))
}

func (h *Handler) getEntry(c *gin.Context) {
	w, r := c.Writer, c.Request
	id, err := parseID(c, "invalid entry id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	entry, err := h.svc.GetEntry(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toEntryResponse(entry))
}

func (h *Handler) listEntries(c *gin.Context) {
	w, r := c.Writer, c.Request
	in := service.ListEntriesInput{}
	query := r.URL.Query()
	if values, ok := query["sourceId"]; ok {
		id, err := uuid.Parse(values[0])
		if err != nil {
			httpx.WriteError(w, apperr.Validation("invalid source id"))
			return
		}
		in.Criteria.SourceID = &id
	}
	for _, criterion := range []struct {
		name   string
		target *bool
	}{{"unread", &in.Criteria.Unread}, {"savedOnly", &in.Criteria.SavedOnly}} {
		if values, ok := query[criterion.name]; ok {
			if values[0] != "true" && values[0] != "false" {
				httpx.WriteError(w, apperr.Validation("invalid "+criterion.name))
				return
			}
			*criterion.target = values[0] == "true"
		}
	}
	if values, ok := query["cursor"]; ok {
		in.Cursor = &values[0]
	}
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			httpx.WriteError(w, apperr.Validation("invalid pageSize"))
			return
		}
		in.PageSize = &n
	}
	page, err := h.svc.ListEntries(r.Context(), in)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	resp := httpx.CursorResponse[entryResponse]{
		Data: make([]entryResponse, 0, len(page.Entries)),
		Page: httpx.CursorPage{NextCursor: page.NextCursor, HasNext: page.NextCursor != nil},
	}
	for _, e := range page.Entries {
		resp.Data = append(resp.Data, toEntryResponse(e))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
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
	w, r := c.Writer, c.Request
	id, err := parseID(c, "invalid entry id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	entry, err := fn(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toEntryResponse(entry))
}

func parseID(c *gin.Context, invalidMsg string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, apperr.Validation(invalidMsg)
	}
	return id, nil
}
