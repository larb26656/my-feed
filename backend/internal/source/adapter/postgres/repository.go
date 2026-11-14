package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
	"github.com/larb26656/quick-feed/backend/internal/source/domain"
	"github.com/larb26656/quick-feed/backend/internal/source/repository"
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

func (r *Repository) CreateSource(ctx context.Context, s *domain.Source) error {
	var rssURL *string
	if s.RSSURL != "" {
		rssURL = &s.RSSURL
	}
	_, err := r.q.Exec(ctx, `
		INSERT INTO sources (id, name, type, rss_url, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		s.ID, s.Name, string(s.Type), rssURL, s.Active, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			if s.Type == domain.TypeManual {
				return apperr.Conflict("manual source already exists")
			}
			return apperr.Conflict("rss url already in use")
		}
		return err
	}
	return nil
}

func (r *Repository) GetSourceByID(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	row := r.q.QueryRow(ctx, `
		SELECT id, name, type, rss_url, active, created_at, updated_at
		FROM sources WHERE id = $1`, id)
	return scanSource(row)
}

func (r *Repository) GetManualSource(ctx context.Context) (*domain.Source, error) {
	row := r.q.QueryRow(ctx, `
		SELECT id, name, type, rss_url, active, created_at, updated_at
		FROM sources WHERE type = $1
		LIMIT 1`, string(domain.TypeManual))
	return scanSource(row)
}

func (r *Repository) ListSources(ctx context.Context, filter repository.ListSourcesFilter) ([]*domain.Source, error) {
	query := `
		SELECT id, name, type, rss_url, active, created_at, updated_at
		FROM sources`
	if !filter.IncludeDeactivated {
		query += ` WHERE active = TRUE`
	}
	query += ` ORDER BY created_at ASC, id ASC`

	rows, err := r.q.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Source
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateSource(ctx context.Context, s *domain.Source) error {
	var rssURL *string
	if s.RSSURL != "" {
		rssURL = &s.RSSURL
	}
	tag, err := r.q.Exec(ctx, `
		UPDATE sources
		SET name = $2, type = $3, rss_url = $4, active = $5, updated_at = $6
		WHERE id = $1`,
		s.ID, s.Name, string(s.Type), rssURL, s.Active, s.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return apperr.Conflict("rss url already in use")
		}
		return err
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("source not found")
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSource(row scannable) (*domain.Source, error) {
	var s domain.Source
	var typ string
	var rssURL *string
	err := row.Scan(&s.ID, &s.Name, &typ, &rssURL, &s.Active, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("source not found")
		}
		return nil, err
	}
	s.Type = domain.Type(typ)
	if rssURL != nil {
		s.RSSURL = *rssURL
	}
	return &s, nil
}
