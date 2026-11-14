package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/larb26656/quick-feed/backend/internal/feed/domain"
	"github.com/larb26656/quick-feed/backend/internal/feed/service"
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
	r.Post("/", h.create)
	r.Get("/{id}", h.get)
	r.Post("/{id}/publish", h.publish)
	r.Post("/{id}/archive", h.archive)
	return r
}

type createRequest struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type feedResponse struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

func toResponse(f *domain.Feed) feedResponse {
	return feedResponse{
		ID:          f.ID.String(),
		Slug:        f.Slug,
		Title:       f.Title,
		Description: f.Description,
		Status:      string(f.Status),
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	f, err := h.svc.Create(r.Context(), service.CreateInput{
		Slug: req.Slug, Title: req.Title, Description: req.Description,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toResponse(f))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseFeedID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	f, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(f))
}

func (h *Handler) publish(w http.ResponseWriter, r *http.Request) {
	id, err := parseFeedID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	f, err := h.svc.Publish(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(f))
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
	id, err := parseFeedID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	f, err := h.svc.Archive(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(f))
}

func parseFeedID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperr.Validation("invalid feed id")
	}
	return id, nil
}
