package httpx

import (
	"net/http"
	"net/url"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

var queryValidator = validator.New()

// DecodeQuery binds scalar query parameters using schema tags and validates
// validate tags. Unknown keys, repeated parameters and malformed queries fail.
// Omitted pointer fields remain nil; explicitly empty values are validated.
func DecodeQuery[T any](r *http.Request) (T, error) {
	var query T
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return query, apperr.Validation("invalid query")
	}
	for key, entries := range values {
		if len(entries) != 1 {
			return query, apperr.Validation("query parameter must appear once: " + key)
		}
	}
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(false)
	decoder.ZeroEmpty(true)
	if err := decoder.Decode(&query, values); err != nil {
		return query, apperr.Validation("invalid query")
	}
	if err := queryValidator.Struct(query); err != nil {
		return query, apperr.Validation("invalid query")
	}
	return query, nil
}
