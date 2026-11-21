package apperr

var (
	InvalidInput     = Definition{Kind: KindInvalid, Code: "VALIDATION"}
	ResourceNotFound = Definition{Kind: KindNotFound, Code: "NOT_FOUND"}
	ResourceConflict = Definition{Kind: KindConflict, Code: "CONFLICT"}
)
