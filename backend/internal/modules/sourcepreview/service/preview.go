// Package service orchestrates Source preview without persisting entries.
package service

import (
	"context"
	"time"

	fetchapi "github.com/larb26656/my-feed/backend/internal/modules/fetch/api"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	previewapi "github.com/larb26656/my-feed/backend/internal/modules/sourcepreview/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type Service struct {
	sources sourceapi.PreviewValidator
	obtain  fetchapi.Obtain
}

func New(sources sourceapi.PreviewValidator, obtain fetchapi.Obtain) *Service {
	return &Service{sources: sources, obtain: obtain}
}

func (s *Service) Preview(ctx context.Context, in previewapi.Input) (*previewapi.Entry, error) {
	ref, err := s.sources.ValidatePreview(ctx, sourceapi.PreviewInput{Type: in.Type, RSSURL: in.RSSURL, YouTubeChannelID: in.YouTubeChannelID})
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
	return &previewapi.Entry{Title: e.Title, Link: e.Link, Description: e.Description, Thumbnail: e.Thumbnail, PublishedAt: e.PublishedAt}, nil
}
