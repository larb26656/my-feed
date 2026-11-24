package http

import (
	"encoding/json"

	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	sortoption "github.com/larb26656/my-feed/backend/internal/platform/sort"
)

type createSourceRequest struct {
	Config map[string]any `json:"config" binding:"required"`
	Name   string         `json:"name" binding:"required"`
	Type   string         `json:"type" binding:"required"`
}
type updateSourceRequest struct {
	Config           map[string]any   `json:"config"`
	Type             *json.RawMessage `json:"type" binding:"isdefault" swaggerignore:"true"`
	YouTubeChannelID *json.RawMessage `json:"youtubeChannelId" binding:"isdefault" swaggerignore:"true"`
	Name             string           `json:"name" binding:"required"`
	RSSURL           *json.RawMessage `json:"rssUrl" binding:"isdefault" swaggerignore:"true"`
}
type listSourcesQuery struct {
	httpx.PageOption
	NameLike      string               `form:"nameLike"`
	TypeEqual     string               `form:"typeEqual"`
	ActiveEqual   *bool                `form:"activeEqual"`
	NameSort      sortoption.Direction `form:"nameSort"`
	TypeSort      sortoption.Direction `form:"typeSort"`
	ActiveSort    sortoption.Direction `form:"activeSort"`
	CreatedAtSort sortoption.Direction `form:"createdAtSort"`
}

type previewSourceRequest struct {
	Config map[string]any `json:"config" binding:"required"`
	Type   string         `json:"type" binding:"required"`
}
