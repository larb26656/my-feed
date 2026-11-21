package service

import (
	"context"

	"github.com/google/uuid"

	sourceapi "github.com/larb26656/my-feed/backend/internal/source/api"
	"github.com/larb26656/my-feed/backend/internal/source/domain"
)

var (
	_ sourceapi.Reader      = (*Service)(nil)
	_ sourceapi.EntryReader = (*Service)(nil)
)

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*sourceapi.SourceRef, error) {
	src, err := s.GetSource(ctx, id)
	if err != nil {
		return nil, err
	}
	return toSourceRef(src), nil
}

func (s *Service) GetManual(ctx context.Context) (*sourceapi.SourceRef, error) {
	src, err := s.repo.GetManualSource(ctx)
	if err != nil {
		return nil, err
	}
	return toSourceRef(src), nil
}

func (s *Service) ListActiveFetchable(ctx context.Context) ([]sourceapi.SourceRef, error) {
	list, err := s.ListSources(ctx, ListSourcesInput{})
	if err != nil {
		return nil, err
	}
	out := make([]sourceapi.SourceRef, 0, len(list))
	for _, src := range list {
		if src.Type != domain.TypeRSS && src.Type != domain.TypeYouTube {
			continue
		}
		out = append(out, *toSourceRef(src))
	}
	return out, nil
}

func toSourceRef(src *domain.Source) *sourceapi.SourceRef {
	return &sourceapi.SourceRef{
		ID:               src.ID,
		Type:             sourceapi.SourceType(src.Type),
		Active:           src.Active,
		RSSURL:           src.RSSURL,
		YouTubeChannelID: src.YouTubeChannelID,
	}
}
