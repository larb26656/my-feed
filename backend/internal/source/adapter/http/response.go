package http

import (
	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/source/domain"
	"time"
)

type sourceResponse struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	RSSURL           *string   `json:"rssUrl"`
	YouTubeChannelID *string   `json:"youtubeChannelId,omitempty"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func toSourceResponse(s *domain.Source) sourceResponse {
	resp := sourceResponse{ID: s.ID, Name: s.Name, Type: string(s.Type), Active: s.Active, CreatedAt: s.CreatedAt.UTC(), UpdatedAt: s.UpdatedAt.UTC()}
	if s.RSSURL != "" {
		resp.RSSURL = &s.RSSURL
	}
	if s.YouTubeChannelID != "" {
		resp.YouTubeChannelID = &s.YouTubeChannelID
	}
	return resp
}
