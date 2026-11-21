package http

import "encoding/json"

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
	IncludeDeactivated bool `form:"includeDeactivated"`
}
