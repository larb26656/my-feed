package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	entryapi "github.com/larb26656/my-feed/backend/internal/modules/entry/api"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/repository"
	"github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

type fakeRepo struct {
	mu   sync.Mutex
	jobs map[uuid.UUID]*domain.FetchJob
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{jobs: make(map[uuid.UUID]*domain.FetchJob)}
}

func cloneJob(j *domain.FetchJob) *domain.FetchJob {
	cp := *j
	return &cp
}

func (f *fakeRepo) CreateFetchJob(_ context.Context, job *domain.FetchJob) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.jobs[job.ID]; ok {
		return apperr.ResourceConflict.New("fetch job already exists")
	}
	f.jobs[job.ID] = cloneJob(job)
	return nil
}

func (f *fakeRepo) GetFetchJobByID(_ context.Context, id uuid.UUID) (*domain.FetchJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.jobs[id]
	if !ok {
		return nil, apperr.ResourceNotFound.New("fetch job not found")
	}
	return cloneJob(job), nil
}

func (f *fakeRepo) ListFetchJobs(_ context.Context, after *repository.FetchJobListCursor, criteria repository.FetchJobListCriteria, limit int) ([]*domain.FetchJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*domain.FetchJob, 0)
	for _, job := range f.jobs {
		if criteria.SourceID != nil && job.SourceID != *criteria.SourceID || criteria.Status != nil && job.Status != *criteria.Status {
			continue
		}
		if after != nil {
			order := job.CreatedAt.Compare(after.CreatedAt)
			if order > 0 || (order == 0 && job.ID.String() >= after.ID.String()) {
				continue
			}
		}
		out = append(out, cloneJob(job))
	}
	return out, nil
}

func (f *fakeRepo) GetUnfinishedFetchJobBySourceID(_ context.Context, sourceID uuid.UUID) (*domain.FetchJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, job := range f.jobs {
		if job.SourceID == sourceID && (job.Status == domain.StatusPending || job.Status == domain.StatusRunning) {
			return cloneJob(job), nil
		}
	}
	return nil, apperr.ResourceNotFound.New("unfinished fetch job not found")
}

func (f *fakeRepo) ListClaimableFetchJobs(_ context.Context, now time.Time) ([]*domain.FetchJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*domain.FetchJob, 0)
	for _, job := range f.jobs {
		if job.Status != domain.StatusPending {
			continue
		}
		if !job.NextAttemptAt.IsZero() && now.Before(job.NextAttemptAt) {
			continue
		}
		out = append(out, cloneJob(job))
	}
	return out, nil
}

func (f *fakeRepo) ClaimFetchJob(_ context.Context, id uuid.UUID, now time.Time) (*domain.FetchJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.jobs[id]
	if !ok {
		return nil, apperr.ResourceNotFound.New("fetch job not found")
	}
	job := cloneJob(stored)
	if err := job.Claim(now); err != nil {
		return nil, err
	}
	// Conditional claim: only persist when still pending in storage (simulates RowsAffected).
	if stored.Status != domain.StatusPending {
		return nil, apperr.ResourceConflict.New("fetch job was already claimed or not claimable")
	}
	if !stored.NextAttemptAt.IsZero() && now.Before(stored.NextAttemptAt) {
		return nil, apperr.ResourceConflict.New("fetch job was already claimed or not claimable")
	}
	f.jobs[id] = cloneJob(job)
	return cloneJob(job), nil
}

func (f *fakeRepo) UpdateFetchJob(_ context.Context, job *domain.FetchJob) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.jobs[job.ID]; !ok {
		return apperr.ResourceNotFound.New("fetch job not found")
	}
	f.jobs[job.ID] = cloneJob(job)
	return nil
}

func (f *fakeRepo) WithTx(database.Transaction) repository.Repository { return f }

type fakeSource struct {
	id     uuid.UUID
	typ    sourceapi.SourceType
	active bool
	rssURL string
}

