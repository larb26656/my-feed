package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sortoption "github.com/larb26656/my-feed/backend/internal/platform/sort"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/modules/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/entry/repository"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/cursor"
)

type Service struct {
	repo    repository.Repository
	sources sourceapi.EntryReader
}

func New(repo repository.Repository, sources sourceapi.EntryReader) *Service {
	return &Service{repo: repo, sources: sources}
}

// SaveLink records an Entry on the system manual "Saved links" Source.
// The client does not pass a Source id; the server resolves the singleton.
func (s *Service) SaveLink(ctx context.Context, in SaveLinkInput) (*SaveLinkResult, error) {
	src, err := s.sources.GetManual(ctx)
	if err != nil {
		return nil, err
	}
	result, err := s.upsertEntry(ctx, src.ID, entryMeta(in))
	if err != nil {
		return nil, err
	}
	return &SaveLinkResult{Entry: result.Entry, Created: result.Created}, nil
}

// IngestEntry records an Entry into an active fetchable Source.
// Rejects manual, missing, and deactivated Sources.
func (s *Service) IngestEntry(ctx context.Context, in IngestEntryInput) (*IngestEntryResult, error) {
	src, err := s.sources.Get(ctx, in.SourceID)
	if err != nil {
		return nil, err
	}
	if src.Type == sourceapi.SourceTypeManual {
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
	if link == "" {
		entry, err := domain.NewEntry(sourceID, meta)
		if err != nil {
			return nil, err
		}
		created, err := s.repo.CreateEntry(ctx, entry)
		if err != nil {
			return nil, err
		}
		return &IngestEntryResult{Entry: created, Created: true}, nil
	}
	existing, err := s.repo.GetEntryByLink(ctx, link)
	if err == nil {
		if existing.SourceID != sourceID {
			return &IngestEntryResult{Entry: existing, Created: false}, nil
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

	// Race: another writer inserted the same link first.
	existing, err = s.repo.GetEntryByLink(ctx, entry.Link)
	if err != nil {
		return nil, err
	}
	if existing.SourceID != sourceID {
		return &IngestEntryResult{Entry: existing, Created: false}, nil
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

func (s *Service) ListEntries(ctx context.Context, in ListEntriesInput) (*ListEntriesResult, error) {
	sort := &in.Sort
	if *sort == (repository.EntryListSort{}) {
		sort.PublishedAtSort = sortoption.Desc
	}
	for _, direction := range []sortoption.Direction{sort.CreatedAtSort, sort.PublishedAtSort, sort.SavedAtSort} {
		if direction != "" && !direction.Valid() {
			return nil, apperr.InvalidInput.New("invalid sort direction")
		}
	}
	pageSize := DefaultEntryPageSize
	if in.PageSize != nil && *in.PageSize >= 1 {
		pageSize = *in.PageSize
	}

	var after *repository.EntryListCursor
	if in.Cursor != nil {
		decoded, err := decodeEntryCursor(*in.Cursor, in.Criteria, in.Sort)
		if err != nil {
			return nil, err
		}
		after = decoded
	}

	rows, err := s.repo.ListEntries(ctx, after, in.Criteria, in.Sort, pageSize+1)
	if err != nil {
		return nil, err
	}

	result := &ListEntriesResult{Entries: rows}
	if len(rows) > pageSize {
		result.Entries = rows[:pageSize]
		last := result.Entries[pageSize-1]
		nextCursor, err := encodeEntryCursor(last, in.Criteria, in.Sort)
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

type entryCursorCriteria struct {
	repository.EntryListCriteria
	Sort repository.EntryListSort `json:"sort"`
}

type entryCursorPayload struct {
	Version  int                        `json:"version"`
	Position repository.EntryListCursor `json:"position"`
	Criteria entryCursorCriteria        `json:"criteria"`
}

func encodeEntryCursor(entry *domain.Entry, criteria repository.EntryListCriteria, sort repository.EntryListSort) (string, error) {
	return cursor.Encode(entryCursorPayload{Version: 1, Position: repository.EntryListCursor{CreatedAt: entry.CreatedAt, PublishedAt: entry.PublishedAt, SavedAt: entry.SavedAt, ID: entry.ID}, Criteria: entryCursorCriteria{EntryListCriteria: criteria, Sort: sort}})
}

func decodeEntryCursor(token string, criteria repository.EntryListCriteria, sort repository.EntryListSort) (*repository.EntryListCursor, error) {
	var payload entryCursorPayload
	if err := cursor.Decode(token, &payload); err != nil || payload.Version != 1 || payload.Position.CreatedAt.IsZero() || payload.Position.ID == uuid.Nil {
		return nil, apperr.InvalidInput.New("invalid cursor")
	}
	stored := payload.Criteria
	if stored.Sort != sort || (stored.SourceID == nil) != (criteria.SourceID == nil) || (stored.SourceID != nil && *stored.SourceID != *criteria.SourceID) || stored.Unread != criteria.Unread || stored.SavedOnly != criteria.SavedOnly {
		return nil, apperr.InvalidInput.New("cursor does not match criteria")
	}
	return &payload.Position, nil
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
