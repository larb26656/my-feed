package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/quick-feed/backend/internal/platform/http"
	"github.com/larb26656/quick-feed/backend/internal/source/domain"
	"github.com/larb26656/quick-feed/backend/internal/source/service"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.createSource)
	r.Get("/", h.listSources)
	r.Get("/{id}", h.getSource)
	r.Post("/{id}/deactivate", h.deactivateSource)
	r.Post("/{id}/reactivate", h.reactivateSource)
	return r
}

type createSourceRequest struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	RSSURL string `json:"rssUrl"`
}

type sourceResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	RSSURL    *string `json:"rssUrl"`
	Active    bool    `json:"active"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
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
		Name:   req.Name,
		Type:   req.Type,
		RSSURL: req.RSSURL,
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
