package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/larb26656/quick-feed/backend/internal/feed/domain"
	"github.com/larb26656/quick-feed/backend/internal/feed/repository"
	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

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

func (r *Repository) Create(ctx context.Context, f *domain.Feed) error {
	_, err := r.q.Exec(ctx, `
		INSERT INTO feeds (id, slug, title, description, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		f.ID, f.Slug, f.Title, f.Description, f.Status, f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("slug already exists")
		}
		return err
	}
	return nil
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Feed, error) {
	row := r.q.QueryRow(ctx, `
		SELECT id, slug, title, description, status, created_at, updated_at
		FROM feeds WHERE id = $1`, id)
	return scanFeed(row)
}

func (r *Repository) Update(ctx context.Context, f *domain.Feed) error {
	tag, err := r.q.Exec(ctx, `
		UPDATE feeds
		SET slug=$2, title=$3, description=$4, status=$5, updated_at=$6
		WHERE id=$1`,
		f.ID, f.Slug, f.Title, f.Description, f.Status, f.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("slug already exists")
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("feed not found")
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanFeed(row scannable) (*domain.Feed, error) {
	var f domain.Feed
	err := row.Scan(&f.ID, &f.Slug, &f.Title, &f.Description, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("feed not found")
		}
		return nil, err
	}
	return &f, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
