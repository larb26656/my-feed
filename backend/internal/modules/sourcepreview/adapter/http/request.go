package http

type previewSourceRequest struct {
	Type             string `json:"type" binding:"required"`
	RSSURL           string `json:"rssUrl"`
	YouTubeChannelID string `json:"youtubeChannelId"`
}
