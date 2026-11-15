package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/larb26656/quick-feed/backend/internal/fetch/domain"
	"github.com/larb26656/quick-feed/backend/internal/fetch/repository"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

const pgUniqueViolation = "23505"

type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Repository struct {
	q Querier
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: pool}
}

func (r *Repository) WithTx(tx pgx.Tx) repository.Repository {
	return &Repository{q: tx}
}

func (r *Repository) CreateFetchJob(ctx context.Context, job *domain.FetchJob) error {
	var nextAttemptAt *time.Time
	if !job.NextAttemptAt.IsZero() {
		t := job.NextAttemptAt.UTC()
		nextAttemptAt = &t
	}
	_, err := r.q.Exec(ctx, `
		INSERT INTO fetch_jobs (
			id, source_id, status, attempt, max_attempts, entries_created,
			next_attempt_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		job.ID, job.SourceID, string(job.Status), job.Attempt, job.MaxAttempts,
		job.EntriesCreated, nextAttemptAt, job.CreatedAt.UTC(), job.UpdatedAt.UTC(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return apperr.Conflict("source already has an unfinished fetch job")
		}
		return err
	}
	return nil
}

func (r *Repository) GetFetchJobByID(ctx context.Context, id uuid.UUID) (*domain.FetchJob, error) {
	row := r.q.QueryRow(ctx, `
		SELECT id, source_id, status, attempt, max_attempts, entries_created,
			next_attempt_at, created_at, updated_at
		FROM fetch_jobs WHERE id = $1`, id)
	return scanFetchJob(row)
}

func (r *Repository) GetUnfinishedFetchJobBySourceID(ctx context.Context, sourceID uuid.UUID) (*domain.FetchJob, error) {
	row := r.q.QueryRow(ctx, `
		SELECT id, source_id, status, attempt, max_attempts, entries_created,
			next_attempt_at, created_at, updated_at
		FROM fetch_jobs
		WHERE source_id = $1 AND status IN ('pending', 'running')
		LIMIT 1`, sourceID)
	return scanFetchJob(row)
}

func (r *Repository) ListClaimableFetchJobs(ctx context.Context, now time.Time) ([]*domain.FetchJob, error) {
	rows, err := r.q.Query(ctx, `
		SELECT id, source_id, status, attempt, max_attempts, entries_created,
			next_attempt_at, created_at, updated_at
		FROM fetch_jobs
		WHERE status = 'pending'
		  AND (next_attempt_at IS NULL OR next_attempt_at <= $1)
		ORDER BY created_at ASC, id ASC`, now.UTC())
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

func (r *Repository) ClaimFetchJob(ctx context.Context, id uuid.UUID, now time.Time) (*domain.FetchJob, error) {
	job, err := r.GetFetchJobByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := job.Claim(now); err != nil {
		return nil, err
	}

	tag, err := r.q.Exec(ctx, `
		UPDATE fetch_jobs SET
			status = $2,
			attempt = $3,
			next_attempt_at = NULL,
			updated_at = $4
		WHERE id = $1
		  AND status = 'pending'
		  AND (next_attempt_at IS NULL OR next_attempt_at <= $4)`,
		job.ID, string(job.Status), job.Attempt, now.UTC(),
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, apperr.FailedPrecondition("fetch job was already claimed or not claimable")
	}
	return job, nil
}

func (r *Repository) UpdateFetchJob(ctx context.Context, job *domain.FetchJob) error {
	var nextAttemptAt *time.Time
	if !job.NextAttemptAt.IsZero() {
		t := job.NextAttemptAt.UTC()
		nextAttemptAt = &t
	}
	tag, err := r.q.Exec(ctx, `
		UPDATE fetch_jobs SET
			status = $2,
			attempt = $3,
			max_attempts = $4,
			entries_created = $5,
			next_attempt_at = $6,
			updated_at = $7
		WHERE id = $1`,
		job.ID, string(job.Status), job.Attempt, job.MaxAttempts,
		job.EntriesCreated, nextAttemptAt, job.UpdatedAt.UTC(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return apperr.Conflict("source already has an unfinished fetch job")
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("fetch job not found")
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanFetchJob(row scannable) (*domain.FetchJob, error) {
	var (
		job           domain.FetchJob
		status        string
		nextAttemptAt *time.Time
	)
	err := row.Scan(
		&job.ID, &job.SourceID, &status, &job.Attempt, &job.MaxAttempts,
		&job.EntriesCreated, &nextAttemptAt, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("fetch job not found")
		}
		return nil, err
	}
	job.Status = domain.Status(status)
	if nextAttemptAt != nil {
		job.NextAttemptAt = nextAttemptAt.UTC()
	}
	return &job, nil
}
