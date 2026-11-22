package service

import (
	"context"
	"net/url"

	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

func (s *Service) ValidatePreview(_ context.Context, in sourceapi.PreviewInput) (*sourceapi.SourceRef, error) {
	var ref sourceapi.SourceRef
	switch in.Type {
	case "rss":
		src, err := domain.NewRSSSource("preview", in.RSSURL)
		if err != nil {
			return nil, err
		}
		u, err := url.Parse(src.RSSURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return nil, apperr.InvalidInput.New("invalid rss feed url")
		}
		ref = sourceapi.SourceRef{Type: sourceapi.SourceTypeRSS, RSSURL: src.RSSURL}
	case "youtube":
		src, err := domain.NewYouTubeSource("preview", in.YouTubeChannelID)
		if err != nil {
			return nil, err
		}
		ref = sourceapi.SourceRef{Type: sourceapi.SourceTypeYouTube, YouTubeChannelID: src.YouTubeChannelID}
	default:
		return nil, apperr.InvalidInput.New("source type must be rss or youtube")
	}
	return &ref, nil
}
