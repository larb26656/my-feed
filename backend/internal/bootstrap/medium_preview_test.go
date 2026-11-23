package bootstrap_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/bootstrap"
	"github.com/stretchr/testify/require"
)

func TestMediumCatalogExposesProfileAndPublication(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/source-types", nil)
	req.Header.Set("Authorization", "Bearer test-secret")
	catalog := httptest.NewRecorder()

	// Act
	app.Router.ServeHTTP(catalog, req)

	// Assert
	require.Equal(t, http.StatusOK, catalog.Code)
	require.Contains(t, catalog.Body.String(), `"code":"medium"`)
	require.Contains(t, catalog.Body.String(), `"label":"Medium"`)
	require.Contains(t, catalog.Body.String(), `"profile"`)
	require.Contains(t, catalog.Body.String(), `"publication"`)
}

func TestMediumPreviewProfileReturnsFirstEntry(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	var requests []string
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.URL.String())
		body := `<rss version="2.0"><channel><title>Alice</title><item><title>Story</title><link>https://medium.com/@alice/story</link><description>Body</description></item></channel></rss>`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})

	// Act
	preview := postMedium(t, app, "/api/v1/sources/preview", `{"type":"medium","config":{"kind":"profile","identifier":" @Alice "}}`, http.StatusOK)

	// Assert
	require.Contains(t, preview, `"title":"Story"`)
	require.Contains(t, preview, `"link":"https://medium.com/@alice/story"`)
	require.Equal(t, []string{"https://medium.com/feed/@Alice"}, requests)
}

func TestMediumPreviewPublicationReturnsFirstEntry(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	var requests []string
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.URL.String())
		body := `<rss version="2.0"><channel><title>Pub</title><item><title>Pub Story</title><link>https://medium.com/alice/story</link><description>Body</description></item></channel></rss>`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})

	// Act
	pubPreview := postMedium(t, app, "/api/v1/sources/preview", `{"type":"medium","config":{"kind":"publication","identifier":"Alice"}}`, http.StatusOK)

	// Assert
	require.Contains(t, pubPreview, `"title":"Pub Story"`)
	require.Equal(t, []string{"https://medium.com/feed/Alice"}, requests)
}

func TestMediumPreviewEmptyReadableFeed(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "https://medium.com/feed/@emptyfeed", r.URL.String())
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`<rss version="2.0"><channel><title>Empty</title></channel></rss>`)), Header: make(http.Header)}, nil
	})

	// Act
	empty := postMedium(t, app, "/api/v1/sources/preview", `{"type":"medium","config":{"kind":"profile","identifier":"emptyfeed"}}`, http.StatusOK)

	// Assert
	require.Contains(t, empty, `"entry":null`)
}

func TestMediumPreviewUpstreamError(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusBadGateway, Body: io.NopCloser(strings.NewReader("nope")), Header: make(http.Header)}, nil
	})

	// Act & Assert
	postMedium(t, app, "/api/v1/sources/preview", `{"type":"medium","config":{"kind":"profile","identifier":"alice"}}`, http.StatusBadRequest)
}

func TestMediumPreviewRejectsCustomDomainRedirect(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	var requests []string
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.URL.String())
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://custom.example/feed"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})

	// Act
	postMedium(t, app, "/api/v1/sources/preview", `{"type":"medium","config":{"kind":"profile","identifier":"alice"}}`, http.StatusBadRequest)

	// Assert
	require.Equal(t, []string{"https://medium.com/feed/@alice"}, requests)
}

func TestMediumPreviewPublicationRejectsCustomDomainRedirect(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	var requests []string
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.URL.String())
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://custom.example/feed"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})

	// Act
	postMedium(t, app, "/api/v1/sources/preview", `{"type":"medium","config":{"kind":"publication","identifier":"alice"}}`, http.StatusBadRequest)

	// Assert
	require.Equal(t, []string{"https://medium.com/feed/alice"}, requests)
}

func TestMediumPreviewRejectsInvalidConfig(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()

	for _, tc := range []struct {
		name string
		body string
	}{
		{"blank identifier", `{"type":"medium","config":{"kind":"profile","identifier":" "}}`},
		{"profile url", `{"type":"medium","config":{"kind":"profile","identifier":"https://medium.com/@alice"}}`},
		{"profile path", `{"type":"medium","config":{"kind":"profile","identifier":"alice/extra"}}`},
		{"publication leading at", `{"type":"medium","config":{"kind":"publication","identifier":"@alice"}}`},
		{"publication path", `{"type":"medium","config":{"kind":"publication","identifier":"alice/extra"}}`},
		{"missing kind", `{"type":"medium","config":{"identifier":"alice"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Act & Assert
			postMedium(t, app, "/api/v1/sources/preview", tc.body, http.StatusBadRequest)
		})
	}
}

func TestMediumCreatePersistsNormalizedConfig(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()

	// Act
	created := postMedium(t, app, "/api/v1/sources/", `{"name":"Alice","type":"medium","config":{"kind":"profile","identifier":"@Alice"}}`, http.StatusCreated)

	// Assert
	var createdBody struct {
		Data struct {
			Config map[string]any `json:"config"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(created), &createdBody))
	require.Equal(t, map[string]any{"kind": "profile", "identifier": "Alice"}, createdBody.Data.Config)
}

func TestMediumCreateConflictsOnDuplicateTarget(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()
	postMedium(t, app, "/api/v1/sources/", `{"name":"Alice","type":"medium","config":{"kind":"profile","identifier":"alice"}}`, http.StatusCreated)

	// Act & Assert
	postMedium(t, app, "/api/v1/sources/", `{"name":"Dup","type":"medium","config":{"kind":"profile","identifier":"alice"}}`, http.StatusConflict)
}

func TestMediumCreatePublicationPersistsNormalizedConfig(t *testing.T) {
	// Arrange
	app := newTestApp(t, "test-secret")
	defer app.Close()

	// Act
	pubCreated := postMedium(t, app, "/api/v1/sources/", `{"name":"Alice Pub","type":"medium","config":{"kind":"publication","identifier":"alice"}}`, http.StatusCreated)

	// Assert
	var createdBody struct {
		Data struct {
			Config map[string]any `json:"config"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(pubCreated), &createdBody))
	require.Equal(t, map[string]any{"kind": "publication", "identifier": "alice"}, createdBody.Data.Config)
}

func postMedium(t *testing.T, app *bootstrap.App, path, body string, want int) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret")
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	require.Equal(t, want, w.Code, w.Body.String())
	return w.Body.String()
}