type fakeSources struct {
	byID            map[uuid.UUID]fakeSource
	activeFetchable []uuid.UUID
}

func newFakeSources() *fakeSources {
	return &fakeSources{byID: make(map[uuid.UUID]fakeSource)}
}

func (f *fakeSources) seed(id uuid.UUID, typ sourceapi.SourceType, active bool, rssURL string) {
	f.byID[id] = fakeSource{id: id, typ: typ, active: active, rssURL: rssURL}
}

func (f *fakeSources) seedActiveFetchable(id uuid.UUID, rssURL string) {
	f.seed(id, sourceapi.SourceTypeRSS, true, rssURL)
	f.activeFetchable = append(f.activeFetchable, id)
}

func (f *fakeSources) Get(_ context.Context, id uuid.UUID) (*sourceapi.SourceRef, error) {
	src, ok := f.byID[id]
	if !ok {
		return nil, apperr.ResourceNotFound.New("source not found")
	}
	return &sourceapi.SourceRef{
		ID:     src.id,
		Type:   src.typ,
		Active: src.active,
		RSSURL: src.rssURL,
	}, nil
}

func (f *fakeSources) ListActiveFetchable(_ context.Context) ([]sourceapi.SourceRef, error) {
	out := make([]sourceapi.SourceRef, 0, len(f.activeFetchable))
	for _, id := range f.activeFetchable {
		src, ok := f.byID[id]
		if !ok {
			continue
		}
		out = append(out, sourceapi.SourceRef{
			ID:     src.id,
			Type:   src.typ,
			Active: src.active,
			RSSURL: src.rssURL,
		})
	}
	return out, nil
}

type fakeObtain struct {
	mu    sync.Mutex
	items []obtainapi.PreparedEntry
	err   error
	calls int
	hook  func()
}

func (f *fakeObtain) Obtain(_ context.Context, _ *obtainapi.Input) ([]obtainapi.PreparedEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.hook != nil {
		f.hook()
	}
	if f.err != nil {
		return nil, f.err
	}
	out := make([]obtainapi.PreparedEntry, len(f.items))
	copy(out, f.items)
	return out, nil
}

type ingestCall struct {
	sourceID uuid.UUID
	item     entryapi.IngestEntryInput
}

type fakeIngest struct {
	mu      sync.Mutex
	sources *fakeSources
	// createdByLink reports whether ingest of that link should count as created.
	createdByLink map[string]bool
	err           error
	calls         []ingestCall
}

func (f *fakeIngest) IngestEntry(ctx context.Context, in entryapi.IngestEntryInput) (*entryapi.IngestEntryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, ingestCall{sourceID: in.SourceID, item: in})
	if f.err != nil {
		return nil, f.err
	}
	if f.sources != nil {
		src, err := f.sources.Get(ctx, in.SourceID)
		if err != nil {
			return nil, err
		}
		if src.Type != sourceapi.SourceTypeRSS {
			return nil, apperr.InvalidInput.New("ingest can only record entries into rss sources")
		}
		if !src.Active {
			return nil, apperr.ResourceConflict.New("cannot ingest into a deactivated source")
		}
	}
	created, ok := f.createdByLink[in.Link]
	if !ok {
		created = true
	}
	return &entryapi.IngestEntryResult{Created: created}, nil
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

type harness struct {
	svc     *service.Service
	repo    *fakeRepo
	sources *fakeSources
	obtain  *fakeObtain
	ingest  *fakeIngest
	clock   *fakeClock
}

func newHarness() *harness {
	repo := newFakeRepo()
	sources := newFakeSources()
	obtain := &fakeObtain{}
	ingest := &fakeIngest{sources: sources, createdByLink: make(map[string]bool)}
	clock := &fakeClock{now: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)}
	return &harness{
		svc: service.New(
			repo, sources, obtain, ingest, clock,
			service.DefaultRetryBackoff,
			service.DefaultScheduleInterval,
		),
		repo:    repo,
		sources: sources,
		obtain:  obtain,
		ingest:  ingest,
		clock:   clock,
	}
}

