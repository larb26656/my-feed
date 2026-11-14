package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
)

type ErrorBody struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, err error) {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		WriteJSON(w, ae.HTTPStatus(), ErrorBody{Error: ae.Message, Code: string(ae.Code)})
		return
	}
	WriteJSON(w, http.StatusInternalServerError, ErrorBody{Error: "internal server error", Code: "internal"})
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apperr.Validation("invalid json body")
	}
	return nil
}
