package service_test

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/quick-feed/backend/internal/feed/domain"
	"github.com/larb26656/quick-feed/backend/internal/feed/repository"
	"github.com/larb26656/quick-feed/backend/internal/feed/service"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

type fakeRepo struct {
	sources map[uuid.UUID]*domain.Source
	entries map[uuid.UUID]*domain.Entry
	byKey   map[string]uuid.UUID
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		sources: make(map[uuid.UUID]*domain.Source),
		entries: make(map[uuid.UUID]*domain.Entry),
		byKey:   make(map[string]uuid.UUID),
	}
}

func entryKey(sourceID uuid.UUID, link string) string {
	return sourceID.String() + "\x00" + link
}

func cloneEntry(e *domain.Entry) *domain.Entry {
	cp := *e
	if e.Thumbnail != nil {
		th := *e.Thumbnail
		cp.Thumbnail = &th
	}
	if e.PublishedAt != nil {
		p := *e.PublishedAt
		cp.PublishedAt = &p
	}
	if e.ReadAt != nil {
		r := *e.ReadAt
		cp.ReadAt = &r
	}
	if e.SavedAt != nil {
		s := *e.SavedAt
		cp.SavedAt = &s
	}
	return &cp
}

func (f *fakeRepo) CreateSource(_ context.Context, s *domain.Source) error {
	cp := *s
	f.sources[s.ID] = &cp
	return nil
}

func (f *fakeRepo) GetSourceByID(_ context.Context, id uuid.UUID) (*domain.Source, error) {
	s, ok := f.sources[id]
	if !ok {
		return nil, apperr.NotFound("source not found")
	}
	cp := *s
	return &cp, nil
}

func (f *fakeRepo) CreateEntry(_ context.Context, e *domain.Entry) (*domain.Entry, error) {
	key := entryKey(e.SourceID, e.Link)
	if _, ok := f.byKey[key]; ok {
		return nil, apperr.Conflict("entry already exists for source and link")
	}
	stored := cloneEntry(e)
	f.entries[e.ID] = stored
	f.byKey[key] = e.ID
	return cloneEntry(stored), nil
}

func (f *fakeRepo) GetEntryByID(_ context.Context, id uuid.UUID) (*domain.Entry, error) {
	e, ok := f.entries[id]
	if !ok {
		return nil, apperr.NotFound("entry not found")
	}
	return cloneEntry(e), nil
}

func (f *fakeRepo) GetEntryBySourceAndLink(_ context.Context, sourceID uuid.UUID, link string) (*domain.Entry, error) {
	id, ok := f.byKey[entryKey(sourceID, link)]
	if !ok {
		return nil, apperr.NotFound("entry not found")
	}
	return cloneEntry(f.entries[id]), nil
}

func (f *fakeRepo) UpdateEntry(_ context.Context, e *domain.Entry) (*domain.Entry, error) {
	if _, ok := f.entries[e.ID]; !ok {
		return nil, apperr.NotFound("entry not found")
	}
	stored := cloneEntry(e)
	f.entries[e.ID] = stored
	return cloneEntry(stored), nil
}

func (f *fakeRepo) WithTx(_ pgx.Tx) repository.Repository {
	return f
}

func TestCreateSourceMakesItRetrievable(t *testing.T) {
	svc := service.New(newFakeRepo())

	created, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, "HN RSS", created.Name)

	got, err := svc.GetSource(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, "HN RSS", got.Name)
}

