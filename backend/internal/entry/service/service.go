package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	entryapi "github.com/larb26656/my-feed/backend/internal/entry/api"
	"github.com/larb26656/my-feed/backend/internal/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/entry/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/cursor"
)

// SourceType is the Entry module's view of a Source's type (not imported from source/domain).
type SourceType string

const (
	SourceTypeRSS     SourceType = "rss"
	SourceTypeYouTube SourceType = "youtube"
	SourceTypeManual  SourceType = "manual"
)

// SourceRef is the Entry module's lookup view of a Source for write-path rules.
type SourceRef struct {
	ID     uuid.UUID
	Type   SourceType
	Active bool
}

// Sources is the consumer-defined port Entry uses for Source lookups.
// Missing Source must surface as apperr.ResourceNotFound. Entry must not import source SQLite.
type Sources interface {
	Get(ctx context.Context, id uuid.UUID) (*SourceRef, error)
	GetManual(ctx context.Context) (*SourceRef, error)
}

type Service struct {
	repo    repository.Repository
	sources Sources
}

func New(repo repository.Repository, sources Sources) *Service {
	return &Service{repo: repo, sources: sources}
}

type SaveLinkInput struct {
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

// SaveLink records an Entry on the system manual "Saved links" Source.
// The client does not pass a Source id; the server resolves the singleton.
func (s *Service) SaveLink(ctx context.Context, in SaveLinkInput) (*domain.Entry, error) {
	src, err := s.sources.GetManual(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.upsertEntry(ctx, src.ID, entryMeta{
		Title:       in.Title,
		Link:        in.Link,
		Description: in.Description,
		Thumbnail:   in.Thumbnail,
		PublishedAt: in.PublishedAt,
	})
	if err != nil {
		return nil, err
	}
	return result.Entry, nil
}

type IngestEntryInput = entryapi.IngestEntryInput

// IngestEntryResult is the outcome of recording an Entry through ingest.
// Created is true only when a new Entry was written; metadata refresh is false.
type IngestEntryResult struct {
	Entry   *domain.Entry
	Created bool
}

// IngestEntry records an Entry into an active fetchable Source.
// Rejects manual, missing, and deactivated Sources.
func (s *Service) IngestEntry(ctx context.Context, in IngestEntryInput) (*IngestEntryResult, error) {
	src, err := s.sources.Get(ctx, in.SourceID)
	if err != nil {
		return nil, err
	}
	if src.Type != SourceTypeRSS && src.Type != SourceTypeYouTube {
		return nil, apperr.InvalidInput.New("ingest can only record Entries into fetchable Sources")
	}
	if !src.Active {
		return nil, domain.ErrSourceDeactivated.New("cannot ingest into a deactivated source")
	}
	return s.upsertEntry(ctx, src.ID, entryMeta{
		Title:       in.Title,
		Link:        in.Link,
		Description: in.Description,
		Thumbnail:   in.Thumbnail,
		PublishedAt: in.PublishedAt,
	})
}

type entryMeta struct {
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

func (s *Service) upsertEntry(ctx context.Context, sourceID uuid.UUID, in entryMeta) (*IngestEntryResult, error) {
	meta := domain.EntryMetadata{
		Title:       in.Title,
		Link:        in.Link,
		Description: in.Description,
		Thumbnail:   in.Thumbnail,
		PublishedAt: in.PublishedAt,
	}

	link := strings.TrimSpace(in.Link)
	existing, err := s.repo.GetEntryBySourceAndLink(ctx, sourceID, link)
	if err == nil {
		if err := existing.RefreshMetadata(meta); err != nil {
			return nil, err
		}
		updated, err := s.repo.UpdateEntry(ctx, existing)
		if err != nil {
			return nil, err
		}
		return &IngestEntryResult{Entry: updated, Created: false}, nil
	}
	if !isNotFound(err) {
		return nil, err
	}

	entry, err := domain.NewEntry(sourceID, meta)
	if err != nil {
		return nil, err
	}
	created, err := s.repo.CreateEntry(ctx, entry)
	if err == nil {
		return &IngestEntryResult{Entry: created, Created: true}, nil
	}
	if !isConflict(err) {
		return nil, err
	}

	// Race: another writer inserted the same Source+link first.
	existing, err = s.repo.GetEntryBySourceAndLink(ctx, sourceID, entry.Link)
	if err != nil {
		return nil, err
	}
	if err := existing.RefreshMetadata(meta); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateEntry(ctx, existing)
	if err != nil {
		return nil, err
	}
	return &IngestEntryResult{Entry: updated, Created: false}, nil
}

func (s *Service) GetEntry(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
	return s.repo.GetEntryByID(ctx, id)
}

const DefaultEntryPageSize = 20

type ListEntriesInput struct {
	Cursor   *string
	PageSize *int
	Criteria repository.EntryListCriteria
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
		decoded, err := decodeEntryCursor(*in.Cursor, in.Criteria)
		if err != nil {
			return nil, err
		}
		after = decoded
	}

	rows, err := s.repo.ListEntries(ctx, after, in.Criteria, pageSize+1)
	if err != nil {
		return nil, err
	}

	result := &ListEntriesResult{Entries: rows}
	if len(rows) > pageSize {
		result.Entries = rows[:pageSize]
		last := result.Entries[pageSize-1]
		nextCursor, err := encodeEntryCursor(last.SortAt(), last.ID, in.Criteria)
		if err != nil {
			return nil, fmt.Errorf("encode Entry cursor: %w", err)
		}
		result.NextCursor = &nextCursor
	}
	return result, nil
}

func (s *Service) CountSavedEntries(ctx context.Context) (int, error) {
	return s.repo.CountSavedEntries(ctx)
}

type entryCursorPayload struct {
	SortAt    time.Time  `json:"sortAt"`
	ID        uuid.UUID  `json:"id"`
	SourceID  *uuid.UUID `json:"sourceId"`
	Unread    bool       `json:"unread"`
	SavedOnly bool       `json:"savedOnly"`
}

func encodeEntryCursor(sortAt time.Time, id uuid.UUID, criteria repository.EntryListCriteria) (string, error) {
	return cursor.Encode(entryCursorPayload{SortAt: sortAt.UTC(), ID: id, SourceID: criteria.SourceID, Unread: criteria.Unread, SavedOnly: criteria.SavedOnly})
}

func decodeEntryCursor(token string, criteria repository.EntryListCriteria) (*repository.EntryListCursor, error) {
	var payload entryCursorPayload
	if err := cursor.Decode(token, &payload); err != nil || payload.SortAt.IsZero() || payload.ID == uuid.Nil {
		return nil, apperr.InvalidInput.New("invalid cursor")
	}
	if (payload.SourceID == nil) != (criteria.SourceID == nil) || (payload.SourceID != nil && *payload.SourceID != *criteria.SourceID) || payload.Unread != criteria.Unread || payload.SavedOnly != criteria.SavedOnly {
		return nil, apperr.InvalidInput.New("cursor does not match criteria")
	}
	return &repository.EntryListCursor{SortAt: payload.SortAt.UTC(), ID: payload.ID}, nil
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
	var ae *apperr.AppError
	return errors.As(err, &ae) && ae.Kind == apperr.KindNotFound
}

func isConflict(err error) bool {
	var ae *apperr.AppError
	return errors.As(err, &ae) && ae.Code == apperr.ResourceConflict.Code
}