func seedActiveRSS(sources *fakeSources) uuid.UUID {
	id := uuid.New()
	sources.seedActiveFetchable(id, "https://example.com/feed.xml")
	return id
}

func TestScheduleRoundCreatesPendingJobPerEligibleSource(t *testing.T) {
	h := newHarness()
	a := seedActiveRSS(h.sources)
	b := seedActiveRSS(h.sources)

	jobs, err := h.svc.ScheduleRound(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 2)

	bySource := map[uuid.UUID]*domain.FetchJob{}
	for _, job := range jobs {
		require.Equal(t, domain.StatusPending, job.Status)
		require.Equal(t, domain.DefaultMaxAttempts, job.MaxAttempts)
		bySource[job.SourceID] = job
	}
	require.Contains(t, bySource, a)
	require.Contains(t, bySource, b)

	storedA, err := h.repo.GetUnfinishedFetchJobBySourceID(context.Background(), a)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, storedA.Status)
	storedB, err := h.repo.GetUnfinishedFetchJobBySourceID(context.Background(), b)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, storedB.Status)
}

func TestScheduleRoundSkipsSourcesWithUnfinishedJob(t *testing.T) {
	h := newHarness()
	busy := seedActiveRSS(h.sources)
	free := seedActiveRSS(h.sources)

	existing, err := h.svc.CreateManualFetchJob(context.Background(), busy)
	require.NoError(t, err)

	jobs, err := h.svc.ScheduleRound(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, free, jobs[0].SourceID)
	require.Equal(t, domain.StatusPending, jobs[0].Status)

	still, err := h.repo.GetUnfinishedFetchJobBySourceID(context.Background(), busy)
	require.NoError(t, err)
	require.Equal(t, existing.ID, still.ID)
}

func TestScheduleRoundExcludesDeactivatedAndManualSources(t *testing.T) {
	h := newHarness()
	active := seedActiveRSS(h.sources)

	deactivated := uuid.New()
	h.sources.seed(deactivated, sourceapi.SourceTypeRSS, false, "https://example.com/off.xml")
	manual := uuid.New()
	h.sources.seed(manual, sourceapi.SourceTypeManual, true, "")

	jobs, err := h.svc.ScheduleRound(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	require.Equal(t, active, jobs[0].SourceID)

	_, err = h.repo.GetUnfinishedFetchJobBySourceID(context.Background(), deactivated)
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceNotFound.Code, ae.Code)

	_, err = h.repo.GetUnfinishedFetchJobBySourceID(context.Background(), manual)
	require.Error(t, err)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceNotFound.Code, ae.Code)
}

func TestScheduleIntervalDefaultsToFifteenMinutes(t *testing.T) {
	h := newHarness()
	require.Equal(t, 15*time.Minute, h.svc.ScheduleInterval())
	require.Equal(t, service.DefaultScheduleInterval, h.svc.ScheduleInterval())
}

func TestCreateManualFetchJobStartsPending(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)

	job, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, job.Status)
	require.Equal(t, sourceID, job.SourceID)
	require.Equal(t, domain.DefaultMaxAttempts, job.MaxAttempts)

	stored, err := h.repo.GetFetchJobByID(context.Background(), job.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, stored.Status)
}

func TestCreateManualFetchJobRejectsWhenPendingJobExists(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)

	first, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, first.Status)

	_, err = h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
}

func TestCreateManualFetchJobRejectsWhenRunningJobExists(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	_, err = h.repo.ClaimFetchJob(context.Background(), created.ID, h.clock.Now())
	require.NoError(t, err)

	_, err = h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
}

