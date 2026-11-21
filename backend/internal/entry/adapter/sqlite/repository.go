package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/entry/domain"
	"github.com/larb26656/my-feed/backend/internal/entry/repository"
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
		panic("sqlite entry repository received a non-SQLite transaction")
	}
	return &Repository{q: sqlTx}
}

const entryColumns = `id, source_id, title, link, description, thumbnail, published_at, created_at, updated_at, read_at, saved_at`

func (r *Repository) CreateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error) {
	row := r.q.QueryRowContext(ctx, `INSERT INTO entries (`+entryColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING `+entryColumns,
		e.ID.String(), e.SourceID.String(), e.Title, e.Link, e.Description, e.Thumbnail,
		timeArg(e.PublishedAt), e.CreatedAt.UTC().UnixNano(), e.UpdatedAt.UTC().UnixNano(), timeArg(e.ReadAt), timeArg(e.SavedAt))
	created, err := scanEntry(row)
	if database.IsUniqueConstraintViolation(err) {
		return nil, apperr.ResourceConflict.New("entry already exists for source and link")
	}
	return created, err
}

func (r *Repository) GetEntryByID(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
	return scanEntry(r.q.QueryRowContext(ctx, `SELECT `+entryColumns+` FROM entries WHERE id = ?`, id.String()))
}

func (r *Repository) GetEntryBySourceAndLink(ctx context.Context, sourceID uuid.UUID, link string) (*domain.Entry, error) {
	return scanEntry(r.q.QueryRowContext(ctx, `SELECT `+entryColumns+` FROM entries WHERE source_id = ? AND link = ?`, sourceID.String(), link))
}

func (r *Repository) ListEntries(ctx context.Context, after *repository.EntryListCursor, criteria repository.EntryListCriteria, limit int) ([]*domain.Entry, error) {
	query := `SELECT ` + entryColumns + ` FROM entries WHERE 1=1`
	args := []any{}
	if criteria.SourceID != nil {
		query += ` AND source_id = ?`
		args = append(args, criteria.SourceID.String())
	}
	if criteria.SavedOnly {
		query += ` AND saved_at IS NOT NULL`
	}
	if criteria.Unread {
		query += ` AND read_at IS NULL`
	}
	if after != nil {
		query += ` AND (COALESCE(published_at, created_at) < ? OR (COALESCE(published_at, created_at) = ? AND id < ?))`
		sortAt := after.SortAt.UTC().UnixNano()
		args = append(args, sortAt, sortAt, after.ID.String())
	}
	query += ` ORDER BY COALESCE(published_at, created_at) DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := r.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*domain.Entry, 0, limit)
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) CountSavedEntries(ctx context.Context) (int, error) {
	var count int
	err := r.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM entries WHERE saved_at IS NOT NULL`).Scan(&count)
	return count, err
}

func (r *Repository) UpdateEntry(ctx context.Context, e *domain.Entry) (*domain.Entry, error) {
	row := r.q.QueryRowContext(ctx, `UPDATE entries SET title = ?, link = ?, description = ?, thumbnail = ?, published_at = ?, updated_at = ?, read_at = ?, saved_at = ? WHERE id = ? RETURNING `+entryColumns,
		e.Title, e.Link, e.Description, e.Thumbnail, timeArg(e.PublishedAt), e.UpdatedAt.UTC().UnixNano(), timeArg(e.ReadAt), timeArg(e.SavedAt), e.ID.String())
	return scanEntry(row)
}

type scanner interface{ Scan(...any) error }

func scanEntry(row scanner) (*domain.Entry, error) {
	var e domain.Entry
	var id, sourceID string
	var thumbnail sql.NullString
	var publishedAt, readAt, savedAt sql.NullInt64
	var createdAt, updatedAt int64
	err := row.Scan(&id, &sourceID, &e.Title, &e.Link, &e.Description, &thumbnail, &publishedAt, &createdAt, &updatedAt, &readAt, &savedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ResourceNotFound.Wrap(err, "entry not found")
		}
		return nil, err
	}
	var parseErr error
	if e.ID, parseErr = uuid.Parse(id); parseErr != nil {
		return nil, parseErr
	}
	if e.SourceID, parseErr = uuid.Parse(sourceID); parseErr != nil {
		return nil, parseErr
	}
	if thumbnail.Valid {
		e.Thumbnail = &thumbnail.String
	}
	e.PublishedAt = nullableTime(publishedAt)
	e.ReadAt = nullableTime(readAt)
	e.SavedAt = nullableTime(savedAt)
	e.CreatedAt = time.Unix(0, createdAt).UTC()
	e.UpdatedAt = time.Unix(0, updatedAt).UTC()
	return &e, nil
}

func timeArg(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.UTC().UnixNano()
}

func nullableTime(v sql.NullInt64) *time.Time {
	if !v.Valid {
		return nil
	}
	t := time.Unix(0, v.Int64).UTC()
	return &t
}
