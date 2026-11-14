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

	"github.com/larb26656/quick-feed/backend/internal/entry/domain"
	"github.com/larb26656/quick-feed/backend/internal/entry/repository"
	"github.com/larb26656/quick-feed/backend/internal/entry/service"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

type fakeRepo struct {
	entries map[uuid.UUID]*domain.Entry
	byKey   map[string]uuid.UUID
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		entries: make(map[uuid.UUID]*domain.Entry),
		byKey:   make(map[string]uuid.UUID),
	}
}

type fakeSource struct {
	id     uuid.UUID
	typ    service.SourceType
	active bool
}

type fakeSources struct {
	byID   map[uuid.UUID]fakeSource
	manual *fakeSource
}

func newFakeSources() *fakeSources {
	return &fakeSources{byID: make(map[uuid.UUID]fakeSource)}
}

func (f *fakeSources) seed(id uuid.UUID, typ service.SourceType, active bool) {
	src := fakeSource{id: id, typ: typ, active: active}
	f.byID[id] = src
	if typ == service.SourceTypeManual {
		cp := src
		f.manual = &cp
	}
}

func (f *fakeSources) Get(_ context.Context, id uuid.UUID) (*service.SourceRef, error) {
	src, ok := f.byID[id]
	if !ok {
		return nil, apperr.NotFound("source not found")
	}
	return &service.SourceRef{ID: src.id, Type: src.typ, Active: src.active}, nil
}

func (f *fakeSources) GetManual(_ context.Context) (*service.SourceRef, error) {
	if f.manual == nil {
		return nil, apperr.NotFound("source not found")
	}
	return &service.SourceRef{ID: f.manual.id, Type: f.manual.typ, Active: f.manual.active}, nil
}

func newService() (*service.Service, *fakeRepo, *fakeSources) {
	repo := newFakeRepo()
	sources := newFakeSources()
	return service.New(repo, sources), repo, sources
}

func seedRSSSource(sources *fakeSources) uuid.UUID {
	id := uuid.New()
	sources.seed(id, service.SourceTypeRSS, true)
	return id
}

func seedManualSource(sources *fakeSources) uuid.UUID {
	id := uuid.New()
	sources.seed(id, service.SourceTypeManual, true)
	return id
}

// seedSource seeds an active RSS Source — the ingest write path default for legacy tests.
func seedSource(sources *fakeSources) uuid.UUID {
	return seedRSSSource(sources)
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

func TestSaveLinkRecordsOnManualWithoutClientSourceID(t *testing.T) {
	svc, _, sources := newService()
	manualID := seedManualSource(sources)
	_ = seedRSSSource(sources) // other Sources must not be chosen

	recorded, err := svc.SaveLink(context.Background(), service.SaveLinkInput{
		Title: "Interesting article",
		Link:  "https://example.com/saved",
	})
	require.NoError(t, err)
	require.Equal(t, manualID, recorded.SourceID)
	require.Equal(t, "Interesting article", recorded.Title)
	require.Equal(t, "https://example.com/saved", recorded.Link)

	got, err := svc.GetEntry(context.Background(), recorded.ID)
	require.NoError(t, err)
	require.Equal(t, manualID, got.SourceID)
}

func TestSaveLinkSameLinkUpsertsMetadata(t *testing.T) {
	svc, repo, sources := newService()
	manualID := seedManualSource(sources)

	first, err := svc.SaveLink(context.Background(), service.SaveLinkInput{
		Title:       "Original",
		Link:        "https://example.com/saved",
		Description: "first",
	})
	require.NoError(t, err)

	time.Sleep(2 * time.Millisecond)
	second, err := svc.SaveLink(context.Background(), service.SaveLinkInput{
		Title:       "Updated",
		Link:        "https://example.com/saved",
		Description: "second",
	})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, manualID, second.SourceID)
	require.Equal(t, "Updated", second.Title)
	require.Equal(t, "second", second.Description)
	require.Len(t, repo.entries, 1)
}

func TestIngestEntryIntoActiveRSSWorks(t *testing.T) {
	svc, _, sources := newService()
	rssID := seedRSSSource(sources)

	recorded, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: rssID,
		Title:    "From feed",
		Link:     "https://example.com/feed-item",
	})
	require.NoError(t, err)
	require.Equal(t, rssID, recorded.SourceID)
	require.Equal(t, "From feed", recorded.Title)
}

