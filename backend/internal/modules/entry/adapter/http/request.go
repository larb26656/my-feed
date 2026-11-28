package http

import (
	"encoding/json"
	"strings"
	"time"

	sortoption "github.com/larb26656/my-feed/backend/internal/platform/sort"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

type saveLinkRequest struct {
	Title       string         `json:"title" binding:"required,min=3,max=255"`
	Link        normalizedLink `json:"link" binding:"omitempty,url"`
	Description string         `json:"description" binding:"max=1000"`
	Thumbnail   *string        `json:"thumbnail" binding:"omitempty,url"`
	PublishedAt *time.Time     `json:"publishedAt"`
}

func (req saveLinkRequest) input() service.SaveLinkInput {
	return service.SaveLinkInput{
		Title:       req.Title,
		Link:        string(req.Link),
		Description: req.Description,
		Thumbnail:   req.Thumbnail,
		PublishedAt: req.PublishedAt,
	}
}

type ingestEntryRequest struct {
	SourceID    uuid.UUID  `json:"sourceId" binding:"required"`
	Title       string     `json:"title" binding:"required"`
	Link        string     `json:"link"`
	Description string     `json:"description"`
	Thumbnail   *string    `json:"thumbnail"`
	PublishedAt *time.Time `json:"publishedAt"`
}

func (req ingestEntryRequest) input() service.IngestEntryInput {
	return service.IngestEntryInput{
		SourceID:    req.SourceID,
		Title:       req.Title,
		Link:        req.Link,
		Description: req.Description,
		Thumbnail:   req.Thumbnail,
		PublishedAt: req.PublishedAt,
	}
}

type listEntriesQuery struct {
	httpx.CursorPageOption
	SourceID        *queryUUID           `form:"sourceId"`
	Unread          bool                 `form:"unread"`
	SavedOnly       bool                 `form:"savedOnly"`
	CreatedAtSort   sortoption.Direction `form:"createdAtSort"`
	PublishedAtSort sortoption.Direction `form:"publishedAtSort"`
	SavedAtSort     sortoption.Direction `form:"savedAtSort"`
}

type queryUUID struct {
	uuid.UUID
}

func (id *queryUUID) UnmarshalParam(value string) error {
	return id.UnmarshalText([]byte(value))
}

func (req listEntriesQuery) input() service.ListEntriesInput {
	in := service.ListEntriesInput{PageSize: &req.PageSize}
	in.Criteria.Unread = req.Unread
	in.Criteria.SavedOnly = req.SavedOnly
	in.Sort.CreatedAtSort = req.CreatedAtSort
	in.Sort.PublishedAtSort = req.PublishedAtSort
	in.Sort.SavedAtSort = req.SavedAtSort
	if req.SourceID != nil {
		in.Criteria.SourceID = &req.SourceID.UUID
	}
	if req.Cursor != "" {
		in.Cursor = &req.Cursor
	}
	return in
}

type normalizedLink string

func (l *normalizedLink) UnmarshalJSON(data []byte) error {
	var value *string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value != nil {
		*l = normalizedLink(strings.TrimSpace(*value))
	}
	return nil
}
