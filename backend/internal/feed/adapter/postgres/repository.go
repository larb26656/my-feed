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

const (
	pgUniqueViolation = "23505"
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

func (r *Repository) CreateSource(ctx context.Context, s *domain.Source) error {
	_, err := r.q.Exec(ctx, `
		INSERT INTO sources (id, name, created_at, updated_at)
		VALUES ($1,$2,$3,$4)`,
		s.ID, s.Name, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *Repository) GetSourceByID(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	row := r.q.QueryRow(ctx, `
		SELECT id, name, created_at, updated_at
		FROM sources WHERE id = $1`, id)
	return scanSource(row)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSource(row scannable) (*domain.Source, error) {
	var s domain.Source
	err := row.Scan(&s.ID, &s.Name, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("source not found")
		}
		return nil, err
	}
	return &s, nil
}

func (r *Repository) CreateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error) {
	row := r.q.QueryRow(ctx, `
		INSERT INTO entries (
			id, source_id, title, link, description, thumbnail,
			published_at, created_at, updated_at, read_at, saved_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, source_id, title, link, description, thumbnail,
			published_at, created_at, updated_at, read_at, saved_at
		`,
		e.ID, e.SourceID, e.Title, e.Link, e.Description, e.Thumbnail,
		e.PublishedAt, e.CreatedAt, e.UpdatedAt, e.ReadAt, e.SavedAt,
	)
	persisted, err := scanEntry(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, apperr.Conflict("entry already exists for source and link")
		}
		return nil, err
	}
	return persisted, nil
}

func (r *Repository) GetEntryByID(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
	row := r.q.QueryRow(ctx, `
		SELECT id, source_id, title, link, description, thumbnail,
			published_at, created_at, updated_at, read_at, saved_at
		FROM entries WHERE id = $1`, id)
	return scanEntry(row)
}

func (r *Repository) GetEntryBySourceAndLink(ctx context.Context, sourceID uuid.UUID, link string) (*domain.Entry, error) {
	row := r.q.QueryRow(ctx, `
		SELECT id, source_id, title, link, description, thumbnail,
			published_at, created_at, updated_at, read_at, saved_at
		FROM entries WHERE source_id = $1 AND link = $2`, sourceID, link)
	return scanEntry(row)
}

func (r *Repository) ListEntries(ctx context.Context, after *repository.EntryListCursor, limit int) ([]*domain.Entry, error) {
	var (
		rows pgx.Rows
		err  error
	)
	const cols = `
		SELECT id, source_id, title, link, description, thumbnail,
			published_at, created_at, updated_at, read_at, saved_at
		FROM entries`
	if after == nil {
		rows, err = r.q.Query(ctx, cols+`
			ORDER BY COALESCE(published_at, created_at) DESC, id DESC
			LIMIT $1`, limit)
	} else {
		rows, err = r.q.Query(ctx, cols+`
			WHERE COALESCE(published_at, created_at) < $1
			   OR (COALESCE(published_at, created_at) = $1 AND id < $2)
			ORDER BY COALESCE(published_at, created_at) DESC, id DESC
			LIMIT $3`, after.SortAt.UTC(), after.ID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*domain.Entry, 0, limit)
	for rows.Next() {
		e, scanErr := scanEntry(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error) {
	row := r.q.QueryRow(ctx, `
		UPDATE entries SET
			title = $2,
			link = $3,
			description = $4,
			thumbnail = $5,
			published_at = $6,
			updated_at = $7,
			read_at = $8,
			saved_at = $9
		WHERE id = $1
		RETURNING id, source_id, title, link, description, thumbnail,
			published_at, created_at, updated_at, read_at, saved_at
		`,
		e.ID, e.Title, e.Link, e.Description, e.Thumbnail,
		e.PublishedAt, e.UpdatedAt, e.ReadAt, e.SavedAt,
	)
	return scanEntry(row)
}

func scanEntry(row scannable) (*domain.Entry, error) {
	var e domain.Entry
	err := row.Scan(
		&e.ID, &e.SourceID, &e.Title, &e.Link, &e.Description, &e.Thumbnail,
		&e.PublishedAt, &e.CreatedAt, &e.UpdatedAt, &e.ReadAt, &e.SavedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("entry not found")
		}
		return nil, err
	}
	return &e, nil
}
