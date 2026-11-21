package api_test

import (
	"net/http/httptest"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/testkit/api"
	"github.com/stretchr/testify/require"
)

func TestRoutingContractCapture(t *testing.T) {
	// Arrange: each case describes one routing outcome.
	cases := []struct {
		name, method, path string
		status             int
		body               string
		allow              []string
	}{
		{"entry collection", "GET", "/api/v1/entries", 200, "{\"success\":true,\"data\":[],\"meta\":{\"pageSize\":10}}", nil},
		{"entry collection with trailing slash", "GET", "/api/v1/entries/", 200, "{\"success\":true,\"data\":[],\"meta\":{\"pageSize\":10}}", nil},
		{"saved count with trailing slash is missing", "GET", "/api/v1/entries/saved-count/", 404, "404 page not found\n", nil},
		{"encoded saved count is treated as an entry id", "GET", "/api/v1/entries/%73aved-count", 400, "{\"success\":false,\"error\":{\"code\":\"validation\",\"message\":\"invalid entry id\"}}", nil},
		{"encoded slash remains in entry id", "GET", "/api/v1/entries/a%2Fb", 400, "{\"success\":false,\"error\":{\"code\":\"validation\",\"message\":\"invalid entry id\"}}", nil},
		{"missing route", "GET", "/missing", 404, "404 page not found\n", nil},
		{"unsupported collection method", "PUT", "/api/v1/entries", 405, "", []string{"GET", "POST"}},
		{"unsupported saved count method", "PUT", "/api/v1/entries/saved-count", 405, "", []string{"GET"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			router := api.New(t)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, tc.path, nil)

			// Act
			router.ServeHTTP(w, req)

			// Assert
			require.Equal(t, tc.status, w.Code)
			require.Equal(t, tc.body, w.Body.String())
			require.ElementsMatch(t, tc.allow, w.Header().Values("Allow"))
		})
	}
}
