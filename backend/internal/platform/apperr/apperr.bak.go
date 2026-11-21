package apperr

import "net/http"

type Code string

const (
	CodeNotFound           Code = "not_found"
	CodeConflict           Code = "conflict"
	CodeValidation         Code = "validation"
	CodeFailedPrecondition Code = "failed_precondition"
	CodeIdempotencyConflict Code = "idempotency_conflict"
)

type Error struct {
	Code    Code
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) HTTPStatus() int {
	switch e.Code {
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict, CodeIdempotencyConflict:
		return http.StatusConflict
	case CodeValidation:
		return http.StatusBadRequest
	case CodeFailedPrecondition:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func NotFound(msg string) *Error {
	return &Error{Code: CodeNotFound, Message: msg}
}

func Conflict(msg string) *Error {
	return &Error{Code: CodeConflict, Message: msg}
}

func Validation(msg string) *Error {
	return &Error{Code: CodeValidation, Message: msg}
}

func FailedPrecondition(msg string) *Error {
	return &Error{Code: CodeFailedPrecondition, Message: msg}
}
