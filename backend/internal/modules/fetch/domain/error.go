package domain

import "github.com/larb26656/my-feed/backend/internal/platform/apperr"

// ErrConflict identifies a fetch operation that conflicts with current state.
var ErrConflict = apperr.Definition{Kind: apperr.KindConflict, Code: "CONFLICT"}
