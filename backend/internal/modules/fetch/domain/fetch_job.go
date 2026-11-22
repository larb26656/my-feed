package domain

import (
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"

	// DefaultMaxAttempts is the product default for new Fetch Jobs.
	DefaultMaxAttempts = 3
)

// FetchJob is one unit of work that obtains content from a single fetchable Source,
// maps it into Entry fields, and records those Entries.
type FetchJob struct {
	ID             uuid.UUID
	SourceID       uuid.UUID
	Status         Status
	Attempt        int
	MaxAttempts    int
	EntriesCreated int
	// NextAttemptAt is when a pending job may be claimed again (zero = immediately).
	NextAttemptAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewFetchJob creates a pending Fetch Job for the given Source.
func NewFetchJob(sourceID uuid.UUID) (*FetchJob, error) {
	if sourceID == uuid.Nil {
		return nil, apperr.InvalidInput.New("source id is required")
	}
	now := time.Now().UTC()
	return &FetchJob{
		ID:          uuid.New(),
		SourceID:    sourceID,
		Status:      StatusPending,
		Attempt:     0,
		MaxAttempts: DefaultMaxAttempts,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// Claim moves pending → running and increments the attempt counter.
// now must be at or after NextAttemptAt when a retry backoff is set.
func (j *FetchJob) Claim(now time.Time) error {
	if j.Status != StatusPending {
		return apperr.ResourceConflict.New("only pending fetch jobs can be claimed")
	}
	if !j.NextAttemptAt.IsZero() && now.Before(j.NextAttemptAt) {
		return apperr.ResourceConflict.New("fetch job retry backoff has not elapsed")
	}
	j.Status = StatusRunning
	j.Attempt++
	j.NextAttemptAt = time.Time{}
	j.UpdatedAt = now.UTC()
	return nil
}

// Succeed moves running → succeeded and records how many Entries were newly created.
func (j *FetchJob) Succeed(entriesCreated int) error {
	if j.Status != StatusRunning {
		return apperr.ResourceConflict.New("only running fetch jobs can succeed")
	}
	if entriesCreated < 0 {
		return apperr.InvalidInput.New("entries created cannot be negative")
	}
	j.Status = StatusSucceeded
	j.EntriesCreated = entriesCreated
	j.UpdatedAt = time.Now().UTC()
	return nil
}

// Fail moves running → failed (terminal).
func (j *FetchJob) Fail() error {
	if j.Status != StatusRunning {
		return apperr.ResourceConflict.New("only running fetch jobs can fail")
	}
	j.Status = StatusFailed
	j.NextAttemptAt = time.Time{}
	j.UpdatedAt = time.Now().UTC()
	return nil
}

// ScheduleRetry moves running → pending after a failed attempt that still has
// retries remaining. nextAttemptAt is when the job may be claimed again.
func (j *FetchJob) ScheduleRetry(nextAttemptAt time.Time) error {
	if j.Status != StatusRunning {
		return apperr.ResourceConflict.New("only running fetch jobs can schedule a retry")
	}
	if j.Attempt >= j.MaxAttempts {
		return apperr.ResourceConflict.New("fetch job has no attempts remaining")
	}
	j.Status = StatusPending
	j.NextAttemptAt = nextAttemptAt.UTC()
	j.UpdatedAt = time.Now().UTC()
	return nil
}

// FailPending moves pending → failed (Source deactivated before the job was claimed).
func (j *FetchJob) FailPending() error {
	if j.Status != StatusPending {
		return apperr.ResourceConflict.New("only pending fetch jobs can be failed without running")
	}
	j.Status = StatusFailed
	j.UpdatedAt = time.Now().UTC()
	return nil
}
