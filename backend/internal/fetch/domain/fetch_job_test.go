package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/quick-feed/backend/internal/fetch/domain"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

func TestNewFetchJobStartsPending(t *testing.T) {
	sourceID := uuid.New()
	job, err := domain.NewFetchJob(sourceID)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, job.ID)
	require.Equal(t, sourceID, job.SourceID)
	require.Equal(t, domain.StatusPending, job.Status)
	require.Equal(t, 0, job.Attempt)
	require.Equal(t, domain.DefaultMaxAttempts, job.MaxAttempts)
	require.Equal(t, 0, job.EntriesCreated)
	require.False(t, job.CreatedAt.IsZero())
	require.Equal(t, job.CreatedAt, job.UpdatedAt)
}

func TestNewFetchJobRejectsNilSourceID(t *testing.T) {
	_, err := domain.NewFetchJob(uuid.Nil)
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestClaimPendingMovesToRunningAndIncrementsAttempt(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	before := job.UpdatedAt
	now := before.Add(2 * time.Millisecond)

	require.NoError(t, job.Claim(now))
	require.Equal(t, domain.StatusRunning, job.Status)
	require.Equal(t, 1, job.Attempt)
	require.True(t, job.UpdatedAt.After(before))
	require.True(t, job.NextAttemptAt.IsZero())
}

func TestClaimRejectsNonPending(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	require.NoError(t, job.Claim(time.Now().UTC()))

	err = job.Claim(time.Now().UTC())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
}

func TestClaimRejectsBeforeNextAttemptAt(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	require.NoError(t, job.Claim(now))
	require.NoError(t, job.ScheduleRetry(now.Add(time.Minute)))

	err = job.Claim(now.Add(30 * time.Second))
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
	require.Equal(t, domain.StatusPending, job.Status)
	require.Equal(t, 1, job.Attempt)
}

func TestSucceedRunningRecordsCreatedCount(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	require.NoError(t, job.Claim(time.Now().UTC()))

	require.NoError(t, job.Succeed(3))
	require.Equal(t, domain.StatusSucceeded, job.Status)
	require.Equal(t, 3, job.EntriesCreated)
}

func TestSucceedAllowsZeroCreated(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	require.NoError(t, job.Claim(time.Now().UTC()))

	require.NoError(t, job.Succeed(0))
	require.Equal(t, domain.StatusSucceeded, job.Status)
	require.Equal(t, 0, job.EntriesCreated)
}

func TestSucceedRejectsNegativeCreated(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	require.NoError(t, job.Claim(time.Now().UTC()))

	err = job.Succeed(-1)
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestSucceedRejectsNonRunning(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)

	err = job.Succeed(1)
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
}

func TestFailRunningMovesToFailed(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	require.NoError(t, job.Claim(time.Now().UTC()))

	require.NoError(t, job.Fail())
	require.Equal(t, domain.StatusFailed, job.Status)
}

func TestFailRejectsNonRunning(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)

	err = job.Fail()
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
}

func TestFailPendingMovesPendingToFailed(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)

	require.NoError(t, job.FailPending())
	require.Equal(t, domain.StatusFailed, job.Status)
}

func TestFailPendingRejectsNonPending(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	require.NoError(t, job.Claim(time.Now().UTC()))

	err = job.FailPending()
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
}

func TestScheduleRetryMovesToPendingWithNextAttemptAt(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	require.NoError(t, job.Claim(now))

	next := now.Add(time.Minute)
	require.NoError(t, job.ScheduleRetry(next))
	require.Equal(t, domain.StatusPending, job.Status)
	require.Equal(t, 1, job.Attempt)
	require.Equal(t, next, job.NextAttemptAt)

	require.NoError(t, job.Claim(next))
	require.Equal(t, domain.StatusRunning, job.Status)
	require.Equal(t, 2, job.Attempt)
	require.True(t, job.NextAttemptAt.IsZero())
}

func TestScheduleRetryRejectsWhenNoAttemptsRemain(t *testing.T) {
	job, err := domain.NewFetchJob(uuid.New())
	require.NoError(t, err)
	job.MaxAttempts = 1
	require.NoError(t, job.Claim(time.Now().UTC()))

	err = job.ScheduleRetry(time.Now().UTC().Add(time.Minute))
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
}
