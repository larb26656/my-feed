package http

import (
	"encoding/json"

	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

type createSourceRequest struct {
	Name             string `json:"name" binding:"required"`
	Type             string `json:"type" binding:"required"`
	RSSURL           string `json:"rssUrl"`
	YouTubeChannelID string `json:"youtubeChannelId"`
}
type updateSourceRequest struct {
	Type             *json.RawMessage `json:"type" binding:"isdefault"`
	YouTubeChannelID *json.RawMessage `json:"youtubeChannelId" binding:"isdefault"`
	Name             string           `json:"name" binding:"required"`
	RSSURL           string           `json:"rssUrl"`
}
type listSourcesQuery struct {
	httpx.PageOption
	IncludeDeactivated *bool  `form:"includeDeactivated"`
	NameLike           string `form:"nameLike"`
	TypeEqual          string `form:"typeEqual" binding:"omitempty,oneof=rss youtube manual"`
	ActiveEqual        *bool  `form:"activeEqual"`
	SortBy             string `form:"sortBy" binding:"oneof=name type active createdAt"`
	SortDirection      string `form:"sortDirection" binding:"oneof=asc desc"`
}
