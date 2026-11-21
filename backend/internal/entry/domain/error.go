package domain

import "github.com/larb26656/my-feed/backend/internal/platform/apperr"

var (
	ErrEntryNotFound = apperr.Definition{
		Kind: apperr.KindNotFound,
		Code: "ENTRY_NOT_FOUND",
	}
)
