package domain

import "github.com/larb26656/my-feed/backend/internal/platform/apperr"

// ErrManualSourceImmutable identifies an operation that cannot change a manual Source.
var ErrManualSourceImmutable = apperr.Definition{Kind: apperr.KindConflict, Code: "MANUAL_SOURCE_IMMUTABLE"}
