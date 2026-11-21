package domain

import "github.com/larb26656/my-feed/backend/internal/platform/apperr"

// ErrInvalidCredentials identifies a failed login attempt.
var ErrInvalidCredentials = apperr.Definition{Kind: apperr.KindUnauthorized, Code: "INVALID_CREDENTIALS"}

// ErrInvalidRefresh identifies a missing, expired, revoked, or reused refresh token.
var ErrInvalidRefresh = apperr.Definition{Kind: apperr.KindUnauthorized, Code: "INVALID_REFRESH"}
