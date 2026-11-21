package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/larb26656/my-feed/backend/internal/modules/auth/domain"
	"github.com/stretchr/testify/require"
)

func TestSessionRevokeIsIdempotent(t *testing.T) {
	// Arrange
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	session := domain.NewSession(now)

	// Act
	session.Revoke(now.Add(time.Minute))
	first := *session.RevokedAt
	session.Revoke(now.Add(2 * time.Minute))

	// Assert
	require.True(t, session.Revoked())
	require.Equal(t, first, *session.RevokedAt)
}

func TestRefreshTokenExpiryAndReplace(t *testing.T) {
	// Arrange
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	token := domain.NewRefreshToken(uuid.New(), "hash", now.Add(time.Hour), now)
	next := uuid.New()

	// Act & Assert
	require.True(t, token.Current())
	require.False(t, token.Expired(now))
	require.True(t, token.Expired(now.Add(time.Hour)))

	token.MarkReplaced(next)
	require.False(t, token.Current())
	require.Equal(t, next, *token.ReplacedByID)
}
