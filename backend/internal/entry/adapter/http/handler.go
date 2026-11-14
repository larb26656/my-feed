package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/larb26656/quick-feed/backend/internal/entry/domain"
	"github.com/larb26656/quick-feed/backend/internal/entry/service"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/quick-feed/backend/internal/platform/http"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/save-link", h.saveLink)
	r.Post("/", h.ingestEntry)
	r.Get("/", h.listEntries)
	r.Get("/{id}", h.getEntry)
	r.Post("/{id}/read", h.markEntryRead)
	r.Post("/{id}/clear-read", h.clearEntryRead)
	r.Post("/{id}/save", h.markEntrySaved)
	r.Post("/{id}/clear-save", h.clearEntrySaved)
	return r
}

type saveLinkRequest struct {
	Title       string  `json:"title"`
	Link        string  `json:"link"`
	Description string  `json:"description"`
	Thumbnail   *string `json:"thumbnail"`
	PublishedAt *string `json:"publishedAt"`
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

type listEntriesResponse struct {
	Entries    []entryResponse `json:"entries"`
	NextCursor *string         `json:"nextCursor,omitempty"`
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

func (h *Handler) saveLink(w http.ResponseWriter, r *http.Request) {
	var req saveLinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	publishedAt, err := parseOptionalTime(req.PublishedAt)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	entry, err := h.svc.SaveLink(r.Context(), service.SaveLinkInput{
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
	httpx.WriteJSON(w, http.StatusCreated, toEntryResponse(entry))
}

func (h *Handler) ingestEntry(w http.ResponseWriter, r *http.Request) {
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
	entry, err := h.svc.IngestEntry(r.Context(), service.IngestEntryInput{
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
	httpx.WriteJSON(w, http.StatusCreated, toEntryResponse(entry))
}

func (h *Handler) getEntry(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "invalid entry id")
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

func (h *Handler) listEntries(w http.ResponseWriter, r *http.Request) {
	in := service.ListEntriesInput{}
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		in.Cursor = &raw
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
	resp := listEntriesResponse{
		Entries:    make([]entryResponse, 0, len(page.Entries)),
		NextCursor: page.NextCursor,
	}
	for _, e := range page.Entries {
		resp.Entries = append(resp.Entries, toEntryResponse(e))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}

type entryMutator func(ctx context.Context, id uuid.UUID) (*domain.Entry, error)

func (h *Handler) markEntryRead(w http.ResponseWriter, r *http.Request) {
	h.mutateEntry(w, r, h.svc.MarkEntryRead)
}

func (h *Handler) clearEntryRead(w http.ResponseWriter, r *http.Request) {
	h.mutateEntry(w, r, h.svc.ClearEntryRead)
}

func (h *Handler) markEntrySaved(w http.ResponseWriter, r *http.Request) {
	h.mutateEntry(w, r, h.svc.MarkEntrySaved)
}

func (h *Handler) clearEntrySaved(w http.ResponseWriter, r *http.Request) {
	h.mutateEntry(w, r, h.svc.ClearEntrySaved)
}

func (h *Handler) mutateEntry(w http.ResponseWriter, r *http.Request, fn entryMutator) {
	id, err := parseID(r, "invalid entry id")
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

func parseID(r *http.Request, invalidMsg string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperr.Validation(invalidMsg)
	}
	return id, nil
}
