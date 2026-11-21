package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/source/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

var _ sourceapi.TransactionalLifecycle = (*Service)(nil)

func (s *Service) WithTx(tx database.Transaction) sourceapi.Lifecycle {
	return New(s.repo.WithTx(tx))
}

type CreateSourceInput = sourceapi.CreateSourceInput

func (s *Service) CreateSource(ctx context.Context, in CreateSourceInput) (*sourceapi.Source, error) {
	typ := domain.Type(strings.TrimSpace(in.Type))
	switch typ {
	case domain.TypeManual:
		return nil, apperr.InvalidInput.New("manual sources cannot be created by the reader")
	case domain.TypeYouTube:
		src, err := domain.NewYouTubeSource(in.Name, in.YouTubeChannelID)
		if err != nil {
			return nil, err
		}
		if err := s.repo.CreateSource(ctx, src); err != nil {
			return nil, err
		}
		return lifecycleSnapshot(src), nil
	case domain.TypeRSS:
		src, err := domain.NewRSSSource(in.Name, in.RSSURL)
		if err != nil {
			return nil, err
		}
		if err := s.repo.CreateSource(ctx, src); err != nil {
			return nil, err
		}
		return lifecycleSnapshot(src), nil
	default:
		return nil, apperr.InvalidInput.New("source type must be rss or youtube")
	}
}

func (s *Service) GetSource(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	return s.repo.GetSourceByID(ctx, id)
}

type ListSourcesInput struct {
	IncludeDeactivated bool
}

type ListSourcesPageInput struct {
	NameLike, TypeEqual, SortBy, SortDirection string
	ActiveEqual                                *bool
	Page, PageSize                             int
}

func (s *Service) ListSourcesPage(ctx context.Context, in ListSourcesPageInput) ([]*domain.Source, int64, error) {
	if in.Page < 1 || in.PageSize < 1 || in.PageSize > 100 {
		return nil, 0, apperr.InvalidInput.New("invalid source page")
	}
	items, total, err := s.repo.ListSourcesPage(ctx, repository.SourcePageQuery{
		NameLike: in.NameLike, TypeEqual: in.TypeEqual, ActiveEqual: in.ActiveEqual,
		SortBy: in.SortBy, SortDirection: in.SortDirection, Page: in.Page, PageSize: in.PageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list sources page: %w", err)
	}
	return items, total, nil
}

func (s *Service) ListSources(ctx context.Context, in ListSourcesInput) ([]*domain.Source, error) {
	return s.repo.ListSources(ctx, repository.ListSourcesFilter{
		IncludeDeactivated: in.IncludeDeactivated,
	})
}

func (s *Service) DeactivateSource(ctx context.Context, id uuid.UUID) (*sourceapi.Source, error) {
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
	return lifecycleSnapshot(src), nil
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
	var ae *apperr.AppError
	return errors.As(err, &ae) && ae.Kind == apperr.KindNotFound
}

func isConflict(err error) bool {
	var ae *apperr.AppError
	return errors.As(err, &ae) && ae.Definition == domain.ErrConflict
}

func lifecycleSnapshot(src *domain.Source) *sourceapi.Source {
	return &sourceapi.Source{
		ID:               src.ID,
		Name:             src.Name,
		Type:             sourceapi.SourceType(src.Type),
		RSSURL:           src.RSSURL,
		YouTubeChannelID: src.YouTubeChannelID,
		Active:           src.Active,
		CreatedAt:        src.CreatedAt,
		UpdatedAt:        src.UpdatedAt,
	}
}
