package service

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/quick-feed/backend/internal/feed/domain"
	"github.com/larb26656/quick-feed/backend/internal/feed/repository"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

type Service struct {
	repo repository.Repository
}

func New(repo repository.Repository) *Service {
	return &Service{repo: repo}
}

type CreateSourceInput struct {
	Name string
}

func (s *Service) CreateSource(ctx context.Context, in CreateSourceInput) (*domain.Source, error) {
	src, err := domain.NewSource(in.Name)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateSource(ctx, src); err != nil {
		return nil, err
	}
	return src, nil
}

func (s *Service) GetSource(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	return s.repo.GetSourceByID(ctx, id)
}

type RecordEntryInput struct {
	SourceID    uuid.UUID
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

func (s *Service) RecordEntry(ctx context.Context, in RecordEntryInput) (*domain.Entry, error) {
	if _, err := s.repo.GetSourceByID(ctx, in.SourceID); err != nil {
		return nil, err
	}

	meta := domain.EntryMetadata{
		Title:       in.Title,
		Link:        in.Link,
		Description: in.Description,
		Thumbnail:   in.Thumbnail,
		PublishedAt: in.PublishedAt,
	}

	link := strings.TrimSpace(in.Link)
	existing, err := s.repo.GetEntryBySourceAndLink(ctx, in.SourceID, link)
	if err == nil {
		if err := existing.RefreshMetadata(meta); err != nil {
			return nil, err
		}
		return s.repo.UpdateEntry(ctx, existing)
	}
	if !isNotFound(err) {
		return nil, err
	}

	entry, err := domain.NewEntry(in.SourceID, meta)
	if err != nil {
		return nil, err
	}
	created, err := s.repo.CreateEntry(ctx, entry)
	if err == nil {
		return created, nil
	}
	if !isConflict(err) {
		return nil, err
	}

	// Race: another writer inserted the same Source+link first.
	existing, err = s.repo.GetEntryBySourceAndLink(ctx, in.SourceID, entry.Link)
	if err != nil {
		return nil, err
	}
	if err := existing.RefreshMetadata(meta); err != nil {
		return nil, err
	}
	return s.repo.UpdateEntry(ctx, existing)
}

func (s *Service) GetEntry(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
	return s.repo.GetEntryByID(ctx, id)
}

const DefaultEntryPageSize = 20

type ListEntriesInput struct {
	Cursor   *string
	PageSize *int
}

type ListEntriesResult struct {
	Entries    []*domain.Entry
	NextCursor *string
}

func (s *Service) ListEntries(ctx context.Context, in ListEntriesInput) (*ListEntriesResult, error) {
	pageSize := DefaultEntryPageSize
	if in.PageSize != nil && *in.PageSize >= 1 {
		pageSize = *in.PageSize
	}

	var after *repository.EntryListCursor
	if in.Cursor != nil {
		decoded, err := decodeEntryCursor(*in.Cursor)
		if err != nil {
			return nil, err
		}
		after = decoded
	}

	rows, err := s.repo.ListEntries(ctx, after, pageSize+1)
	if err != nil {
		return nil, err
	}

	result := &ListEntriesResult{Entries: rows}
	if len(rows) > pageSize {
		result.Entries = rows[:pageSize]
		last := result.Entries[pageSize-1]
		cursor := encodeEntryCursor(last.SortAt(), last.ID)
		result.NextCursor = &cursor
	}
	return result, nil
}

func encodeEntryCursor(sortAt time.Time, id uuid.UUID) string {
	raw := sortAt.UTC().Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeEntryCursor(cursor string) (*repository.EntryListCursor, error) {
	if strings.TrimSpace(cursor) == "" {
		return nil, apperr.Validation("invalid cursor")
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, apperr.Validation("invalid cursor")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, apperr.Validation("invalid cursor")
	}
	sortAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, apperr.Validation("invalid cursor")
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return nil, apperr.Validation("invalid cursor")
	}
	return &repository.EntryListCursor{SortAt: sortAt.UTC(), ID: id}, nil
}

func (s *Service) MarkEntryRead(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
	return s.updateEntry(ctx, id, func(e *domain.Entry) {
		e.MarkRead(time.Now().UTC())
	})
}

func (s *Service) ClearEntryRead(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
	return s.updateEntry(ctx, id, (*domain.Entry).ClearRead)
}

func (s *Service) MarkEntrySaved(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
	return s.updateEntry(ctx, id, func(e *domain.Entry) {
		e.MarkSaved(time.Now().UTC())
	})
}

func (s *Service) ClearEntrySaved(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
	return s.updateEntry(ctx, id, (*domain.Entry).ClearSaved)
}

func (s *Service) updateEntry(ctx context.Context, id uuid.UUID, mutate func(*domain.Entry)) (*domain.Entry, error) {
	entry, err := s.repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	mutate(entry)
	return s.repo.UpdateEntry(ctx, entry)
}

func isNotFound(err error) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == apperr.CodeNotFound
}

func isConflict(err error) bool {
	var ae *apperr.Error
	return errors.As(err, &ae) && ae.Code == apperr.CodeConflict
}
