package service

import (
	"context"
	"time"

	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

func (s *Service) Preview(ctx context.Context, in sourceapi.PreviewInput) (*sourceapi.PreviewEntry, error) {
	if s.types == nil {
		return nil, apperr.InvalidInput.New("source type registry unavailable")
	}
	config, _, err := s.types.Prepare(in.Type, in.Config)
	if err != nil {
		return nil, err
	}
	ref := &obtainapi.Input{Type: obtainapi.Provider(in.Type), Config: config}
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