func TestClaimFetchJobRejectsSecondConcurrentClaim(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	first, err := h.repo.ClaimFetchJob(context.Background(), created.ID, h.clock.Now())
	require.NoError(t, err)
	require.Equal(t, domain.StatusRunning, first.Status)
	require.Equal(t, 1, first.Attempt)

	_, err = h.repo.ClaimFetchJob(context.Background(), created.ID, h.clock.Now())
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)

	stored, err := h.repo.GetFetchJobByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusRunning, stored.Status)
	require.Equal(t, 1, stored.Attempt)
}

func TestExecuteFetchJobRejectsWhenAlreadyClaimed(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.items = nil

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	_, err = h.repo.ClaimFetchJob(context.Background(), created.ID, h.clock.Now())
	require.NoError(t, err)

	_, err = h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
}

func TestCreateManualFetchJobAllowedAfterTerminalJob(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.items = nil

	first, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)
	_, err = h.svc.ExecuteFetchJob(context.Background(), first.ID)
	require.NoError(t, err)

	second, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, second.Status)
	require.NotEqual(t, first.ID, second.ID)
}

func TestCreateManualFetchJobRejectsManualSource(t *testing.T) {
	h := newHarness()
	manualID := uuid.New()
	h.sources.seed(manualID, sourceapi.SourceTypeManual, true, "")

	_, err := h.svc.CreateManualFetchJob(context.Background(), manualID)
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.InvalidInput.Code, ae.Code)
}

func TestCreateManualFetchJobRejectsMissingSource(t *testing.T) {
	h := newHarness()

	_, err := h.svc.CreateManualFetchJob(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceNotFound.Code, ae.Code)
}

func TestCreateManualFetchJobRejectsDeactivatedSource(t *testing.T) {
	h := newHarness()
	sourceID := uuid.New()
	h.sources.seed(sourceID, sourceapi.SourceTypeRSS, false, "https://example.com/feed.xml")

	_, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
}

func TestHandleSourceDeactivatedFailsPendingJob(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, created.Status)

	require.NoError(t, h.svc.HandleSourceDeactivated(context.Background(), sourceID))

	stored, err := h.repo.GetFetchJobByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusFailed, stored.Status)
}

func TestHandleSourceDeactivatedLeavesRunningJobUncancelledButIngestMayFail(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.items = []obtainapi.PreparedEntry{
		{Title: "One", Link: "https://example.com/1"},
	}

	var statusDuringObtain domain.Status
	h.obtain.hook = func() {
		unfinished, err := h.repo.GetUnfinishedFetchJobBySourceID(context.Background(), sourceID)
		require.NoError(t, err)
		require.Equal(t, domain.StatusRunning, unfinished.Status)

		require.NoError(t, h.svc.HandleSourceDeactivated(context.Background(), sourceID))

		still, err := h.repo.GetFetchJobByID(context.Background(), unfinished.ID)
		require.NoError(t, err)
		statusDuringObtain = still.Status

		// Source is deactivated mid-flight; job is not killed, but real ingest rejects inactive RSS.
		h.sources.seed(sourceID, sourceapi.SourceTypeRSS, false, "https://example.com/feed.xml")
	}

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	job, err := h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
	require.Equal(t, domain.StatusRunning, statusDuringObtain)
	// Attempt 1 of 3 → retry scheduled on the same job (not cancelled, not succeeded via bypass).
	require.Equal(t, domain.StatusPending, job.Status)
	require.False(t, job.NextAttemptAt.IsZero())
	require.Equal(t, 1, job.Attempt)
	require.Len(t, h.ingest.calls, 1)
}

func TestDeactivatedSourceReceivesNoNewFetchJobs(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)
	require.NoError(t, h.svc.HandleSourceDeactivated(context.Background(), sourceID))

	h.sources.seed(sourceID, sourceapi.SourceTypeRSS, false, "https://example.com/feed.xml")

	_, err = h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)

	stored, err := h.repo.GetFetchJobByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusFailed, stored.Status)
}

