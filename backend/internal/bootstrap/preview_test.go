package bootstrap_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type previewTransport func(*http.Request) (*http.Response, error)

func (f previewTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPreviewRouteYouTubeAndPersistence(t *testing.T) {
	app := newTestApp(t, "test-secret")
	defer app.Close()
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "https://www.youtube.com/feeds/videos.xml?channel_id=UC1234567890123456789012", r.URL.String())
		body := `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015" xmlns:media="http://search.yahoo.com/mrss/"><entry><yt:videoId>first</yt:videoId><title>First video</title><media:group><media:description>Video description</media:description><media:thumbnail url="https://example.test/thumb.jpg"/></media:group></entry><entry><yt:videoId>second</yt:videoId><title>Second</title></entry></feed>`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	counts := func() []int {
		result := make([]int, 4)
		for i, query := range []string{"SELECT count(*) FROM sources", "SELECT count(*) FROM sources WHERE profile_image_id IS NOT NULL", "SELECT count(*) FROM entries", "SELECT count(*) FROM fetch_jobs"} {
			require.NoError(t, app.Pool.DB.QueryRow(query).Scan(&result[i]))
		}
		return result
	}
	before := counts()
	for _, input := range []string{"UC1234567890123456789012", "https://www.youtube.com/channel/UC1234567890123456789012"} {
		body, err := json.Marshal(map[string]string{"type": "youtube", "youtubeChannelId": input})
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/preview", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-secret")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var response struct {
			Data struct {
				Entry struct{ Title, Link, Description, Thumbnail string }
			}
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		require.Equal(t, "First video", response.Data.Entry.Title)
		require.Equal(t, "https://www.youtube.com/watch?v=first", response.Data.Entry.Link)
		require.Equal(t, "Video description", response.Data.Entry.Description)
		require.Equal(t, "https://example.test/thumb.jpg", response.Data.Entry.Thumbnail)
	}
	require.Equal(t, before, counts())
	denied := httptest.NewRecorder()
	app.Router.ServeHTTP(denied, httptest.NewRequest(http.MethodPost, "/api/v1/sources/preview", strings.NewReader(`{"type":"rss","rssUrl":"https://example.test"}`)))
	require.Equal(t, http.StatusUnauthorized, denied.Code)
}

func TestPreviewRouteRejectsInvalidConfiguration(t *testing.T) {
	app := newTestApp(t, "test-secret")
	defer app.Close()
	for _, input := range []string{
		`{"type":"rss","rssUrl":"not a URL"}`,
		`{"type":"youtube","youtubeChannelId":"invalid"}`,
		`{"type":"unknown"}`,
	} {
		t.Run(input, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/preview", strings.NewReader(input))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-secret")
			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
}