func TestGetSourceMissingIDNotFound(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.GetSource(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestRecordEntryMakesItRetrievable(t *testing.T) {
	svc := service.New(newFakeRepo())
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	thumb := "https://example.com/thumb.png"
	published := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	recorded, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID:    src.ID,
		Title:       "Show HN: Quick Feed",
		Link:        "https://example.com/a",
		Description: "A personal reader",
		Thumbnail:   &thumb,
		PublishedAt: &published,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, recorded.ID)
	require.Equal(t, src.ID, recorded.SourceID)
	require.Equal(t, "Show HN: Quick Feed", recorded.Title)
	require.Equal(t, "https://example.com/a", recorded.Link)
	require.Equal(t, "A personal reader", recorded.Description)
	require.NotNil(t, recorded.Thumbnail)
	require.Equal(t, thumb, *recorded.Thumbnail)
	require.NotNil(t, recorded.PublishedAt)
	require.True(t, published.Equal(recorded.PublishedAt.UTC()))

	got, err := svc.GetEntry(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.Equal(t, recorded.ID, got.ID)
	require.Equal(t, "Show HN: Quick Feed", got.Title)
	require.Equal(t, "https://example.com/a", got.Link)
}

func TestGetEntryMissingIDNotFound(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.GetEntry(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestRecordEntryBlankTitleValidation(t *testing.T) {
	svc := service.New(newFakeRepo())
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	_, err = svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID: src.ID,
		Title:    "   ",
		Link:     "https://example.com/a",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestRecordEntryBlankLinkValidation(t *testing.T) {
	svc := service.New(newFakeRepo())
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	_, err = svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID: src.ID,
		Title:    "Show HN",
		Link:     "",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestRecordEntryMissingSourceRejected(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID: uuid.New(),
		Title:    "Show HN",
		Link:     "https://example.com/a",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestRecordEntrySameSourceAndLinkUpsertsMetadata(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo)
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	first, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID:    src.ID,
		Title:       "Original title",
		Link:        "https://example.com/a",
		Description: "original",
	})
	require.NoError(t, err)
	createdAt := first.CreatedAt
	updatedAt := first.UpdatedAt

	time.Sleep(2 * time.Millisecond)

	thumb := "https://example.com/new.png"
	published := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	second, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID:    src.ID,
		Title:       "Updated title",
		Link:        "https://example.com/a",
		Description: "refreshed",
		Thumbnail:   &thumb,
		PublishedAt: &published,
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, "Updated title", second.Title)
	require.Equal(t, "refreshed", second.Description)
	require.NotNil(t, second.Thumbnail)
	require.Equal(t, thumb, *second.Thumbnail)
	require.NotNil(t, second.PublishedAt)
	require.True(t, published.Equal(second.PublishedAt.UTC()))
	require.Equal(t, createdAt, second.CreatedAt)
	require.True(t, second.UpdatedAt.After(updatedAt))

	got, err := svc.GetEntry(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, "Updated title", got.Title)

	require.Len(t, repo.entries, 1)
}

func (f *fakeRepo) seedEntry(e *domain.Entry) {
	stored := cloneEntry(e)
	f.entries[e.ID] = stored
	f.byKey[entryKey(e.SourceID, e.Link)] = e.ID
}

func uuidLess(a, b uuid.UUID) bool {
	return bytes.Compare(a[:], b[:]) < 0
}

func (f *fakeRepo) ListEntries(_ context.Context, after *repository.EntryListCursor, limit int) ([]*domain.Entry, error) {
	type ranked struct {
		entry  *domain.Entry
		sortAt time.Time
	}
	rankedEntries := make([]ranked, 0, len(f.entries))
	for _, e := range f.entries {
		rankedEntries = append(rankedEntries, ranked{entry: e, sortAt: e.SortAt()})
	}
	sort.Slice(rankedEntries, func(i, j int) bool {
		if rankedEntries[i].sortAt.Equal(rankedEntries[j].sortAt) {
			// Newest-first: larger UUID first — match Postgres uuid byte order, not String().
			return bytes.Compare(rankedEntries[i].entry.ID[:], rankedEntries[j].entry.ID[:]) > 0
		}
		return rankedEntries[i].sortAt.After(rankedEntries[j].sortAt)
	})
	out := make([]*domain.Entry, 0, limit)
	for _, item := range rankedEntries {
		if after != nil {
			if item.sortAt.After(after.SortAt) {
				continue
			}
			if item.sortAt.Equal(after.SortAt) && !uuidLess(item.entry.ID, after.ID) {
				continue
			}
		}
		out = append(out, cloneEntry(item.entry))
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func TestListEntriesNewestFirstPublishedAtFallbackCreatedAtIDTieBreak(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo)
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	pubSame := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	pubNewest := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	createdFallback := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// idHexA (...000a) sorts AFTER idHex10 (...0010) in Postgres uuid byte order,
	// but BEFORE it under ID.String() lexicographic order — catch that divergence.
	idHexA := uuid.MustParse("00000000-0000-4000-8000-00000000000a")
	idHex10 := uuid.MustParse("00000000-0000-4000-8000-000000000010")
	idNewest := uuid.MustParse("00000000-0000-4000-8000-0000000000aa")
	idFallback := uuid.MustParse("00000000-0000-4000-8000-0000000000bb")

	repo.seedEntry(&domain.Entry{
		ID: idHexA, SourceID: src.ID, Title: "same-a", Link: "https://example.com/same-a",
		PublishedAt: &pubSame, CreatedAt: pubSame, UpdatedAt: pubSame,
	})
	repo.seedEntry(&domain.Entry{
		ID: idHex10, SourceID: src.ID, Title: "same-10", Link: "https://example.com/same-10",
		PublishedAt: &pubSame, CreatedAt: pubSame, UpdatedAt: pubSame,
	})
	repo.seedEntry(&domain.Entry{
		ID: idNewest, SourceID: src.ID, Title: "newest", Link: "https://example.com/newest",
		PublishedAt: &pubNewest, CreatedAt: pubNewest, UpdatedAt: pubNewest,
	})
	repo.seedEntry(&domain.Entry{
		ID: idFallback, SourceID: src.ID, Title: "no-pub", Link: "https://example.com/no-pub",
		CreatedAt: createdFallback, UpdatedAt: createdFallback,
	})

	pageSize := 10
	page, err := svc.ListEntries(context.Background(), service.ListEntriesInput{PageSize: &pageSize})
	require.NoError(t, err)
	require.Len(t, page.Entries, 4)
	require.Equal(t, idNewest, page.Entries[0].ID)
	require.Equal(t, idHex10, page.Entries[1].ID)
	require.Equal(t, idHexA, page.Entries[2].ID)
	require.Equal(t, idFallback, page.Entries[3].ID)
	require.Nil(t, page.NextCursor)
}

func seedTimedEntry(repo *fakeRepo, sourceID, id uuid.UUID, link string, publishedAt *time.Time, createdAt time.Time) {
	repo.seedEntry(&domain.Entry{
		ID:          id,
		SourceID:    sourceID,
		Title:       link,
		Link:        link,
		PublishedAt: publishedAt,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	})
}

func TestListEntriesCursorContinuesWithoutOffsetDrift(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo)
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	t4 := time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC)

	id1 := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	id2 := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	id3 := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	id4 := uuid.MustParse("00000000-0000-4000-8000-000000000004")

	seedTimedEntry(repo, src.ID, id1, "https://example.com/1", &t1, t1)
	seedTimedEntry(repo, src.ID, id2, "https://example.com/2", &t2, t2)
	seedTimedEntry(repo, src.ID, id3, "https://example.com/3", &t3, t3)
	seedTimedEntry(repo, src.ID, id4, "https://example.com/4", &t4, t4)

	pageSize := 2
	first, err := svc.ListEntries(context.Background(), service.ListEntriesInput{PageSize: &pageSize})
	require.NoError(t, err)
	require.Len(t, first.Entries, 2)
	require.Equal(t, id4, first.Entries[0].ID)
	require.Equal(t, id3, first.Entries[1].ID)
	require.NotNil(t, first.NextCursor)

	// Newer entry arrives after first page was fetched — cursor must not drift like offset paging.
	t5 := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	id5 := uuid.MustParse("00000000-0000-4000-8000-000000000005")
	seedTimedEntry(repo, src.ID, id5, "https://example.com/5", &t5, t5)

	second, err := svc.ListEntries(context.Background(), service.ListEntriesInput{
		Cursor:   first.NextCursor,
		PageSize: &pageSize,
	})
	require.NoError(t, err)
	require.Len(t, second.Entries, 2)
	require.Equal(t, id2, second.Entries[0].ID)
	require.Equal(t, id1, second.Entries[1].ID)
	require.Nil(t, second.NextCursor)
}

func TestListEntriesEndOfListHasNoNextCursor(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo)
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	t1 := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	seedTimedEntry(repo, src.ID, uuid.MustParse("00000000-0000-4000-8000-000000000011"), "https://example.com/a", &t1, t1)

	pageSize := 5
	page, err := svc.ListEntries(context.Background(), service.ListEntriesInput{PageSize: &pageSize})
	require.NoError(t, err)
	require.Len(t, page.Entries, 1)
	require.Nil(t, page.NextCursor)
}

func TestListEntriesInvalidCursorFailsClearly(t *testing.T) {
	svc := service.New(newFakeRepo())
	bad := "not-a-valid-cursor"
	_, err := svc.ListEntries(context.Background(), service.ListEntriesInput{Cursor: &bad})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	empty := ""
	_, err = svc.ListEntries(context.Background(), service.ListEntriesInput{Cursor: &empty})
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestListEntriesOmittingPageSizeUsesDefault(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo)
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	base := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	total := service.DefaultEntryPageSize + 3
	for i := 0; i < total; i++ {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", i+1))
		pub := base.Add(time.Duration(i) * time.Hour)
		seedTimedEntry(repo, src.ID, id, fmt.Sprintf("https://example.com/%d", i), &pub, pub)
	}

	page, err := svc.ListEntries(context.Background(), service.ListEntriesInput{})
	require.NoError(t, err)
	require.Len(t, page.Entries, service.DefaultEntryPageSize)
	require.NotNil(t, page.NextCursor)
}

func TestListEntriesInvalidSmallPageSizeUsesDefault(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo)
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	base := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	total := service.DefaultEntryPageSize + 2
	for i := 0; i < total; i++ {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", i+1))
		pub := base.Add(time.Duration(i) * time.Hour)
		seedTimedEntry(repo, src.ID, id, fmt.Sprintf("https://example.com/%d", i), &pub, pub)
	}

	for _, size := range []int{0, -1} {
		pageSize := size
		page, err := svc.ListEntries(context.Background(), service.ListEntriesInput{PageSize: &pageSize})
		require.NoError(t, err)
		require.Len(t, page.Entries, service.DefaultEntryPageSize)
		require.NotNil(t, page.NextCursor)
	}
}

