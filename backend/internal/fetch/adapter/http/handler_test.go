package http

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

func TestListJobsQueryRejectsInvalidInput(t *testing.T) {
	for _, query := range []string{
		"sourceId=bad", "sourceId=", "status=", "status=PENDING", "status=queued",
		"pageSize=", "pageSize=0", "pageSize=-1", "pageSize=101", "pageSize=nope",
		"pageSize=99999999999999999999999999", "unknown=value", "status=pending&status=failed",
		"sourceId=a&sourceId=b", "cursor=a&cursor=b", "pageSize=1&pageSize=2",
		"status=pending&bad=%zz", "status=pending;status=failed", "status.foo=pending",
	} {
		t.Run(query, func(t *testing.T) {
			// Arrange
			req := httptest.NewRequest("GET", "/?"+query, nil)
			// Act
			rec := httptest.NewRecorder()
			NewHandler(nil).Routes().ServeHTTP(rec, req)
			// Assert
			require.Equal(t, 400, rec.Code)
			var body httpx.ErrorBody
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, string(apperr.CodeValidation), body.Code)
			require.NotEmpty(t, body.Error)
		})
	}
}

func TestListJobsQueryPreservesOmittedFields(t *testing.T) {
	// Arrange
	req := httptest.NewRequest("GET", "/", nil)
	// Act
	query, err := httpx.DecodeQuery[listJobsQuery](req)
	// Assert
	require.NoError(t, err)
	require.Equal(t, listJobsQuery{}, query)
}

func TestListJobsQueryAcceptsFilters(t *testing.T) {
	for _, status := range []string{"pending", "running", "succeeded", "failed"} {
		for _, size := range []string{"1", "100"} {
			t.Run(status+size, func(t *testing.T) {
				// Arrange
				req := httptest.NewRequest("GET", "/?sourceId=00000000-0000-0000-0000-000000000000&status="+status+"&pageSize="+size+"&cursor=a%2Bb%3D", nil)
				// Act
				query, err := httpx.DecodeQuery[listJobsQuery](req)
				// Assert
				require.NoError(t, err)
				require.Equal(t, "00000000-0000-0000-0000-000000000000", *query.SourceID)
				require.Equal(t, status, *query.Status)
				require.Equal(t, "a+b=", *query.Cursor)
				require.Contains(t, []int{1, 100}, *query.PageSize)
			})
		}
	}
}
