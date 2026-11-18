package api_test

import (
	"github.com/larb26656/my-feed/backend/internal/testkit/api"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func TestRoutingContractCapture(t *testing.T) {
	router := api.New(t)
	for _, tc := range []struct {
		method, path string
		status       int
		body         string
		allow        []string
	}{
		{"GET", "/api/v1/entries", 200, "{\"data\":[],\"page\":{\"nextCursor\":null,\"hasNext\":false}}\n", nil}, {"GET", "/api/v1/entries/", 200, "{\"data\":[],\"page\":{\"nextCursor\":null,\"hasNext\":false}}\n", nil},
		{"GET", "/api/v1/entries/saved-count/", 404, "404 page not found\n", nil}, {"GET", "/api/v1/entries/%73aved-count", 400, "{\"error\":\"invalid entry id\",\"code\":\"validation\"}\n", nil},
		{"GET", "/api/v1/entries/a%2Fb", 400, "{\"error\":\"invalid entry id\",\"code\":\"validation\"}\n", nil}, {"GET", "/missing", 404, "404 page not found\n", nil},
		{"PUT", "/api/v1/entries", 405, "", []string{"GET", "POST"}}, {"PUT", "/api/v1/entries/saved-count", 405, "", []string{"GET"}},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		require.Equal(t, tc.status, w.Code, tc.path)
		require.Equal(t, tc.body, w.Body.String(), tc.path)
		require.ElementsMatch(t, tc.allow, w.Header().Values("Allow"), tc.path)
	}
}