func TestMarkEntryReadSetsReadAt(t *testing.T) {
	svc := service.New(newFakeRepo())
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	recorded, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID: src.ID,
		Title:    "Show HN",
		Link:     "https://example.com/a",
	})
	require.NoError(t, err)
	require.Nil(t, recorded.ReadAt)

	before := time.Now().UTC()
	marked, err := svc.MarkEntryRead(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.NotNil(t, marked.ReadAt)
	require.False(t, marked.ReadAt.Before(before))
	require.Nil(t, marked.SavedAt)

	got, err := svc.GetEntry(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.NotNil(t, got.ReadAt)
	require.True(t, marked.ReadAt.Equal(got.ReadAt.UTC()))
}

func TestMarkEntrySavedSetsSavedAt(t *testing.T) {
	svc := service.New(newFakeRepo())
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	recorded, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID: src.ID,
		Title:    "Show HN",
		Link:     "https://example.com/a",
	})
	require.NoError(t, err)
	require.Nil(t, recorded.SavedAt)

	before := time.Now().UTC()
	marked, err := svc.MarkEntrySaved(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.NotNil(t, marked.SavedAt)
	require.False(t, marked.SavedAt.Before(before))
	require.Nil(t, marked.ReadAt)

	got, err := svc.GetEntry(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.NotNil(t, got.SavedAt)
	require.True(t, marked.SavedAt.Equal(got.SavedAt.UTC()))
}

func TestMarkEntrySavedWithoutReadLeavesReadAtNil(t *testing.T) {
	svc := service.New(newFakeRepo())
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	recorded, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID: src.ID,
		Title:    "Show HN",
		Link:     "https://example.com/a",
	})
	require.NoError(t, err)

	marked, err := svc.MarkEntrySaved(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.NotNil(t, marked.SavedAt)
	require.Nil(t, marked.ReadAt)
}

