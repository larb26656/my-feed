package domain

import "github.com/larb26656/my-feed/backend/internal/platform/apperr"

// ErrConflict identifies a Source operation that conflicts with existing state.
var ErrConflict = apperr.Definition{Kind: apperr.KindConflict, Code: "CONFLICT"}

// ErrManualSourceImmutable identifies an operation that cannot change a manual Source.
var ErrManualSourceImmutable = apperr.Definition{Kind: apperr.KindConflict, Code: "MANUAL_SOURCE_IMMUTABLE"}