func TestExecuteFetchJobSucceedsWithCreatedCount(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.items = []obtainapi.PreparedEntry{
		{Title: "One", Link: "https://example.com/1"},
		{Title: "Two", Link: "https://example.com/2"},
	}
	h.ingest.createdByLink["https://example.com/1"] = true
	h.ingest.createdByLink["https://example.com/2"] = true

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	job, err := h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, job.Status)
	require.Equal(t, 2, job.EntriesCreated)
	require.Equal(t, 1, job.Attempt)
	require.Equal(t, 1, h.obtain.calls)
	require.Len(t, h.ingest.calls, 2)
	require.Equal(t, sourceID, h.ingest.calls[0].sourceID)
}

func TestExecuteFetchJobSucceedsWithZeroCreated(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.items = nil

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	job, err := h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, job.Status)
	require.Equal(t, 0, job.EntriesCreated)
}

func TestExecuteFetchJobRefreshDoesNotIncrementCreated(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.items = []obtainapi.PreparedEntry{
		{Title: "New", Link: "https://example.com/new"},
		{Title: "Seen", Link: "https://example.com/seen"},
	}
	h.ingest.createdByLink["https://example.com/new"] = true
	h.ingest.createdByLink["https://example.com/seen"] = false

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	job, err := h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, job.Status)
	require.Equal(t, 1, job.EntriesCreated)
}

func TestExecuteFetchJobObtainFailureSchedulesRetryOnSameJob(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.err = errors.New("origin unreachable")

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	job, err := h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.Error(t, err)
	require.Equal(t, created.ID, job.ID)
	require.Equal(t, domain.StatusPending, job.Status)
	require.Equal(t, 1, job.Attempt)
	require.Equal(t, domain.DefaultMaxAttempts, job.MaxAttempts)
	require.Equal(t, 0, job.EntriesCreated)
	require.Equal(t, h.clock.Now().Add(service.DefaultRetryBackoff), job.NextAttemptAt)
	require.Empty(t, h.ingest.calls)

	// Backoff not elapsed: cannot claim again yet.
	_, err = h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.Error(t, err)
	stored, err := h.repo.GetFetchJobByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, stored.Status)
	require.Equal(t, 1, stored.Attempt)
}

func TestExecuteFetchJobRetriesAfterBackoffAndSucceeds(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.err = errors.New("origin unreachable")
	h.obtain.items = []obtainapi.PreparedEntry{
		{Title: "One", Link: "https://example.com/1"},
	}

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	_, err = h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.Error(t, err)
	require.Equal(t, 1, h.obtain.calls)

	h.obtain.err = nil
	h.clock.advance(service.DefaultRetryBackoff)

	job, err := h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, job.ID)
	require.Equal(t, domain.StatusSucceeded, job.Status)
	require.Equal(t, 2, job.Attempt)
	require.Equal(t, 1, job.EntriesCreated)
	require.Equal(t, 2, h.obtain.calls)
	require.Len(t, h.ingest.calls, 1)
}

func TestExecuteFetchJobRecordFailureSchedulesRetry(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.items = []obtainapi.PreparedEntry{
		{Title: "One", Link: "https://example.com/1"},
	}
	h.ingest.err = errors.New("ingest blew up")

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	job, err := h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.Error(t, err)
	require.Equal(t, domain.StatusPending, job.Status)
	require.Equal(t, 1, job.Attempt)
	require.Equal(t, 1, h.obtain.calls)
}

