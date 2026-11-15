package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/source/domain"
	"github.com/larb26656/my-feed/backend/internal/source/repository"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

type CreateSourceInput struct {
	Name   string
	Type   string
	RSSURL string
}

func (s *Service) CreateSource(ctx context.Context, in CreateSourceInput) (*domain.Source, error) {
	typ := domain.Type(strings.TrimSpace(in.Type))
	switch typ {
	case domain.TypeManual:
		return nil, apperr.Validation("manual sources cannot be created by the reader")
	case domain.TypeRSS:
		src, err := domain.NewRSSSource(in.Name, in.RSSURL)
		if err != nil {
			return nil, err
		}
		if err := s.repo.CreateSource(ctx, src); err != nil {
			return nil, err
		}
		return src, nil
	default:
		return nil, apperr.Validation("source type must be rss")
	}
}

func (s *Service) GetSource(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	return s.repo.GetSourceByID(ctx, id)
}

type ListSourcesInput struct {
	IncludeDeactivated bool
}

func (s *Service) ListSources(ctx context.Context, in ListSourcesInput) ([]*domain.Source, error) {
	return s.repo.ListSources(ctx, repository.ListSourcesFilter{
		IncludeDeactivated: in.IncludeDeactivated,
	})
}

func (s *Service) DeactivateSource(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	src, err := s.repo.GetSourceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := src.Deactivate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSource(ctx, src); err != nil {
		return nil, err
	}
	return src, nil
}

func (s *Service) ReactivateSource(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	src, err := s.repo.GetSourceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := src.Reactivate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSource(ctx, src); err != nil {
		return nil, err
	}
	return src, nil
}

type UpdateSourceInput struct {
	Name   string
	RSSURL string
}

func (s *Service) UpdateSource(ctx context.Context, id uuid.UUID, in UpdateSourceInput) (*domain.Source, error) {
	src, err := s.repo.GetSourceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := src.UpdateDetails(in.Name, in.RSSURL); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSource(ctx, src); err != nil {
		return nil, err
	}
	return src, nil
}

// EnsureSavedLinks idempotently ensures the system manual Source exists.
func (s *Service) EnsureSavedLinks(ctx context.Context) (*domain.Source, error) {
	existing, err := s.repo.GetManualSource(ctx)
	if err == nil {
		return existing, nil
	}
	if !isNotFound(err) {
		return nil, err
	}

	src, err := domain.NewManualSource(domain.SavedLinksName)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateSource(ctx, src); err != nil {
		if isConflict(err) {
			return s.repo.GetManualSource(ctx)
		}
		return nil, err
	}
	return src, nil
}

func isNotFound(err error) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == apperr.CodeNotFound
}

func isConflict(err error) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == apperr.CodeConflict
}