func TestMarkEntryReadWithoutSaveLeavesSavedAtNil(t *testing.T) {
	svc := service.New(newFakeRepo())
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	recorded, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID: src.ID,
		Title:    "Show HN",
		Link:     "https://example.com/a",
	})
	require.NoError(t, err)

	marked, err := svc.MarkEntryRead(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.NotNil(t, marked.ReadAt)
	require.Nil(t, marked.SavedAt)
}

func TestMarkEntryReadUnknownIDNotFound(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.MarkEntryRead(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestMarkEntrySavedUnknownIDNotFound(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.MarkEntrySaved(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestClearEntryReadClearsReadAt(t *testing.T) {
	svc := service.New(newFakeRepo())
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	recorded, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID: src.ID,
		Title:    "Show HN",
		Link:     "https://example.com/a",
	})
	require.NoError(t, err)

	marked, err := svc.MarkEntryRead(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.NotNil(t, marked.ReadAt)
	updatedAt := marked.UpdatedAt

	time.Sleep(2 * time.Millisecond)
	cleared, err := svc.ClearEntryRead(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.Nil(t, cleared.ReadAt)
	require.True(t, cleared.UpdatedAt.After(updatedAt))
}

func TestClearEntrySavedClearsSavedAt(t *testing.T) {
	svc := service.New(newFakeRepo())
	src, err := svc.CreateSource(context.Background(), service.CreateSourceInput{Name: "HN RSS"})
	require.NoError(t, err)

	recorded, err := svc.RecordEntry(context.Background(), service.RecordEntryInput{
		SourceID: src.ID,
		Title:    "Show HN",
		Link:     "https://example.com/a",
	})
	require.NoError(t, err)

	marked, err := svc.MarkEntrySaved(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.NotNil(t, marked.SavedAt)
	updatedAt := marked.UpdatedAt

	time.Sleep(2 * time.Millisecond)
	cleared, err := svc.ClearEntrySaved(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.Nil(t, cleared.SavedAt)
	require.True(t, cleared.UpdatedAt.After(updatedAt))
}

func TestClearEntryReadUnknownIDNotFound(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.ClearEntryRead(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}
