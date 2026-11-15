package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/fetch/domain"
	"github.com/larb26656/my-feed/backend/internal/fetch/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

// DefaultRetryBackoff is the fixed delay between attempts on the same Fetch Job.
const DefaultRetryBackoff = time.Minute

// DefaultScheduleInterval is the global fixed-rate interval for schedule rounds.
const DefaultScheduleInterval = 15 * time.Minute

// SourceType is the Fetch module's view of a Source's type.
type SourceType string

const (
	SourceTypeRSS    SourceType = "rss"
	SourceTypeManual SourceType = "manual"
)

// SourceRef is the Fetch module's lookup view of a Source for create/obtain rules.
type SourceRef struct {
	ID     uuid.UUID
	Type   SourceType
	Active bool
	RSSURL string
}

// Sources is the consumer-defined port Fetch uses for Source lookups.
type Sources interface {
	Get(ctx context.Context, id uuid.UUID) (*SourceRef, error)
	// ListActiveFetchable returns active fetchable Sources (excludes manual and deactivated).
	ListActiveFetchable(ctx context.Context) ([]SourceRef, error)
}

// PreparedEntry is Entry-ready fields returned by an obtain adapter (no DB writes).
type PreparedEntry struct {
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

// Obtain is the port that obtains Entry-ready items for a Source (fake or real adapter).
type Obtain interface {
	Obtain(ctx context.Context, source *SourceRef) ([]PreparedEntry, error)
}

// IngestEntryInput is the record contract Fetch drives through the entry ingest port.
type IngestEntryInput struct {
	SourceID    uuid.UUID
	Title       string
	Link        string
	Description string
	Thumbnail   *string
	PublishedAt *time.Time
}

// IngestEntryResult reports whether ingest newly created an Entry.
type IngestEntryResult struct {
	Created bool
}

// EntryIngest is the consumer-defined port Fetch uses to record Entries.
type EntryIngest interface {
	IngestEntry(ctx context.Context, in IngestEntryInput) (*IngestEntryResult, error)
}

// Clock provides the current time for retry backoff (injectable in tests).
type Clock interface {
	Now() time.Time
}

// Service owns Fetch Job create and execute orchestration.
type Service struct {
	repo             repository.Repository
	sources          Sources
	obtain           Obtain
	ingest           EntryIngest
	clock            Clock
	retryBackoff     time.Duration
	scheduleInterval time.Duration
}

func New(
	repo repository.Repository,
	sources Sources,
	obtain Obtain,
	ingest EntryIngest,
	clock Clock,
	retryBackoff time.Duration,
	scheduleInterval time.Duration,
) *Service {
	if retryBackoff <= 0 {
		retryBackoff = DefaultRetryBackoff
	}
	if scheduleInterval <= 0 {
		scheduleInterval = DefaultScheduleInterval
	}
	return &Service{
		repo:             repo,
		sources:          sources,
		obtain:           obtain,
		ingest:           ingest,
		clock:            clock,
		retryBackoff:     retryBackoff,
		scheduleInterval: scheduleInterval,
	}
}

// ScheduleInterval returns the global fixed-rate interval for schedule rounds.
func (s *Service) ScheduleInterval() time.Duration {
	return s.scheduleInterval
}

// GetFetchJob returns a Fetch Job by id for status observation.
func (s *Service) GetFetchJob(ctx context.Context, id uuid.UUID) (*domain.FetchJob, error) {
	return s.repo.GetFetchJobByID(ctx, id)
}

const DefaultFetchJobPageSize = 20

type ListFetchJobsInput struct {
	Cursor   *string
	PageSize *int
	Criteria repository.FetchJobListCriteria
}

type ListFetchJobsResult struct {
	Jobs       []*domain.FetchJob
	NextCursor *string
}

func (s *Service) ListFetchJobs(ctx context.Context, in ListFetchJobsInput) (*ListFetchJobsResult, error) {
	pageSize := DefaultFetchJobPageSize
	if in.PageSize != nil {
		if *in.PageSize < 1 || *in.PageSize > 100 {
			return nil, apperr.Validation("pageSize must be between 1 and 100")
		}
		pageSize = *in.PageSize
	}
	var after *repository.FetchJobListCursor
	if in.Cursor != nil {
		decoded, err := decodeFetchJobCursor(*in.Cursor, in.Criteria)
		if err != nil {
			return nil, err
		}
		after = decoded
	}
	rows, err := s.repo.ListFetchJobs(ctx, after, in.Criteria, pageSize+1)
	if err != nil {
		return nil, err
	}
	result := &ListFetchJobsResult{Jobs: rows}
	if len(rows) > pageSize {
		result.Jobs = rows[:pageSize]
		last := result.Jobs[pageSize-1]
		token := encodeFetchJobCursor(last.CreatedAt, last.ID, in.Criteria)
		result.NextCursor = &token
	}
	return result, nil
}

type fetchJobCursorPayload struct {
	CreatedAt time.Time      `json:"createdAt"`
	ID        uuid.UUID      `json:"id"`
	SourceID  *uuid.UUID     `json:"sourceId"`
	Status    *domain.Status `json:"status"`
}

func encodeFetchJobCursor(createdAt time.Time, id uuid.UUID, criteria repository.FetchJobListCriteria) string {
	raw, _ := json.Marshal(fetchJobCursorPayload{CreatedAt: createdAt.UTC(), ID: id, SourceID: criteria.SourceID, Status: criteria.Status})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeFetchJobCursor(cursor string, criteria repository.FetchJobListCriteria) (*repository.FetchJobListCursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, apperr.Validation("invalid cursor")
	}
	var payload fetchJobCursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil || payload.CreatedAt.IsZero() || payload.ID == uuid.Nil {
		return nil, apperr.Validation("invalid cursor")
	}
	if (payload.SourceID == nil) != (criteria.SourceID == nil) || (payload.SourceID != nil && *payload.SourceID != *criteria.SourceID) || (payload.Status == nil) != (criteria.Status == nil) || (payload.Status != nil && *payload.Status != *criteria.Status) {
		return nil, apperr.Validation("cursor does not match filters")
	}
	return &repository.FetchJobListCursor{CreatedAt: payload.CreatedAt.UTC(), ID: payload.ID}, nil
}

// WithRepository returns a shallow copy that persists through repo (e.g. a transaction).
func (s *Service) WithRepository(repo repository.Repository) *Service {
	cp := *s
	cp.repo = repo
	return &cp
}

// ExecuteReadyFetchJobs claims and runs all claimable pending jobs in parallel
// (one goroutine per job). One unfinished job per Source remains enforced by create/claim rules.
func (s *Service) ExecuteReadyFetchJobs(ctx context.Context) error {
	jobs, err := s.repo.ListClaimableFetchJobs(ctx, s.clock.Now())
	if err != nil {
		return err
	}
	if len(jobs) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(jobs))
	for _, job := range jobs {
		wg.Add(1)
		go func(jobID uuid.UUID) {
			defer wg.Done()
			if _, err := s.ExecuteFetchJob(ctx, jobID); err != nil {
				errCh <- err
			}
		}(job.ID)
	}
	wg.Wait()
	close(errCh)

	// Prefer the first business/operational error if any; parallel peers may still succeed.
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

// ScheduleRound creates one pending Fetch Job per active fetchable Source that
// does not already have an unfinished job. Sources with unfinished jobs are skipped.
func (s *Service) ScheduleRound(ctx context.Context) ([]*domain.FetchJob, error) {
	sources, err := s.sources.ListActiveFetchable(ctx)
	if err != nil {
		return nil, err
	}

	created := make([]*domain.FetchJob, 0, len(sources))
	for _, src := range sources {
		unfinished, err := s.repo.GetUnfinishedFetchJobBySourceID(ctx, src.ID)
		if err == nil && unfinished != nil {
			continue
		}
		if err != nil {
			if ae, ok := err.(*apperr.Error); !ok || ae.Code != apperr.CodeNotFound {
				return created, err
			}
		}

		job, err := domain.NewFetchJob(src.ID)
		if err != nil {
			return created, err
		}
		if err := s.repo.CreateFetchJob(ctx, job); err != nil {
			return created, err
		}
		created = append(created, job)
	}
	return created, nil
}

// CreateManualFetchJob creates a pending Fetch Job for one active fetchable Source.
func (s *Service) CreateManualFetchJob(ctx context.Context, sourceID uuid.UUID) (*domain.FetchJob, error) {
	src, err := s.sources.Get(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if src.Type != SourceTypeRSS {
		return nil, apperr.Validation("fetch jobs can only target fetchable sources")
	}
	if !src.Active {
		return nil, apperr.FailedPrecondition("cannot create a fetch job for a deactivated source")
	}

	unfinished, err := s.repo.GetUnfinishedFetchJobBySourceID(ctx, sourceID)
	if err == nil && unfinished != nil {
		return nil, apperr.Conflict("source already has an unfinished fetch job")
	}
	if err != nil {
		if ae, ok := err.(*apperr.Error); !ok || ae.Code != apperr.CodeNotFound {
			return nil, err
		}
	}

	job, err := domain.NewFetchJob(sourceID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateFetchJob(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

// HandleSourceDeactivated fails any pending Fetch Job for the Source.
// An already-running job is left alone so it may finish normally.
func (s *Service) HandleSourceDeactivated(ctx context.Context, sourceID uuid.UUID) error {
	job, err := s.repo.GetUnfinishedFetchJobBySourceID(ctx, sourceID)
	if err != nil {
		if ae, ok := err.(*apperr.Error); ok && ae.Code == apperr.CodeNotFound {
			return nil
		}
		return err
	}
	if job.Status != domain.StatusPending {
		return nil
	}
	if err := job.FailPending(); err != nil {
		return err
	}
	return s.repo.UpdateFetchJob(ctx, job)
}

// ExecuteFetchJob claims a pending job, runs obtain→record, and finishes succeeded,
// schedules a retry on the same job, or marks failed at max attempts.
func (s *Service) ExecuteFetchJob(ctx context.Context, jobID uuid.UUID) (*domain.FetchJob, error) {
	job, err := s.repo.ClaimFetchJob(ctx, jobID, s.clock.Now())
	if err != nil {
		return nil, err
	}

	src, err := s.sources.Get(ctx, job.SourceID)
	if err != nil {
		return s.failOrRetry(ctx, job, err)
	}

	items, err := s.obtain.Obtain(ctx, src)
	if err != nil {
		return s.failOrRetry(ctx, job, err)
	}

	created := 0
	for _, item := range items {
		result, err := s.ingest.IngestEntry(ctx, IngestEntryInput{
			SourceID:    job.SourceID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Thumbnail:   item.Thumbnail,
			PublishedAt: item.PublishedAt,
		})
		if err != nil {
			return s.failOrRetry(ctx, job, err)
		}
		if result.Created {
			created++
		}
	}

	if err := job.Succeed(created); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateFetchJob(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Service) failOrRetry(ctx context.Context, job *domain.FetchJob, cause error) (*domain.FetchJob, error) {
	if job.Attempt >= job.MaxAttempts {
		if err := job.Fail(); err != nil {
			return job, err
		}
	} else {
		if err := job.ScheduleRetry(s.clock.Now().Add(s.retryBackoff)); err != nil {
			return job, err
		}
	}
	if err := s.repo.UpdateFetchJob(ctx, job); err != nil {
		return job, err
	}
	return job, cause
}
