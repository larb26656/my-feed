package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/larb26656/my-feed/backend/internal/source/domain"
	"github.com/larb26656/my-feed/backend/internal/source/service"
)

// SourceAPI is the application surface the HTTP adapter needs.
// Kept as an interface so bootstrap can wrap cross-module hooks (e.g. fetch on deactivate).
type SourceAPI interface {
	CreateSource(ctx context.Context, in service.CreateSourceInput) (*domain.Source, error)
	ListSources(ctx context.Context, in service.ListSourcesInput) ([]*domain.Source, error)
	GetSource(ctx context.Context, id uuid.UUID) (*domain.Source, error)
	UpdateSource(ctx context.Context, id uuid.UUID, in service.UpdateSourceInput) (*domain.Source, error)
	DeactivateSource(ctx context.Context, id uuid.UUID) (*domain.Source, error)
	ReactivateSource(ctx context.Context, id uuid.UUID) (*domain.Source, error)
}

type Handler struct {
	svc SourceAPI
}

func NewHandler(svc SourceAPI) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.createSource)
	r.Get("/", h.listSources)
	r.Get("/{id}", h.getSource)
	r.Patch("/{id}", h.updateSource)
	r.Post("/{id}/deactivate", h.deactivateSource)
	r.Post("/{id}/reactivate", h.reactivateSource)
	return r
}

type createSourceRequest struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	RSSURL           string `json:"rssUrl"`
	YouTubeChannelID string `json:"youtubeChannelId"`
}

type updateSourceRequest struct {
	Name   string `json:"name"`
	RSSURL string `json:"rssUrl"`
}

type sourceResponse struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	RSSURL           *string `json:"rssUrl"`
	YouTubeChannelID *string `json:"youtubeChannelId,omitempty"`
	Active           bool    `json:"active"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

func toSourceResponse(s *domain.Source) sourceResponse {
	resp := sourceResponse{
		ID:        s.ID.String(),
		Name:      s.Name,
		Type:      string(s.Type),
		Active:    s.Active,
		CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: s.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
	if s.YouTubeChannelID != "" {
		resp.YouTubeChannelID = &s.YouTubeChannelID
	}
	if s.RSSURL != "" {
		url := s.RSSURL
		resp.RSSURL = &url
	}
	return resp
}

func (h *Handler) createSource(w http.ResponseWriter, r *http.Request) {
	var req createSourceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	s, err := h.svc.CreateSource(r.Context(), service.CreateSourceInput{
		Name:             req.Name,
		Type:             req.Type,
		RSSURL:           req.RSSURL,
		YouTubeChannelID: req.YouTubeChannelID,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toSourceResponse(s))
}

func (h *Handler) listSources(w http.ResponseWriter, r *http.Request) {
	includeDeactivated := r.URL.Query().Get("includeDeactivated") == "true"
	list, err := h.svc.ListSources(r.Context(), service.ListSourcesInput{
		IncludeDeactivated: includeDeactivated,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	out := make([]sourceResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toSourceResponse(s))
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) getSource(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "invalid source id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	s, err := h.svc.GetSource(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSourceResponse(s))
}

func (h *Handler) updateSource(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "invalid source id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var req updateSourceRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	s, err := h.svc.UpdateSource(r.Context(), id, service.UpdateSourceInput{
		Name:   req.Name,
		RSSURL: req.RSSURL,
	})
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSourceResponse(s))
}

func (h *Handler) deactivateSource(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "invalid source id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	s, err := h.svc.DeactivateSource(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSourceResponse(s))
}

func (h *Handler) reactivateSource(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r, "invalid source id")
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	s, err := h.svc.ReactivateSource(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toSourceResponse(s))
}

func parseID(r *http.Request, invalidMsg string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperr.Validation(invalidMsg)
	}
	return id, nil
}