func TestExecuteFetchJobReachesMaxAttemptsThenFailed(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.err = errors.New("origin unreachable")

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)
	require.Equal(t, 3, created.MaxAttempts)

	var job *domain.FetchJob
	for i := 1; i <= 3; i++ {
		job, err = h.svc.ExecuteFetchJob(context.Background(), created.ID)
		require.Error(t, err)
		if i < 3 {
			require.Equal(t, domain.StatusPending, job.Status)
			require.Equal(t, i, job.Attempt)
			h.clock.advance(service.DefaultRetryBackoff)
		}
	}
	require.Equal(t, domain.StatusFailed, job.Status)
	require.Equal(t, 3, job.Attempt)
	require.Equal(t, 3, h.obtain.calls)

	// Terminal failed: a new Fetch Job can be created for the same Source.
	next, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)
	require.NotEqual(t, created.ID, next.ID)
	require.Equal(t, domain.StatusPending, next.Status)
	require.Equal(t, 0, next.Attempt)
	require.Equal(t, domain.DefaultMaxAttempts, next.MaxAttempts)
}

func TestExecuteFetchJobEachAttemptRerunsFullPath(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	h.obtain.items = []obtainapi.PreparedEntry{
		{Title: "One", Link: "https://example.com/1"},
	}
	h.ingest.err = errors.New("ingest blew up")

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	_, err = h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.Error(t, err)
	require.Equal(t, 1, h.obtain.calls)
	require.Len(t, h.ingest.calls, 1)

	h.ingest.err = nil
	h.clock.advance(service.DefaultRetryBackoff)

	job, err := h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, job.Status)
	require.Equal(t, 2, h.obtain.calls)
	require.Len(t, h.ingest.calls, 2)
}

func TestExecuteFetchJobPassesPreparedFieldsToIngest(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	thumb := "https://example.com/t.png"
	published := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	h.obtain.items = []obtainapi.PreparedEntry{
		{
			Title:       "Full",
			Link:        "https://example.com/full",
			Description: "desc",
			Thumbnail:   &thumb,
			PublishedAt: &published,
		},
	}

	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	_, err = h.svc.ExecuteFetchJob(context.Background(), created.ID)
	require.NoError(t, err)
	require.Len(t, h.ingest.calls, 1)
	got := h.ingest.calls[0].item
	require.Equal(t, "Full", got.Title)
	require.Equal(t, "https://example.com/full", got.Link)
	require.Equal(t, "desc", got.Description)
	require.Equal(t, &thumb, got.Thumbnail)
	require.Equal(t, &published, got.PublishedAt)
}

func TestGetFetchJobReturnsStoredJob(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	created, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	got, err := h.svc.GetFetchJob(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created.ID, got.ID)
	require.Equal(t, domain.StatusPending, got.Status)
}

func TestExecuteReadyFetchJobsRunsPendingJobsAcrossSources(t *testing.T) {
	h := newHarness()
	a := seedActiveRSS(h.sources)
	b := seedActiveRSS(h.sources)
	h.obtain.items = []obtainapi.PreparedEntry{
		{Title: "Shared", Link: "https://example.com/shared"},
	}

	jobA, err := h.svc.CreateManualFetchJob(context.Background(), a)
	require.NoError(t, err)
	jobB, err := h.svc.CreateManualFetchJob(context.Background(), b)
	require.NoError(t, err)

	require.NoError(t, h.svc.ExecuteReadyFetchJobs(context.Background()))

	gotA, err := h.svc.GetFetchJob(context.Background(), jobA.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, gotA.Status)
	gotB, err := h.svc.GetFetchJob(context.Background(), jobB.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSucceeded, gotB.Status)

	h.obtain.mu.Lock()
	require.Equal(t, 2, h.obtain.calls)
	h.obtain.mu.Unlock()
}

func TestExecuteReadyFetchJobsStillEnforcesOneUnfinishedPerSource(t *testing.T) {
	h := newHarness()
	sourceID := seedActiveRSS(h.sources)
	first, err := h.svc.CreateManualFetchJob(context.Background(), sourceID)
	require.NoError(t, err)

	_, err = h.svc.CreateManualFetchJob(context.Background(), sourceID)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.ResourceConflict.Code, ae.Code)

	stored, err := h.svc.GetFetchJob(context.Background(), first.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, stored.Status)
}
