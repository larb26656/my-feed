package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/fetch/domain"
	"github.com/larb26656/my-feed/backend/internal/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.createManual)
	r.Get("/", h.listJobs)
	r.Get("/{id}", h.getJob)
	return r
}

type createFetchJobRequest struct {
	SourceID string `json:"sourceId"`
}

type fetchJobResponse struct {
	ID             string `json:"id"`
	SourceID       string `json:"sourceId"`
	Status         string `json:"status"`
	Attempt        int    `json:"attempt"`
	MaxAttempts    int    `json:"maxAttempts"`
	EntriesCreated int    `json:"entriesCreated"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

func toFetchJobResponse(job *domain.FetchJob) fetchJobResponse {
	return fetchJobResponse{
		ID:             job.ID.String(),
		SourceID:       job.SourceID.String(),
		Status:         string(job.Status),
		Attempt:        job.Attempt,
		MaxAttempts:    job.MaxAttempts,
		EntriesCreated: job.EntriesCreated,
		CreatedAt:      job.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      job.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *Handler) createManual(w http.ResponseWriter, r *http.Request) {
	var req createFetchJobRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	sourceID, err := uuid.Parse(req.SourceID)
	if err != nil {
		httpx.WriteError(w, apperr.Validation("invalid source id"))
		return
	}
	job, err := h.svc.CreateManualFetchJob(r.Context(), sourceID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toFetchJobResponse(job))
}

func (h *Handler) getJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, apperr.Validation("invalid fetch job id"))
		return
	}
	job, err := h.svc.GetFetchJob(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toFetchJobResponse(job))
}

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	in := service.ListFetchJobsInput{}
	query := r.URL.Query()
	if values, ok := query["sourceId"]; ok {
		if len(values) != 1 {
			httpx.WriteError(w, apperr.Validation("invalid source id"))
			return
		}
		id, err := uuid.Parse(values[0])
		if err != nil {
			httpx.WriteError(w, apperr.Validation("invalid source id"))
			return
		}
		in.Criteria.SourceID = &id
	}
	if values, ok := query["status"]; ok {
		if len(values) != 1 {
			httpx.WriteError(w, apperr.Validation("invalid status"))
			return
		}
		status := domain.Status(values[0])
		switch status {
		case domain.StatusPending, domain.StatusRunning, domain.StatusSucceeded, domain.StatusFailed:
			in.Criteria.Status = &status
		default:
			httpx.WriteError(w, apperr.Validation("invalid status"))
			return
		}
	}
	if values, ok := query["cursor"]; ok {
		if len(values) != 1 {
			httpx.WriteError(w, apperr.Validation("invalid cursor"))
			return
		}
		in.Cursor = &values[0]
	}
	if values, ok := query["pageSize"]; ok {
		if len(values) != 1 {
			httpx.WriteError(w, apperr.Validation("invalid pageSize"))
			return
		}
		n, err := strconv.Atoi(values[0])
		if err != nil || n < 1 || n > 100 {
			httpx.WriteError(w, apperr.Validation("pageSize must be between 1 and 100"))
			return
		}
		in.PageSize = &n
	}
	page, err := h.svc.ListFetchJobs(r.Context(), in)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	resp := httpx.CursorResponse[fetchJobResponse]{
		Data: make([]fetchJobResponse, 0, len(page.Jobs)),
		Page: httpx.CursorPage{NextCursor: page.NextCursor, HasNext: page.NextCursor != nil},
	}
	for _, job := range page.Jobs {
		resp.Data = append(resp.Data, toFetchJobResponse(job))
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
