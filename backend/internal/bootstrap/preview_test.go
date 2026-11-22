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
		body, err := json.Marshal(map[string]any{"type": "youtube", "config": map[string]any{"channelId": input}})
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
	app.Router.ServeHTTP(denied, httptest.NewRequest(http.MethodPost, "/api/v1/sources/preview", strings.NewReader(`{"type":"rss","config":{"url":"https://example.test"}}`)))
	require.Equal(t, http.StatusUnauthorized, denied.Code)
}

func TestPreviewRouteRejectsInvalidConfiguration(t *testing.T) {
	app := newTestApp(t, "test-secret")
	defer app.Close()
	for _, input := range []string{
		`{"type":"rss","config":{"url":"not a URL"}}`,
		`{"type":"youtube","config":{"channelId":"invalid"}}`,
		`{"type":"youtube","config":{}}`,
		`{"type":"youtube","config":{"channelId":"UC1234567890123456789012","extra":true}}`,
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

func TestPreviewRSSConfigUsesCreateValidation(t *testing.T) {
	app := newTestApp(t, "test-secret")
	defer app.Close()
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		require.Contains(t, []string{"https://example.test/feed", "https://example.test/empty"}, r.URL.String())
		body := `<rss version="2.0"><channel><title>Feed</title><item><title>First</title><link>https://example.test/first</link></item><item><title>Second</title><link>https://example.test/second</link></item></channel></rss>`
		if r.URL.Path == "/empty" {
			body = `<rss version="2.0"><channel><title>Empty</title></channel></rss>`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	counts := func() []int {
		result := make([]int, 3)
		for i, q := range []string{"SELECT count(*) FROM sources", "SELECT count(*) FROM entries", "SELECT count(*) FROM fetch_jobs"} {
			require.NoError(t, app.Pool.DB.QueryRow(q).Scan(&result[i]))
		}
		return result
	}
	before := counts()
	for _, tc := range []struct {
		body   string
		status int
		entry  string
	}{
		{`{"type":"rss","config":{"url":" https://example.test/feed "}}`, http.StatusOK, `"title":"First"`},
		{`{"type":"rss","config":{"url":"https://example.test/empty"}}`, http.StatusOK, `"entry":null`},
		{`{"type":"rss","config":{"url":"https://example.test/feed","extra":true}}`, http.StatusBadRequest, ""},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/preview", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-secret")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		require.Equal(t, tc.status, w.Code, w.Body.String())
		if tc.entry != "" {
			require.Contains(t, w.Body.String(), tc.entry)
		}
	}
	require.Equal(t, before, counts())
}

func TestPreviewRejectsRedirectToPrivateAddress(t *testing.T) {
	app := newTestApp(t, "test-secret")
	defer app.Close()
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		require.NotEqual(t, "127.0.0.1", r.URL.Hostname())
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"http://127.0.0.1/internal"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/preview", strings.NewReader(`{"type":"rss","config":{"url":"https://example.test/feed"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret")
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

func TestDevToPreviewAndSourceIdentity(t *testing.T) {
	app := newTestApp(t, "test-secret")
	defer app.Close()

	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "dev.to", r.URL.Host)
		require.Equal(t, "/api/articles", r.URL.Path)
		require.Equal(t, "application/vnd.forem.api-v1+json", r.Header.Get("Accept"))
		require.Empty(t, r.Header.Get("api-key"))
		query := r.URL.Query()
		require.Equal(t, "1", query.Get("page"))
		require.Equal(t, "30", query.Get("per_page"))
		require.Len(t, query, 3)
		if query.Has("tag") {
			require.Equal(t, "go", query.Get("tag"))
		} else {
			require.Equal(t, "go", query.Get("username"))
		}
		body := `[{"title":"Go article","url":"https://dev.to/go/article","canonical_url":"https://example.test/article","description":"About Go","cover_image":"https://example.test/cover.jpg","published_at":"2025-01-02T03:04:05Z"}]`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})

	post := func(path, config string, want int) string {
		t.Helper()
		body := `{"type":"devto","name":"DEV.to","config":` + config + `}`
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-secret")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		require.Equal(t, want, w.Code, w.Body.String())
		return w.Body.String()
	}

	const sources = "/api/v1/sources"
	const tag = `{"kind":"tag","value":" #Go "}`
	const username = `{"kind":"username","value":" @Go "}`
	for _, config := range []string{tag, username} {
		preview := post(sources+"/preview", config, http.StatusOK)
		require.Contains(t, preview, `"link":"https://dev.to/go/article"`)
		require.Contains(t, preview, `"description":"About Go"`)
		require.Contains(t, preview, `"thumbnail":"https://example.test/cover.jpg"`)
	}
	post(sources+"/preview", `{"kind":"tag","value":"@go"}`, http.StatusBadRequest)
	post(sources, tag, http.StatusCreated)
	post(sources, `{"kind":"tag","value":"go"}`, http.StatusConflict)
	post(sources, username, http.StatusCreated)
}

