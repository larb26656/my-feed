package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/source/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

type querier interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Repository struct{ q querier }

func NewRepository(db *sql.DB) *Repository { return &Repository{q: db} }

func (r *Repository) WithTx(tx database.Transaction) repository.Repository {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		panic("sqlite source repository received a non-SQLite transaction")
	}
	return &Repository{q: sqlTx}
}

func (r *Repository) CreateSource(ctx context.Context, s *domain.Source) error {
	var rssURL any
	var channelID any
	if s.YouTubeChannelID != "" {
		channelID = s.YouTubeChannelID
	}
	if s.RSSURL != "" {
		rssURL = s.RSSURL
	}
	_, err := r.q.ExecContext(ctx, `
		INSERT INTO sources (id, name, type, rss_url, youtube_channel_id, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID.String(), s.Name, string(s.Type), rssURL, channelID, s.Active, s.CreatedAt.UTC().UnixNano(), s.UpdatedAt.UTC().UnixNano())
	if database.IsUniqueConstraintViolation(err) {
		if s.Type == domain.TypeManual {
			return domain.ErrConflict.Wrap(err, "manual source already exists")
		}
		if s.Type == domain.TypeYouTube {
			return domain.ErrConflict.Wrap(err, "youtube channel already in use")
		}
		return domain.ErrConflict.Wrap(err, "rss url already in use")
	}
	return err
}

func (r *Repository) GetSourceByID(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	return scanSource(r.q.QueryRowContext(ctx, `SELECT id, name, type, rss_url, youtube_channel_id, active, created_at, updated_at FROM sources WHERE id = ?`, id.String()))
}

func (r *Repository) GetManualSource(ctx context.Context) (*domain.Source, error) {
	return scanSource(r.q.QueryRowContext(ctx, `SELECT id, name, type, rss_url, youtube_channel_id, active, created_at, updated_at FROM sources WHERE type = ? LIMIT 1`, string(domain.TypeManual)))
}

func (r *Repository) ListSources(ctx context.Context, filter repository.ListSourcesFilter) ([]*domain.Source, error) {
	query := `SELECT id, name, type, rss_url, youtube_channel_id, active, created_at, updated_at FROM sources`
	if !filter.IncludeDeactivated {
		query += ` WHERE active = 1`
	}
	query += ` ORDER BY created_at ASC, id ASC`
	rows, err := r.q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
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
	var rssURL any
	if s.RSSURL != "" {
		rssURL = s.RSSURL
	}
	result, err := r.q.ExecContext(ctx, `UPDATE sources SET name = ?, type = ?, rss_url = ?, active = ?, updated_at = ? WHERE id = ?`,
		s.Name, string(s.Type), rssURL, s.Active, s.UpdatedAt.UTC().UnixNano(), s.ID.String())
	if database.IsUniqueConstraintViolation(err) {
		return domain.ErrConflict.Wrap(err, "rss url already in use")
	}
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return apperr.ResourceNotFound.New("source not found")
	}
	return nil
}

type scanner interface{ Scan(...any) error }

func scanSource(row scanner) (*domain.Source, error) {
	var s domain.Source
	var typ, id string
	var rssURL, channelID sql.NullString
	var active bool
	var createdAt, updatedAt int64
	if err := row.Scan(&id, &s.Name, &typ, &rssURL, &channelID, &active, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ResourceNotFound.Wrap(err, "source not found")
		}
		return nil, err
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	s.ID, s.Type, s.Active = parsedID, domain.Type(typ), active
	if rssURL.Valid {
		s.RSSURL = rssURL.String
	}
	if channelID.Valid {
		s.YouTubeChannelID = channelID.String
	}
	s.CreatedAt = unixTime(createdAt)
	s.UpdatedAt = unixTime(updatedAt)
	return &s, nil
}

func unixTime(n int64) time.Time { return time.Unix(0, n).UTC() }
