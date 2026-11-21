package service_test

import (
	"context"
	"testing"
	"time"

	authsqlite "github.com/larb26656/my-feed/backend/internal/modules/auth/adapter/sqlite"
	"github.com/larb26656/my-feed/backend/internal/modules/auth/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/auth/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/testkit"
	"github.com/stretchr/testify/require"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func TestLoginRejectsBadPassword(t *testing.T) {
	// Arrange
	svc := newAuthService(t, fixedClock{t: time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)})

	// Act
	_, err := svc.Login(context.Background(), service.LoginInput{Username: "admin", Password: "wrong"})

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, domain.ErrInvalidCredentials.Code, failure.Code)
}

func TestLoginAndRefreshRotateTokens(t *testing.T) {
	// Arrange
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	svc := newAuthService(t, fixedClock{t: now})
	first, err := svc.Login(context.Background(), service.LoginInput{Username: "admin", Password: "secret"})
	require.NoError(t, err)

	// Act
	second, err := svc.Refresh(context.Background(), service.RefreshInput{RefreshToken: first.RefreshToken})

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, second.AccessToken)
	require.NotEqual(t, first.RefreshToken, second.RefreshToken)
	require.NotEqual(t, first.AccessToken, second.AccessToken)
}

func TestRefreshReuseRevokesOnlyThatSession(t *testing.T) {
	// Arrange
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	svc := newAuthService(t, fixedClock{t: now})
	deviceA, err := svc.Login(context.Background(), service.LoginInput{Username: "admin", Password: "secret"})
	require.NoError(t, err)
	deviceB, err := svc.Login(context.Background(), service.LoginInput{Username: "admin", Password: "secret"})
	require.NoError(t, err)
	rotatedA, err := svc.Refresh(context.Background(), service.RefreshInput{RefreshToken: deviceA.RefreshToken})
	require.NoError(t, err)

	// Act: reuse old refresh from device A
	_, reuseErr := svc.Refresh(context.Background(), service.RefreshInput{RefreshToken: deviceA.RefreshToken})

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, reuseErr, &failure)
	require.Equal(t, domain.ErrInvalidRefresh.Code, failure.Code)

	_, err = svc.Refresh(context.Background(), service.RefreshInput{RefreshToken: rotatedA.RefreshToken})
	require.Error(t, err)

	stillB, err := svc.Refresh(context.Background(), service.RefreshInput{RefreshToken: deviceB.RefreshToken})
	require.NoError(t, err)
	require.NotEmpty(t, stillB.AccessToken)
}

func TestLogoutBlocksFurtherRefresh(t *testing.T) {
	// Arrange
	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	svc := newAuthService(t, fixedClock{t: now})
	pair, err := svc.Login(context.Background(), service.LoginInput{Username: "admin", Password: "secret"})
	require.NoError(t, err)

	// Act
	require.NoError(t, svc.Logout(context.Background(), service.RefreshInput{RefreshToken: pair.RefreshToken}))
	_, err = svc.Refresh(context.Background(), service.RefreshInput{RefreshToken: pair.RefreshToken})

	// Assert
	var failure *apperr.AppError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, domain.ErrInvalidRefresh.Code, failure.Code)
}

func newAuthService(t *testing.T, clock service.Clock) *service.Service {
	t.Helper()
	pool := testkit.NewDatabase(t)
	repo := authsqlite.NewRepository(pool.DB)
	svc := service.New(repo, pool, service.Config{
		Username:   "admin",
		Password:   "secret",
		JWTSecret:  "jwt-test-secret",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	return svc.WithClock(clock)
}
