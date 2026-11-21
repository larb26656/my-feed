package httpx

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
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
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		switch appErr.Kind {
		case apperr.KindInvalid:
			WriteJSON(w, http.StatusBadRequest, ErrorBody{Error: appErr.Message, Code: appErr.Code})
			return
		case apperr.KindNotFound:
			WriteJSON(w, http.StatusNotFound, ErrorBody{Error: appErr.Message, Code: appErr.Code})
			return
		}
	}
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

// CursorResponse is the common envelope for cursor-paginated lists.
type CursorResponse[T any] struct {
	Data []T        `json:"data"`
	Page CursorPage `json:"page"`
}

// CursorPage describes continuation of a cursor-paginated list.
type CursorPage struct {
	NextCursor *string `json:"nextCursor"`
	HasNext    bool    `json:"hasNext"`
}
