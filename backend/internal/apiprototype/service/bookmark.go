package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/larb26656/my-feed/backend/internal/apiprototype/domain"
	"github.com/larb26656/my-feed/backend/internal/apiprototype/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

var ErrInvalid = apperr.InvalidInput.New("invalid input")

type Service struct{ repo repository.Repository }

func New(repo repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) Create(ctx context.Context, title, link string) (domain.Bookmark, error) {
	title, link = strings.TrimSpace(title), strings.TrimSpace(link)
	u, err := url.Parse(link)
	if title == "" || err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return domain.Bookmark{}, ErrInvalid
	}
	return s.repo.Create(ctx, domain.Bookmark{Title: title, URL: link})
}
func (s *Service) Get(ctx context.Context, id string) (domain.Bookmark, error) {
	return s.repo.Get(ctx, id)
}
func (s *Service) MarkRead(ctx context.Context, id string) (domain.Bookmark, error) {
	return s.repo.MarkRead(ctx, id)
}

type ListInput struct {
	Cursor, TitleLike, StatusEqual, SortBy, SortOrder string
	PageSize                                          int
}

// List uses the last item's ID as a cursor. This simple approach is only for the prototype.
func (s *Service) List(ctx context.Context, input ListInput) ([]domain.Bookmark, string, error) {
	if input.PageSize < 1 || input.PageSize > 100 ||
		(input.StatusEqual != "" && input.StatusEqual != "read" && input.StatusEqual != "unread") ||
		(input.SortBy != "" && input.SortBy != "createdAt" && input.SortBy != "title") ||
		(input.SortOrder != "" && input.SortOrder != "asc" && input.SortOrder != "desc") {
		return nil, "", ErrInvalid
	}
	items, cursor, err := s.repo.List(ctx, repository.ListQuery{
		Cursor: input.Cursor, PageSize: input.PageSize,
		TitleLike: input.TitleLike, StatusEqual: input.StatusEqual,
		SortBy: input.SortBy, SortOrder: input.SortOrder,
	})
	if errors.Is(err, repository.ErrInvalidCursor) {
		return nil, "", ErrInvalid
	}
	return items, cursor, err
}