func TestBlueskyPreviewAndSourceIdentity(t *testing.T) {
	app := newTestApp(t, "test-secret")
	defer app.Close()

	const did = "did:plc:z72i7hdynmk6r22z27h6tvur"
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "public.api.bsky.app", r.URL.Host)
		switch r.URL.Path {
		case "/xrpc/com.atproto.identity.resolveHandle":
			handle := r.URL.Query().Get("handle")
			switch handle {
			case "alice.bsky.social", "alice.newhandle.social":
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"did":"` + did + `"}`))}, nil
			case "missing.bsky.social":
				return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":"InvalidRequest"}`))}, nil
			default:
				t.Fatalf("unexpected handle %q", handle)
			}
		case "/xrpc/app.bsky.feed.getAuthorFeed":
			require.Equal(t, did, r.URL.Query().Get("actor"))
			require.Equal(t, "50", r.URL.Query().Get("limit"))
			body := `{
  "feed": [
    {
      "post": {
        "uri": "at://` + did + `/app.bsky.feed.post/repost1",
        "author": {"did": "did:plc:other", "handle": "other.bsky.social"},
        "record": {"$type": "app.bsky.feed.post", "text": "Reposted", "createdAt": "2026-01-03T00:00:00.000Z"}
      },
      "reason": {"$type": "app.bsky.feed.defs#reasonRepost", "by": {"did": "` + did + `"}}
    },
    {
      "post": {
        "uri": "at://` + did + `/app.bsky.feed.post/3eligible",
        "author": {"did": "` + did + `", "handle": "alice.bsky.social"},
        "record": {"$type": "app.bsky.feed.post", "text": "First line\nRest of post", "createdAt": "2026-01-02T03:04:05.000Z"}
      }
    }
  ]
}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		return nil, nil
	})

	counts := func() []int {
		result := make([]int, 3)
		for i, q := range []string{"SELECT count(*) FROM sources", "SELECT count(*) FROM entries", "SELECT count(*) FROM fetch_jobs"} {
			require.NoError(t, app.Pool.DB.QueryRow(q).Scan(&result[i]))
		}
		return result
	}
	before := counts()

	post := func(method, path, body string, want int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-secret")
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		require.Equal(t, want, w.Code, w.Body.String())
		return w.Body.String()
	}

	const sources = "/api/v1/sources"
	for _, config := range []string{
		`{"did":"alice.bsky.social"}`,
		`{"did":" @alice.bsky.social "}`,
		`{"did":"https://bsky.app/profile/alice.bsky.social"}`,
		`{"did":"` + did + `"}`,
	} {
		preview := post(http.MethodPost, sources+"/preview", `{"type":"bluesky","config":`+config+`}`, http.StatusOK)
		require.Contains(t, preview, `"title":"First line"`)
		require.Contains(t, preview, `"description":"First line\nRest of post"`)
		require.Contains(t, preview, `"link":"https://bsky.app/profile/`+did+`/post/3eligible"`)
	}
	require.Equal(t, before, counts())

	post(http.MethodPost, sources+"/preview", `{"type":"bluesky","config":{"did":"missing.bsky.social"}}`, http.StatusBadRequest)
	post(http.MethodPost, sources+"/preview", `{"type":"bluesky","config":{"did":""}}`, http.StatusBadRequest)

	created := post(http.MethodPost, sources, `{"type":"bluesky","name":"Alice","config":{"did":"@alice.bsky.social"}}`, http.StatusCreated)
	require.Contains(t, created, `"did":"`+did+`"`)
	post(http.MethodPost, sources, `{"type":"bluesky","name":"Dup","config":{"did":"https://bsky.app/profile/alice.bsky.social"}}`, http.StatusConflict)
	post(http.MethodPost, sources, `{"type":"bluesky","name":"Renamed handle","config":{"did":"alice.newhandle.social"}}`, http.StatusConflict)

	var envelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(created), &envelope))

	post(http.MethodPatch, sources+"/"+envelope.Data.ID, `{"name":"Renamed Alice"}`, http.StatusOK)
	reject := post(http.MethodPatch, sources+"/"+envelope.Data.ID, `{"name":"Renamed Alice","config":{"did":"did:plc:otherprofile00000000000"}}`, http.StatusBadRequest)
	require.Contains(t, reject, "bluesky profile cannot be changed")
}

func TestBlueskyPreviewEmptyEligibleFeedPersistsNothing(t *testing.T) {
	app := newTestApp(t, "test-secret")
	defer app.Close()

	const did = "did:plc:z72i7hdynmk6r22z27h6tvur"
	old := http.DefaultClient.Transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	http.DefaultClient.Transport = previewTransport(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/xrpc/com.atproto.identity.resolveHandle":
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"did":"` + did + `"}`))}, nil
		case "/xrpc/app.bsky.feed.getAuthorFeed":
			body := `{"feed":[{
      "post": {
        "uri": "at://` + did + `/app.bsky.feed.post/reply1",
        "author": {"did": "` + did + `", "handle": "alice.bsky.social"},
        "record": {"$type": "app.bsky.feed.post", "text": "reply only", "createdAt": "2026-01-01T00:00:00.000Z", "reply": {"root": {"uri": "at://x"}, "parent": {"uri": "at://x"}}}
      },
      "reply": {"root": {"uri": "at://x"}, "parent": {"uri": "at://x"}}
    }]}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
			return nil, nil
		}
	})
	counts := func() []int {
		result := make([]int, 3)
		for i, q := range []string{"SELECT count(*) FROM sources", "SELECT count(*) FROM entries", "SELECT count(*) FROM fetch_jobs"} {
			require.NoError(t, app.Pool.DB.QueryRow(q).Scan(&result[i]))
		}
		return result
	}
	before := counts()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sources/preview", strings.NewReader(`{"type":"bluesky","config":{"did":"alice.bsky.social"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-secret")
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), `"entry":null`)
	require.Equal(t, before, counts())
}
