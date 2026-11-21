package httpx

import (
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type validationField struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
}

// InvalidRequest converts a Gin binding error into a client-safe application error.
func InvalidRequest(err error) *apperr.AppError {
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return apperr.InvalidInput.Wrap(err, "Invalid JSON body")
	}

	fields := make([]validationField, 0, len(validationErrors))
	for _, field := range validationErrors {
		name := field.Namespace()
		if _, path, ok := strings.Cut(name, "."); ok {
			name = path
		}
		fields = append(fields, validationField{Field: name, Rule: field.Tag()})
	}

	return apperr.InvalidInput.Wrap(err, "Validation failed").WithDetail(struct {
		Fields []validationField `json:"fields"`
	}{Fields: fields})
}
