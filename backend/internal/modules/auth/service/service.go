package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/larb26656/my-feed/backend/internal/modules/auth/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/auth/repository"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
)

// Config is the Reader credential and token lifetime settings.
type Config struct {
	Username   string
	Password   string
	JWTSecret  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// Clock supplies the current time for token expiry and Session timestamps.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// TokenPair is the access and refresh credentials returned after login/refresh.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

// LoginInput is the Reader credential submitted at login.
type LoginInput struct {
	Username string
	Password string
}

// RefreshInput carries the opaque refresh token for rotation or logout.
type RefreshInput struct {
	RefreshToken string
}

// Service handles login, refresh rotation, and logout for Reader Sessions.
type Service struct {
	repo  repository.Repository
	db    database.Transactor
	cfg   Config
	clock Clock
}

func New(repo repository.Repository, db database.Transactor, cfg Config) *Service {
	return &Service{repo: repo, db: db, cfg: cfg, clock: realClock{}}
}

// WithClock overrides the clock (tests).
func (s *Service) WithClock(clock Clock) *Service {
	s.clock = clock
	return s
}

// Login verifies env credentials and starts a new Session.
func (s *Service) Login(ctx context.Context, in LoginInput) (*TokenPair, error) {
	if !constantEqual(in.Username, s.cfg.Username) || !constantEqual(in.Password, s.cfg.Password) {
		return nil, domain.ErrInvalidCredentials.New("invalid username or password")
	}

	now := s.clock.Now()
	session := domain.NewSession(now)
	rawRefresh, hash, err := newRefreshMaterial()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refresh := domain.NewRefreshToken(session.ID, hash, now.Add(s.cfg.RefreshTTL), now)

	err = s.db.WithinTransaction(ctx, func(ctx context.Context, tx database.Transaction) error {
		repo := s.repo.WithTx(tx)
		if err := repo.CreateSession(ctx, session); err != nil {
			return err
		}
		return repo.CreateRefreshToken(ctx, refresh)
	})
	if err != nil {
		return nil, fmt.Errorf("persist login session: %w", err)
	}

	access, err := s.issueAccess(in.Username, now)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		ExpiresIn:    s.cfg.AccessTTL,
	}, nil
}

// Refresh rotates the refresh token for a Session and issues a new access token.
func (s *Service) Refresh(ctx context.Context, in RefreshInput) (*TokenPair, error) {
	if in.RefreshToken == "" {
		return nil, domain.ErrInvalidRefresh.New("refresh token is required")
	}
	now := s.clock.Now()
	hash := hashRefresh(in.RefreshToken)

	var (
		pair          *TokenPair
		reuseDetected bool
	)
	err := s.db.WithinTransaction(ctx, func(ctx context.Context, tx database.Transaction) error {
		repo := s.repo.WithTx(tx)
		token, err := repo.GetRefreshTokenByHash(ctx, hash)
		if err != nil {
			if isNotFound(err) {
				return domain.ErrInvalidRefresh.New("invalid refresh token")
			}
			return err
		}

		session, err := repo.GetSession(ctx, token.SessionID)
		if err != nil {
			if isNotFound(err) {
				return domain.ErrInvalidRefresh.New("invalid refresh token")
			}
			return err
		}

		if session.Revoked() {
			return domain.ErrInvalidRefresh.New("invalid refresh token")
		}

		if !token.Current() {
			// Commit the revoke, then surface reuse to the caller outside the tx
			// so WithinTransaction does not roll the revoke back.
			session.Revoke(now)
			if err := repo.RevokeSession(ctx, session); err != nil {
				return err
			}
			reuseDetected = true
			return nil
		}

		if token.Expired(now) {
			return domain.ErrInvalidRefresh.New("refresh token expired")
		}

		rawNext, nextHash, err := newRefreshMaterial()
		if err != nil {
			return fmt.Errorf("generate refresh token: %w", err)
		}
		next := domain.NewRefreshToken(session.ID, nextHash, now.Add(s.cfg.RefreshTTL), now)
		if err := repo.CreateRefreshToken(ctx, next); err != nil {
			return err
		}
		token.MarkReplaced(next.ID)
		if err := repo.MarkRefreshTokenReplaced(ctx, token); err != nil {
			return err
		}

		access, err := s.issueAccess(s.cfg.Username, now)
		if err != nil {
			return err
		}
		pair = &TokenPair{
			AccessToken:  access,
			RefreshToken: rawNext,
			ExpiresIn:    s.cfg.AccessTTL,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if reuseDetected {
		return nil, domain.ErrInvalidRefresh.New("refresh token reuse detected")
	}
	return pair, nil
}

// Logout revokes the Session that owns the given refresh token.
func (s *Service) Logout(ctx context.Context, in RefreshInput) error {
	if in.RefreshToken == "" {
		return domain.ErrInvalidRefresh.New("refresh token is required")
	}
	now := s.clock.Now()
	hash := hashRefresh(in.RefreshToken)

	return s.db.WithinTransaction(ctx, func(ctx context.Context, tx database.Transaction) error {
		repo := s.repo.WithTx(tx)
		token, err := repo.GetRefreshTokenByHash(ctx, hash)
		if err != nil {
			if isNotFound(err) {
				return domain.ErrInvalidRefresh.New("invalid refresh token")
			}
			return err
		}
		session, err := repo.GetSession(ctx, token.SessionID)
		if err != nil {
			if isNotFound(err) {
				return domain.ErrInvalidRefresh.New("invalid refresh token")
			}
			return err
		}
		if session.Revoked() {
			return nil
		}
		session.Revoke(now)
		return repo.RevokeSession(ctx, session)
	})
}

func (s *Service) issueAccess(username string, now time.Time) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   username,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTTL)),
		ID:        uuid.NewString(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

func constantEqual(a, b string) bool {
	if len(a) != len(b) {
		// Still compare to keep timing closer when lengths differ.
		subtle.ConstantTimeCompare([]byte(a), []byte(a))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func newRefreshMaterial() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashRefresh(raw), nil
}

func hashRefresh(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func isNotFound(err error) bool {
	var app *apperr.AppError
	return errors.As(err, &app) && app.Code == apperr.ResourceNotFound.Code
}
