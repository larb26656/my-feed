package http

import (
	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/fetch/domain"
	"github.com/larb26656/my-feed/backend/internal/fetch/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

type createFetchJobRequest struct {
	SourceID uuid.UUID `json:"sourceId" binding:"required"`
}
type queryUUID struct{ uuid.UUID }

func (id *queryUUID) UnmarshalParam(value string) error { return id.UUID.UnmarshalText([]byte(value)) }

type listFetchJobsQuery struct {
	httpx.CursorPageOption
	SourceID *queryUUID     `form:"sourceId"`
	Status   *domain.Status `form:"status" binding:"omitempty,oneof=pending running succeeded failed"`
}

func (req listFetchJobsQuery) input() service.ListFetchJobsInput {
	in := service.ListFetchJobsInput{PageSize: &req.PageSize}
	if req.SourceID != nil {
		in.Criteria.SourceID = &req.SourceID.UUID
	}
	in.Criteria.Status = req.Status
	if req.Cursor != "" {
		in.Cursor = &req.Cursor
	}
	return in
}
