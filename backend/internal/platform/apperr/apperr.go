package apperr

type Kind uint8

const (
	KindUnknown Kind = iota
	KindNotFound
	KindConflict
	KindInvalid
	KindUnauthorized
	KindForbidden
	KindRateLimited
	KindUnavailable
	KindInternal
)

type Definition struct {
	Kind Kind
	Code string
}

var InvalidInput = Definition{Kind: KindInvalid, Code: "validation"}
var ResourceNotFound = Definition{Kind: KindNotFound, Code: "NOT_FOUND"}

type AppError struct {
	Definition
	Message string
	Detail  any
	Cause   error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func (e *AppError) WithDetail(detail any) *AppError {
	e.Detail = detail
	return e
}

func (d Definition) New(message string) *AppError {
	return &AppError{
		Definition: d,
		Message:    message,
	}
}

func (d Definition) Wrap(err error, message string) *AppError {
	return &AppError{
		Definition: d,
		Message:    message,
		Cause:      err,
	}
}
