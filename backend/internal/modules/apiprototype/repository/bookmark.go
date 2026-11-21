package repository

import (
	"context"
	"errors"

	"github.com/larb26656/my-feed/backend/internal/modules/apiprototype/domain"
)

var ErrInvalidCursor = errors.New("invalid cursor")

type ListQuery struct {
	Cursor, TitleLike, StatusEqual, SortBy, SortOrder string
	PageSize                                          int
}

type Repository interface {
	Create(context.Context, domain.Bookmark) (domain.Bookmark, error)
	Get(context.Context, string) (domain.Bookmark, error)
	List(context.Context, ListQuery) ([]domain.Bookmark, string, error)
	MarkRead(context.Context, string) (domain.Bookmark, error)
}
