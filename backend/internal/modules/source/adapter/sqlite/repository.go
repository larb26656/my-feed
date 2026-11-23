package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"modernc.org/sqlite"

	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/source/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

func init() {
	sqlite.MustRegisterDeterministicScalarFunction("source_name_contains", 2, func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		name, okName := args[0].(string)
		needle, okNeedle := args[1].(string)
		if !okName || !okNeedle {
			return int64(0), nil
		}
		text, search := []rune(name), []rune(needle)
		for i := 0; i+len(search) <= len(text); i++ {
			if strings.EqualFold(string(text[i:i+len(search)]), needle) {
				return int64(1), nil
			}
		}
		return int64(0), nil
	})
}

type querier interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (r *Repository) ListSourcesPage(ctx context.Context, q repository.SourcePageQuery) ([]*domain.Source, int64, error) {
	where := []string{"1=1"}
	args := []any{}
	if q.NameLike != "" {
		where = append(where, `source_name_contains(name, ?) = 1`)
		args = append(args, q.NameLike)
	}
	if q.TypeEqual != "" {
		where = append(where, "type = ?")
		args = append(args, q.TypeEqual)
	}
	if q.ActiveEqual != nil {
		where = append(where, "active = ?")
		args = append(args, *q.ActiveEqual)
	}
	clause := " WHERE " + strings.Join(where, " AND ")
	columns := map[string]string{"name": "name", "type": "type", "active": "active", "createdAt": "created_at"}
	column, ok := columns[q.SortBy]
	if !ok {
		return nil, 0, fmt.Errorf("invalid sort field")
	}
	direction := "ASC"
	if q.SortDirection == "desc" {
		direction = "DESC"
	} else if q.SortDirection != "asc" {
		return nil, 0, fmt.Errorf("invalid sort direction")
	}
	query := `WITH filtered AS (SELECT * FROM sources` + clause + `),
		page AS (SELECT * FROM filtered ORDER BY ` + column + " " + direction + `, id ASC LIMIT ? OFFSET ?)
		SELECT page.id, page.name, page.type, COALESCE(json_extract(page.config, '$.url'), page.rss_url), COALESCE(json_extract(page.config, '$.channelId'), page.youtube_channel_id),
		page.active, page.created_at, page.updated_at, page.profile_image_id, page.config, page.key, (SELECT count(*) FROM filtered)
		FROM (SELECT 1) LEFT JOIN page ON 1=1 ORDER BY page.` + column + " " + direction + ", page.id ASC"
	offset := math.MaxInt
	if q.Page-1 <= math.MaxInt/q.PageSize {
		offset = (q.Page - 1) * q.PageSize
	}
	args = append(args, q.PageSize, offset)
	rows, err := r.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()
	out := make([]*domain.Source, 0)
	var total int64
	for rows.Next() {
		var id, name, typ, rssURL, channelID, imageID, config, key sql.NullString
		var active sql.NullBool
		var createdAt, updatedAt sql.NullInt64
		if err := rows.Scan(&id, &name, &typ, &rssURL, &channelID, &active, &createdAt, &updatedAt, &imageID, &config, &key, &total); err != nil {
			return nil, 0, err
		}
		if !id.Valid {
			continue
		}
		parsedID, err := uuid.Parse(id.String)
		if err != nil {
			return nil, 0, err
		}
		var decoded map[string]any
		if err := json.Unmarshal([]byte(config.String), &decoded); err != nil {
			return nil, 0, err
		}
		out = append(out, &domain.Source{Config: decoded, Key: key.String, ProfileImageID: imageID.String, ID: parsedID, Name: name.String, Type: domain.Type(typ.String), RSSURL: rssURL.String, YouTubeChannelID: channelID.String, Active: active.Bool, CreatedAt: unixTime(createdAt.Int64), UpdatedAt: unixTime(updatedAt.Int64)})
	}
	return out, total, rows.Err()
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
	config, key := sourceConfig(s)
	_, err := r.q.ExecContext(ctx, `
		INSERT INTO sources (id, name, type, rss_url, youtube_channel_id, config, key, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID.String(), s.Name, string(s.Type), rssURL, channelID, config, key, s.Active, s.CreatedAt.UTC().UnixNano(), s.UpdatedAt.UTC().UnixNano())
	if database.IsUniqueConstraintViolation(err) {
		return uniqueConflict(err, s.Type)
	}
	return err
}

func (r *Repository) GetSourceByID(ctx context.Context, id uuid.UUID) (*domain.Source, error) {
	return scanSource(r.q.QueryRowContext(ctx, `SELECT id, name, type, COALESCE(json_extract(config, '$.url'), rss_url), COALESCE(json_extract(config, '$.channelId'), youtube_channel_id), active, created_at, updated_at, profile_image_id, config, key FROM sources WHERE id = ?`, id.String()))
}

func (r *Repository) GetManualSource(ctx context.Context) (*domain.Source, error) {
	return scanSource(r.q.QueryRowContext(ctx, `SELECT id, name, type, COALESCE(json_extract(config, '$.url'), rss_url), COALESCE(json_extract(config, '$.channelId'), youtube_channel_id), active, created_at, updated_at, profile_image_id, config, key FROM sources WHERE type = ? LIMIT 1`, string(domain.TypeManual)))
}

func (r *Repository) ListSources(ctx context.Context, filter repository.ListSourcesFilter) ([]*domain.Source, error) {
	query := `SELECT id, name, type, COALESCE(json_extract(config, '$.url'), rss_url), COALESCE(json_extract(config, '$.channelId'), youtube_channel_id), active, created_at, updated_at, profile_image_id, config, key FROM sources`
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
	var rssURL, channelID any
	if s.RSSURL != "" {
		rssURL = s.RSSURL
	}
	if s.YouTubeChannelID != "" {
		channelID = s.YouTubeChannelID
	}
	config, key := sourceConfig(s)
	result, err := r.q.ExecContext(ctx, `UPDATE sources SET name = ?, type = ?, rss_url = ?, youtube_channel_id = ?, config = ?, key = ?, active = ?, updated_at = ? WHERE id = ?`,
		s.Name, string(s.Type), rssURL, channelID, config, key, s.Active, s.UpdatedAt.UTC().UnixNano(), s.ID.String())
	if database.IsUniqueConstraintViolation(err) {
		return uniqueConflict(err, s.Type)
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
	var rssURL, channelID, imageID, config, key sql.NullString
	var active bool
	var createdAt, updatedAt int64
	if err := row.Scan(&id, &s.Name, &typ, &rssURL, &channelID, &active, &createdAt, &updatedAt, &imageID, &config, &key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ResourceNotFound.Wrap(err, "source not found")
		}
		return nil, err
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(config.String), &s.Config); err != nil {
		return nil, err
	}
	s.Key = key.String
	s.ID, s.Type, s.Active = parsedID, domain.Type(typ), active
	s.ProfileImageID = imageID.String
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

func (r *Repository) SetProfileImage(ctx context.Context, id uuid.UUID, imageID string) error {
	var value any
	if imageID != "" {
		value = imageID
	}
	result, err := r.q.ExecContext(ctx, `UPDATE sources SET profile_image_id = ? WHERE id = ?`, value, id.String())
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

func sourceConfig(s *domain.Source) (string, string) {
	encoded, _ := json.Marshal(s.Config)
	return string(encoded), s.Key
}

func uniqueConflict(err error, typ domain.Type) error {
	switch typ {
	case domain.TypeManual:
		return apperr.ResourceConflict.Wrap(err, "manual source already exists")
	case domain.TypeYouTube:
		return apperr.ResourceConflict.Wrap(err, "youtube channel already in use")
	case domain.TypeMedium:
		return apperr.ResourceConflict.Wrap(err, "medium source already in use")
	case domain.TypeBluesky:
		return apperr.ResourceConflict.Wrap(err, "bluesky profile already in use")
	default:
		return apperr.ResourceConflict.Wrap(err, "rss url already in use")
	}
}
