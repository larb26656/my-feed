package apperr

var InvalidInput = Definition{Kind: KindInvalid, Code: "VALIDATION"}
var ResourceNotFound = Definition{Kind: KindNotFound, Code: "NOT_FOUND"}
var ResourceConflict = Definition{Kind: KindConflict, Code: "CONFLICT"}
