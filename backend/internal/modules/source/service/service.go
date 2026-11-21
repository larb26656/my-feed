package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"strings"
	"sync"

	fileapi "github.com/larb26656/my-feed/backend/internal/modules/file/api"

	"github.com/google/uuid"

	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/source/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

type Service struct {
	repo    repository.Repository
	files   fileapi.Storage
	imageMu *sync.Mutex
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo, imageMu: &sync.Mutex{}}
}

var _ sourceapi.TransactionalLifecycle = (*Service)(nil)

func (s *Service) WithTx(tx database.Transaction) sourceapi.Lifecycle {
	return &Service{repo: s.repo.WithTx(tx), files: s.files, imageMu: s.imageMu}
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
		ProfileImageID:   src.ProfileImageID,
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

// SetFileStorage injects the file module at bootstrap.
func (s *Service) SetFileStorage(files fileapi.Storage) { s.files = files }

func (s *Service) ReplaceProfileImage(ctx context.Context, id uuid.UUID, data []byte) (*domain.Source, error) {
	s.imageMu.Lock()
	defer s.imageMu.Unlock()
	src, err := s.repo.GetSourceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get source for profile image: %w", err)
	}
	if err := src.ValidateProfileImageChange(); err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > 2*1024*1024 {
		return nil, apperr.InvalidInput.New("image must be at most 2 MB")
	}
	_, format, err := image.Decode(bytes.NewReader(data))
	if err != nil || (format != "png" && format != "jpeg") {
		return nil, apperr.InvalidInput.New("image must be a valid JPEG or PNG")
	}
	if s.files == nil {
		return nil, fmt.Errorf("file storage unavailable")
	}
	newID, err := s.files.Save(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("save profile image: %w", err)
	}
	if err := s.repo.SetProfileImage(ctx, id, newID); err != nil {
		if cleanup := s.files.Delete(ctx, newID); cleanup != nil {
			slog.Error("cleanup new profile image", "error", cleanup)
		}
		return nil, fmt.Errorf("set profile image association: %w", err)
	}
	old := src.ProfileImageID
	_ = src.SetProfileImage(newID)
	if old != "" {
		if err := s.files.Delete(ctx, old); err != nil {
			slog.Error("cleanup old profile image", "error", err)
		}
	}
	return src, nil
}

func (s *Service) RemoveProfileImage(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	s.imageMu.Lock()
	defer s.imageMu.Unlock()
	src, err := s.repo.GetSourceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get source for profile image: %w", err)
	}
	if err := src.ValidateProfileImageChange(); err != nil {
		return nil, err
	}
	old := src.ProfileImageID
	if old == "" {
		return src, nil
	}
	if err := s.repo.SetProfileImage(ctx, id, ""); err != nil {
		return nil, fmt.Errorf("remove profile image association: %w", err)
	}
	_ = src.SetProfileImage("")
	if s.files == nil {
		slog.Error("cleanup old profile image", "error", "file storage unavailable")
	} else if err := s.files.Delete(ctx, old); err != nil {
		slog.Error("cleanup old profile image", "error", err)
	}
	return src, nil
}

func (s *Service) GetProfileImage(ctx context.Context, id uuid.UUID) ([]byte, error) {
	src, err := s.repo.GetSourceByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get source for profile image: %w", err)
	}
	if src.ProfileImageID == "" {
		return nil, apperr.ResourceNotFound.New("profile image not found")
	}
	if s.files == nil {
		return nil, fmt.Errorf("file storage unavailable")
	}
	data, err := s.files.Read(ctx, src.ProfileImageID)
	if err != nil {
		return nil, fmt.Errorf("read profile image: %w", err)
	}
	return data, nil
}
