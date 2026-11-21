package domain

import "github.com/larb26656/my-feed/backend/internal/platform/apperr"

var ErrSourceDeactivated = apperr.Definition{
	Kind: apperr.KindConflict,
	Code: "SOURCE_DEACTIVATED",
}
