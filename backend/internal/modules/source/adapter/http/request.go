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
	Type             *json.RawMessage `json:"type" binding:"isdefault" swaggerignore:"true"`
	YouTubeChannelID *json.RawMessage `json:"youtubeChannelId" binding:"isdefault" swaggerignore:"true"`
	Name             string           `json:"name" binding:"required"`
	RSSURL           string           `json:"rssUrl"`
}
type listSourcesQuery struct {
	httpx.PageOption
	NameLike      string `form:"nameLike"`
	TypeEqual     string `form:"typeEqual" binding:"omitempty,oneof=rss youtube manual"`
	ActiveEqual   *bool  `form:"activeEqual"`
	SortBy        string `form:"sortBy" binding:"oneof=name type active createdAt"`
	SortDirection string `form:"sortDirection" binding:"oneof=asc desc"`
}
