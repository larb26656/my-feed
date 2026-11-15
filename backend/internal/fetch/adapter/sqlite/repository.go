package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/quick-feed/backend/internal/fetch/domain"
	"github.com/larb26656/quick-feed/backend/internal/fetch/repository"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
	"github.com/larb26656/quick-feed/backend/internal/platform/database"
)

// Querier is implemented by both *sql.DB and *sql.Tx.
type Querier interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository struct{ q Querier }

func NewRepository(db *sql.DB) *Repository { return &Repository{q: db} }

func (r *Repository) WithTx(tx database.Transaction) repository.Repository {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		panic("sqlite fetch repository received a non-SQLite transaction")
	}
	return &Repository{q: sqlTx}
}

func (r *Repository) CreateFetchJob(ctx context.Context, job *domain.FetchJob) error {
	_, err := r.q.ExecContext(ctx, `INSERT INTO fetch_jobs
		(id, source_id, status, attempt, max_attempts, entries_created, next_attempt_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, job.ID.String(), job.SourceID.String(), string(job.Status), job.Attempt,
		job.MaxAttempts, job.EntriesCreated, nullableTime(job.NextAttemptAt), formatTime(job.CreatedAt), formatTime(job.UpdatedAt))
	if err != nil {
		if database.IsUniqueConstraintViolation(err, "fetch_jobs.source_id") {
			return apperr.Conflict("source already has an unfinished fetch job")
		}
		return err
	}
	return nil
}

func (r *Repository) GetFetchJobByID(ctx context.Context, id uuid.UUID) (*domain.FetchJob, error) {
	return scanFetchJob(r.q.QueryRowContext(ctx, selectFetchJob+` WHERE id = ?`, id.String()))
}

func (r *Repository) GetUnfinishedFetchJobBySourceID(ctx context.Context, sourceID uuid.UUID) (*domain.FetchJob, error) {
	return scanFetchJob(r.q.QueryRowContext(ctx, selectFetchJob+` WHERE source_id = ? AND status IN ('pending', 'running') LIMIT 1`, sourceID.String()))
}

func (r *Repository) ListClaimableFetchJobs(ctx context.Context, now time.Time) ([]*domain.FetchJob, error) {
	rows, err := r.q.QueryContext(ctx, selectFetchJob+` WHERE status = 'pending' AND (next_attempt_at IS NULL OR next_attempt_at <= ?)
		ORDER BY created_at ASC, id ASC`, formatTime(now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.FetchJob, 0)
	for rows.Next() {
		job, scanErr := scanFetchJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

// ClaimFetchJob claims only if the job remains pending and its backoff elapsed.
// The conditional UPDATE is the serialization point when multiple workers race.
func (r *Repository) ClaimFetchJob(ctx context.Context, id uuid.UUID, now time.Time) (*domain.FetchJob, error) {
	now = now.UTC()
	result, err := r.q.ExecContext(ctx, `UPDATE fetch_jobs SET status = 'running', attempt = attempt + 1,
		next_attempt_at = NULL, updated_at = ? WHERE id = ? AND status = 'pending'
		AND (next_attempt_at IS NULL OR next_attempt_at <= ?)`, formatTime(now), id.String(), formatTime(now))
	if err != nil {
		return nil, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if changed == 0 {
		_, getErr := r.GetFetchJobByID(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		return nil, apperr.FailedPrecondition("fetch job was already claimed or not claimable")
	}
	return r.GetFetchJobByID(ctx, id)
}

func (r *Repository) UpdateFetchJob(ctx context.Context, job *domain.FetchJob) error {
	result, err := r.q.ExecContext(ctx, `UPDATE fetch_jobs SET status = ?, attempt = ?, max_attempts = ?, entries_created = ?,
		next_attempt_at = ?, updated_at = ? WHERE id = ?`, string(job.Status), job.Attempt, job.MaxAttempts,
		job.EntriesCreated, nullableTime(job.NextAttemptAt), formatTime(job.UpdatedAt), job.ID.String())
	if err != nil {
		if database.IsUniqueConstraintViolation(err, "fetch_jobs.source_id") {
			return apperr.Conflict("source already has an unfinished fetch job")
		}
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return apperr.NotFound("fetch job not found")
	}
	return nil
}

const selectFetchJob = `SELECT id, source_id, status, attempt, max_attempts, entries_created,
	next_attempt_at, created_at, updated_at FROM fetch_jobs`

type scannable interface{ Scan(...any) error }

func scanFetchJob(row scannable) (*domain.FetchJob, error) {
	var job domain.FetchJob
	var id, sourceID, status, createdAt, updatedAt string
	var nextAttemptAt sql.NullString
	err := row.Scan(&id, &sourceID, &status, &job.Attempt, &job.MaxAttempts, &job.EntriesCreated, &nextAttemptAt, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("fetch job not found")
	}
	if err != nil {
		return nil, err
	}
	if job.ID, err = uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("parse fetch job id: %w", err)
	}
	if job.SourceID, err = uuid.Parse(sourceID); err != nil {
		return nil, fmt.Errorf("parse fetch job source id: %w", err)
	}
	job.Status = domain.Status(status)
	if job.CreatedAt, err = parseTime(createdAt); err != nil {
		return nil, err
	}
	if job.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return nil, err
	}
	if nextAttemptAt.Valid {
		if job.NextAttemptAt, err = parseTime(nextAttemptAt.String); err != nil {
			return nil, err
		}
	}
	return &job, nil
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return formatTime(t)
}

func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse fetch job timestamp %q: %w", value, err)
	}
	return t.UTC(), nil
}
