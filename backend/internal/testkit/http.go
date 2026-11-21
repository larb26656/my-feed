package testkit

import (
	"bytes"
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// Request sends a JSON request through a real handler and checks the status.
// A string body is sent verbatim so malformed JSON can also be tested.
func Request(t testing.TB, handler http.Handler, method, path string, body any, status int) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if value, ok := body.(string); ok {
		raw = []byte(value)
	} else if body != nil {
		var err error
		raw, err = json.Marshal(body)
		require.NoError(t, err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	require.Equal(t, status, response.Code, "%s %s: %s", method, path, response.Body.String())
	mediaType, _, err := mime.ParseMediaType(response.Header().Get("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "application/json", mediaType)
	return response
}

func Decode[T any](t testing.TB, response *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result), response.Body.String())
	return result
}
