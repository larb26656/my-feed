package service

import (
	"context"
	"net/url"
	"time"

	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/obtain/api"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

func validatePreview(in sourceapi.PreviewInput) (*obtainapi.Input, error) {
	var ref obtainapi.Input
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
		ref = obtainapi.Input{Type: obtainapi.ProviderRSS, RSSURL: src.RSSURL}
	case "youtube":
		src, err := domain.NewYouTubeSource("preview", in.YouTubeChannelID)
		if err != nil {
			return nil, err
		}
		ref = obtainapi.Input{Type: obtainapi.ProviderYouTube, YouTubeChannelID: src.YouTubeChannelID}
	default:
		return nil, apperr.InvalidInput.New("source type must be rss or youtube")
	}
	return &ref, nil
}

func (s *Service) Preview(ctx context.Context, in sourceapi.PreviewInput) (*sourceapi.PreviewEntry, error) {
	ref, err := validatePreview(in)
	if err != nil {
		return nil, err
	}
	if s.obtain == nil {
		return nil, apperr.InvalidInput.New("preview is unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	entries, err := s.obtain.Obtain(ctx, ref)
	if err != nil {
		return nil, apperr.InvalidInput.Wrap(err, "Could not fetch or read the origin")
	}
	if len(entries) == 0 {
		return nil, nil
	}
	e := entries[0]
	return &sourceapi.PreviewEntry{Title: e.Title, Link: e.Link, Description: e.Description, Thumbnail: e.Thumbnail, PublishedAt: e.PublishedAt}, nil
}
