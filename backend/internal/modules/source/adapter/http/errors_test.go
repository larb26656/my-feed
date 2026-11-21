package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	"github.com/larb26656/my-feed/backend/internal/modules/source/repository"
	"github.com/larb26656/my-feed/backend/internal/modules/source/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
	"github.com/stretchr/testify/require"
)

type failingListRepo struct{ repository.Repository }

func (failingListRepo) ListSourcesPage(context.Context, repository.SourcePageQuery) ([]*domain.Source, int64, error) {
	return nil, 0, errors.New("private database failure")
}

func TestListErrorsUseCentralEnvelope(t *testing.T) {
	router := httpx.NewRouter()
	NewHandler(service.New(failingListRepo{}), nil).RegisterRoutes(router.Group("/sources"))
	for _, tc := range []struct {
		path   string
		status int
		code   string
	}{
		{"/sources?activeEqual=invalid", http.StatusBadRequest, "VALIDATION"},
		{"/sources", http.StatusInternalServerError, "INTERNAL_SERVER_ERROR"},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, tc.status, w.Code)
		require.Contains(t, w.Body.String(), tc.code)
		require.NotContains(t, w.Body.String(), "private database failure")
	}
}
