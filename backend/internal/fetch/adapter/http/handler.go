package http

import (
	"net/http"
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

type listJobsQuery struct {
	SourceID *string `schema:"sourceId" validate:"omitempty,uuid"`
	Status   *string `schema:"status" validate:"omitempty,oneof=pending running succeeded failed"`
	Cursor   *string `schema:"cursor"`
	PageSize *int    `schema:"pageSize" validate:"omitempty,min=1,max=100"`
}

func (h *Handler) listJobs(w http.ResponseWriter, r *http.Request) {
	query, err := httpx.DecodeQuery[listJobsQuery](r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	in := service.ListFetchJobsInput{Cursor: query.Cursor, PageSize: query.PageSize}
	if query.SourceID != nil {
		id := uuid.MustParse(*query.SourceID)
		in.Criteria.SourceID = &id
	}
	if query.Status != nil {
		status := domain.Status(*query.Status)
		in.Criteria.Status = &status
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