func TestIngestEntryIntoManualRejected(t *testing.T) {
	svc, _, sources := newService()
	manualID := seedManualSource(sources)

	_, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: manualID,
		Title:    "Worker should not",
		Link:     "https://example.com/x",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestIngestEntryIntoDeactivatedRSSRejected(t *testing.T) {
	svc, _, sources := newService()
	id := uuid.New()
	sources.seed(id, service.SourceTypeRSS, false)

	_, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: id,
		Title:    "Stopped feed",
		Link:     "https://example.com/stopped",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
}

func TestSameURLUnderManualAndRSSYieldsTwoEntries(t *testing.T) {
	svc, repo, sources := newService()
	manualID := seedManualSource(sources)
	rssID := seedRSSSource(sources)
	link := "https://example.com/shared"

	manualEntry, err := svc.SaveLink(context.Background(), service.SaveLinkInput{
		Title: "Saved myself",
		Link:  link,
	})
	require.NoError(t, err)

	rssEntry, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: rssID,
		Title:    "From RSS",
		Link:     link,
	})
	require.NoError(t, err)

	require.NotEqual(t, manualEntry.ID, rssEntry.ID)
	require.Equal(t, manualID, manualEntry.SourceID)
	require.Equal(t, rssID, rssEntry.SourceID)
	require.Len(t, repo.entries, 2)
}

func TestReadAndSavedBookmarksWorkOnManualAndRSSEntries(t *testing.T) {
	svc, _, sources := newService()
	manualID := seedManualSource(sources)
	rssID := seedRSSSource(sources)

	manualEntry, err := svc.SaveLink(context.Background(), service.SaveLinkInput{
		Title: "Manual item",
		Link:  "https://example.com/manual-bm",
	})
	require.NoError(t, err)
	require.Equal(t, manualID, manualEntry.SourceID)

	rssEntry, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: rssID,
		Title:    "RSS item",
		Link:     "https://example.com/rss-bm",
	})
	require.NoError(t, err)

	for _, id := range []uuid.UUID{manualEntry.ID, rssEntry.ID} {
		read, err := svc.MarkEntryRead(context.Background(), id)
		require.NoError(t, err)
		require.NotNil(t, read.ReadAt)

		saved, err := svc.MarkEntrySaved(context.Background(), id)
		require.NoError(t, err)
		require.NotNil(t, saved.SavedAt)
		require.NotNil(t, saved.ReadAt)

		clearedRead, err := svc.ClearEntryRead(context.Background(), id)
		require.NoError(t, err)
		require.Nil(t, clearedRead.ReadAt)
		require.NotNil(t, clearedRead.SavedAt)

		clearedSaved, err := svc.ClearEntrySaved(context.Background(), id)
		require.NoError(t, err)
		require.Nil(t, clearedSaved.SavedAt)
	}
}

