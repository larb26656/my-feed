package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/modules/auth/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/auth/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

type querier interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Repository stores auth Sessions in SQLite.
type Repository struct {
	q querier
}

func NewRepository(db *sql.DB) *Repository { return &Repository{q: db} }

func (r *Repository) WithTx(tx database.Transaction) repository.Repository {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		panic("sqlite auth repository received a non-SQLite transaction")
	}
	return &Repository{q: sqlTx}
}

func (r *Repository) CreateSession(ctx context.Context, session *domain.Session) error {
	_, err := r.q.ExecContext(ctx,
		`INSERT INTO auth_sessions (id, created_at, revoked_at) VALUES (?, ?, NULL)`,
		session.ID.String(), session.CreatedAt.UTC().UnixNano())
	if err != nil {
		return fmt.Errorf("create auth session: %w", err)
	}
	return nil
}

func (r *Repository) GetSession(ctx context.Context, id uuid.UUID) (*domain.Session, error) {
	var (
		rawID     string
		createdAt int64
		revokedAt sql.NullInt64
	)
	err := r.q.QueryRowContext(ctx,
		`SELECT id, created_at, revoked_at FROM auth_sessions WHERE id = ?`,
		id.String()).Scan(&rawID, &createdAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ResourceNotFound.Wrap(err, "Session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get auth session: %w", err)
	}
	sessionID, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("parse auth session id: %w", err)
	}
	session := &domain.Session{
		ID:        sessionID,
		CreatedAt: unixTime(createdAt),
	}
	if revokedAt.Valid {
		t := unixTime(revokedAt.Int64)
		session.RevokedAt = &t
	}
	return session, nil
}

func (r *Repository) RevokeSession(ctx context.Context, session *domain.Session) error {
	if session.RevokedAt == nil {
		return fmt.Errorf("revoke auth session: revoked_at is required")
	}
	res, err := r.q.ExecContext(ctx,
		`UPDATE auth_sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`,
		session.RevokedAt.UTC().UnixNano(), session.ID.String())
	if err != nil {
		return fmt.Errorf("revoke auth session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke auth session rows: %w", err)
	}
	if n == 0 {
		// Already revoked or missing — treat as success for idempotent logout/reuse paths.
		return nil
	}
	return nil
}

func (r *Repository) CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	_, err := r.q.ExecContext(ctx,
		`INSERT INTO auth_refresh_tokens (id, session_id, token_hash, expires_at, replaced_by_id, created_at)
		 VALUES (?, ?, ?, ?, NULL, ?)`,
		token.ID.String(), token.SessionID.String(), token.TokenHash,
		token.ExpiresAt.UTC().UnixNano(), token.CreatedAt.UTC().UnixNano())
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	var (
		rawID, rawSessionID string
		tokenHashOut        string
		expiresAt           int64
		replacedBy          sql.NullString
		createdAt           int64
	)
	err := r.q.QueryRowContext(ctx,
		`SELECT id, session_id, token_hash, expires_at, replaced_by_id, created_at
		 FROM auth_refresh_tokens WHERE token_hash = ?`,
		tokenHash).Scan(&rawID, &rawSessionID, &tokenHashOut, &expiresAt, &replacedBy, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.ResourceNotFound.Wrap(err, "Refresh token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token id: %w", err)
	}
	sessionID, err := uuid.Parse(rawSessionID)
	if err != nil {
		return nil, fmt.Errorf("parse refresh token session id: %w", err)
	}
	token := &domain.RefreshToken{
		ID:        id,
		SessionID: sessionID,
		TokenHash: tokenHashOut,
		ExpiresAt: unixTime(expiresAt),
		CreatedAt: unixTime(createdAt),
	}
	if replacedBy.Valid && replacedBy.String != "" {
		replacedID, err := uuid.Parse(replacedBy.String)
		if err != nil {
			return nil, fmt.Errorf("parse replaced_by id: %w", err)
		}
		token.ReplacedByID = &replacedID
	}
	return token, nil
}

func (r *Repository) MarkRefreshTokenReplaced(ctx context.Context, token *domain.RefreshToken) error {
	if token.ReplacedByID == nil {
		return fmt.Errorf("mark refresh token replaced: replaced_by_id is required")
	}
	res, err := r.q.ExecContext(ctx,
		`UPDATE auth_refresh_tokens SET replaced_by_id = ? WHERE id = ? AND replaced_by_id IS NULL`,
		token.ReplacedByID.String(), token.ID.String())
	if err != nil {
		return fmt.Errorf("mark refresh token replaced: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark refresh token replaced rows: %w", err)
	}
	if n == 0 {
		return apperr.ResourceConflict.New("Refresh token already replaced")
	}
	return nil
}

func unixTime(n int64) time.Time { return time.Unix(0, n).UTC() }
