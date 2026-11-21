package domain

import (
	"time"

	"github.com/google/uuid"
)

// Session is one login of the Reader on a particular client.
type Session struct {
	ID        uuid.UUID
	CreatedAt time.Time
	RevokedAt *time.Time
}

// NewSession starts a new Session at now.
func NewSession(now time.Time) *Session {
	return &Session{
		ID:        uuid.New(),
		CreatedAt: now.UTC(),
	}
}

// Revoke marks the Session revoked at now. Idempotent when already revoked.
func (s *Session) Revoke(now time.Time) {
	if s.RevokedAt != nil {
		return
	}
	t := now.UTC()
	s.RevokedAt = &t
}

// Revoked reports whether the Session has been revoked.
func (s *Session) Revoked() bool {
	return s.RevokedAt != nil
}

// RefreshToken is one opaque refresh credential belonging to a Session.
type RefreshToken struct {
	ID           uuid.UUID
	SessionID    uuid.UUID
	TokenHash    string
	ExpiresAt    time.Time
	ReplacedByID *uuid.UUID
	CreatedAt    time.Time
}

// NewRefreshToken creates a current refresh token row for a Session.
func NewRefreshToken(sessionID uuid.UUID, tokenHash string, expiresAt, now time.Time) *RefreshToken {
	return &RefreshToken{
		ID:        uuid.New(),
		SessionID: sessionID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt.UTC(),
		CreatedAt: now.UTC(),
	}
}

// Current reports whether this refresh token is still the active one.
func (t *RefreshToken) Current() bool {
	return t.ReplacedByID == nil
}

// MarkReplaced records that this token was rotated to nextID.
func (t *RefreshToken) MarkReplaced(nextID uuid.UUID) {
	id := nextID
	t.ReplacedByID = &id
}

// Expired reports whether the refresh token is past expiresAt at now.
func (t *RefreshToken) Expired(now time.Time) bool {
	return !now.UTC().Before(t.ExpiresAt)
}
