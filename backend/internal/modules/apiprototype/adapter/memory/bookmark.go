// Package memory is a temporary, in-memory repository for the prototype.
package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/apiprototype/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/apiprototype/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type Repository struct {
	mu    sync.Mutex
	items []domain.Bookmark
}

func New() *Repository { return &Repository{} }

func (r *Repository) Create(_ context.Context, item domain.Bookmark) (domain.Bookmark, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.items {
		if existing.URL == item.URL {
			return domain.Bookmark{}, domain.ErrBookmarkConflict.New("bookmark URL already exists")
		}
	}
	item.ID = uuid.NewString()
	r.items = append(r.items, item)
	return item, nil
}

func (r *Repository) Get(_ context.Context, id string) (domain.Bookmark, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.Bookmark{}, apperr.ResourceNotFound.New("bookmark not found")
}

// List applies filters and ordering before seeking past the cursor.
func (r *Repository) List(_ context.Context, input repository.ListQuery) ([]domain.Bookmark, string, error) {
	r.mu.Lock()
	items := append([]domain.Bookmark{}, r.items...)
	r.mu.Unlock()
	filtered := items[:0]
	for _, item := range items {
		if !strings.Contains(strings.ToLower(item.Title), strings.ToLower(input.TitleLike)) ||
			(input.StatusEqual == "read" && !item.Read) || (input.StatusEqual == "unread" && item.Read) {
			continue
		}
		filtered = append(filtered, item)
	}
	items = filtered
	if input.SortBy == "title" {
		sort.SliceStable(items, func(i, j int) bool {
			if input.SortOrder == "desc" {
				return items[i].Title > items[j].Title
			}
			return items[i].Title < items[j].Title
		})
	} else if input.SortOrder == "desc" {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}
	start := 0
	if input.Cursor != "" {
		found := false
		for i, item := range items {
			if item.ID == input.Cursor {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return nil, "", repository.ErrInvalidCursor
		}
	}
	end := start + input.PageSize
	if end >= len(items) {
		return items[start:], "", nil
	}
	return items[start:end], items[end-1].ID, nil
}

func (r *Repository) MarkRead(_ context.Context, id string) (domain.Bookmark, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.items {
		if r.items[i].ID == id {
			r.items[i].Read = true
			return r.items[i], nil
		}
	}
	return domain.Bookmark{}, apperr.ResourceNotFound.New("bookmark not found")
}
