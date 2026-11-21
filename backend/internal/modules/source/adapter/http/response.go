package http

import (
	"time"

	"github.com/google/uuid"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
)

type sourceResponse struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	RSSURL           *string   `json:"rssUrl" extensions:"x-nullable"`
	YouTubeChannelID *string   `json:"youtubeChannelId,omitempty"`
	Active           bool      `json:"active"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func toSourceResponse(s *domain.Source) sourceResponse {
	resp := sourceResponse{ID: s.ID, Name: s.Name, Type: string(s.Type), Active: s.Active, CreatedAt: s.CreatedAt.UTC(), UpdatedAt: s.UpdatedAt.UTC()}
	resp.RSSURL = optionalString(s.RSSURL)
	resp.YouTubeChannelID = optionalString(s.YouTubeChannelID)
	return resp
}

func toLifecycleResponse(s *sourceapi.Source) sourceResponse {
	return sourceResponse{
		ID:               s.ID,
		Name:             s.Name,
		Type:             string(s.Type),
		RSSURL:           optionalString(s.RSSURL),
		YouTubeChannelID: optionalString(s.YouTubeChannelID),
		Active:           s.Active,
		CreatedAt:        s.CreatedAt.UTC(),
		UpdatedAt:        s.UpdatedAt.UTC(),
	}
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
