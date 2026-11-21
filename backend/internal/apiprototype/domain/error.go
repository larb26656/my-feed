package domain

import "github.com/larb26656/my-feed/backend/internal/platform/apperr"

var ErrBookmarkConflict = apperr.Definition{
	Kind: apperr.KindConflict,
	Code: "CONFLICT",
}