func TestIngestEntryMakesItRetrievable(t *testing.T) {
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	thumb := "https://example.com/thumb.png"
	published := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	recorded, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID:    sourceID,
		Title:       "Show HN: Quick Feed",
		Link:        "https://example.com/a",
		Description: "A personal reader",
		Thumbnail:   &thumb,
		PublishedAt: &published,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, recorded.ID)
	require.Equal(t, sourceID, recorded.SourceID)
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
	svc, _, _ := newService()

	_, err := svc.GetEntry(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestIngestEntryBlankTitleValidation(t *testing.T) {
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	_, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: sourceID,
		Title:    "   ",
		Link:     "https://example.com/a",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestIngestEntryBlankLinkValidation(t *testing.T) {
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	_, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: sourceID,
		Title:    "Show HN",
		Link:     "",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestIngestEntryMissingSourceRejected(t *testing.T) {
	svc, _, _ := newService()

	_, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: uuid.New(),
		Title:    "Show HN",
		Link:     "https://example.com/a",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestIngestEntrySameSourceAndLinkUpsertsMetadata(t *testing.T) {
	svc, repo, sources := newService()
	sourceID := seedSource(sources)

	first, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID:    sourceID,
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
	second, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID:    sourceID,
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

func TestListEntriesIncludesEntriesWhoseSourceIsDeactivated(t *testing.T) {
	// Entry list does not filter by Source active state. After a Source is
	// deactivated (source module), its existing Entries stay in the stream.
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	recorded, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: sourceID,
		Title:    "From a feed I unfollow",
		Link:     "https://example.com/kept",
	})
	require.NoError(t, err)

	pageSize := 10
	page, err := svc.ListEntries(context.Background(), service.ListEntriesInput{PageSize: &pageSize})
	require.NoError(t, err)
	require.Len(t, page.Entries, 1)
	require.Equal(t, recorded.ID, page.Entries[0].ID)
	require.Equal(t, sourceID, page.Entries[0].SourceID)
}

func TestListEntriesNewestFirstPublishedAtFallbackCreatedAtIDTieBreak(t *testing.T) {
	svc, repo, sources := newService()
	sourceID := seedSource(sources)

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
		ID: idHexA, SourceID: sourceID, Title: "same-a", Link: "https://example.com/same-a",
		PublishedAt: &pubSame, CreatedAt: pubSame, UpdatedAt: pubSame,
	})
	repo.seedEntry(&domain.Entry{
		ID: idHex10, SourceID: sourceID, Title: "same-10", Link: "https://example.com/same-10",
		PublishedAt: &pubSame, CreatedAt: pubSame, UpdatedAt: pubSame,
	})
	repo.seedEntry(&domain.Entry{
		ID: idNewest, SourceID: sourceID, Title: "newest", Link: "https://example.com/newest",
		PublishedAt: &pubNewest, CreatedAt: pubNewest, UpdatedAt: pubNewest,
	})
	repo.seedEntry(&domain.Entry{
		ID: idFallback, SourceID: sourceID, Title: "no-pub", Link: "https://example.com/no-pub",
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
	svc, repo, sources := newService()
	sourceID := seedSource(sources)

	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	t4 := time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC)

	id1 := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	id2 := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	id3 := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	id4 := uuid.MustParse("00000000-0000-4000-8000-000000000004")

	seedTimedEntry(repo, sourceID, id1, "https://example.com/1", &t1, t1)
	seedTimedEntry(repo, sourceID, id2, "https://example.com/2", &t2, t2)
	seedTimedEntry(repo, sourceID, id3, "https://example.com/3", &t3, t3)
	seedTimedEntry(repo, sourceID, id4, "https://example.com/4", &t4, t4)

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
	seedTimedEntry(repo, sourceID, id5, "https://example.com/5", &t5, t5)

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
	svc, repo, sources := newService()
	sourceID := seedSource(sources)

	t1 := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	seedTimedEntry(repo, sourceID, uuid.MustParse("00000000-0000-4000-8000-000000000011"), "https://example.com/a", &t1, t1)

	pageSize := 5
	page, err := svc.ListEntries(context.Background(), service.ListEntriesInput{PageSize: &pageSize})
	require.NoError(t, err)
	require.Len(t, page.Entries, 1)
	require.Nil(t, page.NextCursor)
}

func TestListEntriesInvalidCursorFailsClearly(t *testing.T) {
	svc, _, _ := newService()
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
	svc, repo, sources := newService()
	sourceID := seedSource(sources)

	base := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	total := service.DefaultEntryPageSize + 3
	for i := 0; i < total; i++ {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", i+1))
		pub := base.Add(time.Duration(i) * time.Hour)
		seedTimedEntry(repo, sourceID, id, fmt.Sprintf("https://example.com/%d", i), &pub, pub)
	}

	page, err := svc.ListEntries(context.Background(), service.ListEntriesInput{})
	require.NoError(t, err)
	require.Len(t, page.Entries, service.DefaultEntryPageSize)
	require.NotNil(t, page.NextCursor)
}

func TestListEntriesInvalidSmallPageSizeUsesDefault(t *testing.T) {
	svc, repo, sources := newService()
	sourceID := seedSource(sources)

	base := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	total := service.DefaultEntryPageSize + 2
	for i := 0; i < total; i++ {
		id := uuid.MustParse(fmt.Sprintf("00000000-0000-4000-8000-%012x", i+1))
		pub := base.Add(time.Duration(i) * time.Hour)
		seedTimedEntry(repo, sourceID, id, fmt.Sprintf("https://example.com/%d", i), &pub, pub)
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
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	recorded, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: sourceID,
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
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	recorded, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: sourceID,
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
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	recorded, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: sourceID,
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
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	recorded, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: sourceID,
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
	svc, _, _ := newService()

	_, err := svc.MarkEntryRead(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestMarkEntrySavedUnknownIDNotFound(t *testing.T) {
	svc, _, _ := newService()

	_, err := svc.MarkEntrySaved(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestClearEntryReadClearsReadAt(t *testing.T) {
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	recorded, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: sourceID,
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
	svc, _, sources := newService()
	sourceID := seedSource(sources)

	recorded, err := svc.IngestEntry(context.Background(), service.IngestEntryInput{
		SourceID: sourceID,
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
	svc, _, _ := newService()

	_, err := svc.ClearEntryRead(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}
