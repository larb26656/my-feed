package devto_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/devto"
	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type responseClient struct {
	body   string
	status int
	err    error
	last   *http.Request
}

func (c *responseClient) Do(r *http.Request) (*http.Response, error) {
	c.last = r
	if c.err != nil {
		return nil, c.err
	}
	status := c.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(c.body))}, nil
}

func TestNormalizeTrimsAndFingerprintsTagAndUsername(t *testing.T) {
	for _, tc := range []struct {
		name, kind, value, wantValue, wantFingerprint string
	}{
		{"tag with hash", "tag", " #Go ", "go", "tag:go"},
		{"username with at", "username", " @Ben ", "ben", "username:ben"},
		{"tag already clean", "tag", "javascript", "javascript", "tag:javascript"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			normalize := devto.Definition(nil).Normalize
			// Act
			config, fingerprint, err := normalize(context.Background(), map[string]any{"kind": tc.kind, "value": tc.value})
			// Assert
			require.NoError(t, err)
			require.Equal(t, map[string]any{"kind": tc.kind, "value": tc.wantValue}, config)
			require.Equal(t, tc.wantFingerprint, fingerprint)
		})
	}
}

func TestNormalizeRejectsInvalidKindOrValue(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config map[string]any
		want   string
	}{
		{"unknown kind", map[string]any{"kind": "org", "value": "go"}, "invalid DEV.to kind"},
		{"missing kind", map[string]any{"value": "go"}, "invalid DEV.to kind"},
		{"empty value", map[string]any{"kind": "tag", "value": "  "}, "invalid DEV.to tag or username"},
		{"illegal characters", map[string]any{"kind": "username", "value": "bad/name"}, "invalid DEV.to tag or username"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			normalize := devto.Definition(nil).Normalize
			// Act
			config, fingerprint, err := normalize(context.Background(), tc.config)
			// Assert
			require.Nil(t, config)
			require.Empty(t, fingerprint)
			var ae *apperr.AppError
			require.ErrorAs(t, err, &ae)
			require.Equal(t, apperr.InvalidInput.Code, ae.Code)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestObtainMapsArticlesForTag(t *testing.T) {
	// Arrange
	body := `[{"title":"Go Tips","description":"Useful tips","url":"https://dev.to/ben/go-tips","cover_image":"https://media.dev.to/cover.jpg","published_at":"2024-01-02T03:04:05Z"}]`
	client := &responseClient{body: body}
	obtainer := devto.Definition(client).Obtainer
	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"kind": "tag", "value": "#Go"},
	})
	// Assert
	require.NoError(t, err)
	require.Equal(t, "https://dev.to/api/articles?page=1&per_page=30&tag=go", client.last.URL.String())
	require.Equal(t, "application/vnd.forem.api-v1+json", client.last.Header.Get("Accept"))
	require.Len(t, entries, 1)
	require.Equal(t, "Go Tips", entries[0].Title)
	require.Equal(t, "https://dev.to/ben/go-tips", entries[0].Link)
	require.Equal(t, "Useful tips", entries[0].Description)
	require.Equal(t, "https://media.dev.to/cover.jpg", *entries[0].Thumbnail)
	require.Equal(t, time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), entries[0].PublishedAt.UTC())
}

func TestObtainMapsArticlesForUsernameWithoutOptionalFields(t *testing.T) {
	// Arrange
	body := `[{"title":"Hello","description":"","url":"https://dev.to/ben/hello","cover_image":null,"published_at":"not-a-date"}]`
	client := &responseClient{body: body}
	obtainer := devto.Definition(client).Obtainer
	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"kind": "username", "value": "@Ben"},
	})
	// Assert
	require.NoError(t, err)
	require.Equal(t, "https://dev.to/api/articles?page=1&per_page=30&username=ben", client.last.URL.String())
	require.Len(t, entries, 1)
	require.Equal(t, "Hello", entries[0].Title)
	require.Equal(t, "https://dev.to/ben/hello", entries[0].Link)
	require.Empty(t, entries[0].Description)
	require.Nil(t, entries[0].Thumbnail)
	require.Nil(t, entries[0].PublishedAt)
}

func TestObtainAcceptsEmptyArticleList(t *testing.T) {
	// Arrange
	obtainer := devto.Definition(&responseClient{body: `[]`}).Obtainer
	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"kind": "tag", "value": "go"},
	})
	// Assert
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestObtainRejectsMissingSource(t *testing.T) {
	// Arrange
	obtainer := devto.Definition(&responseClient{}).Obtainer
	// Act
	entries, err := obtainer.Obtain(context.Background(), nil)
	// Assert
	require.Nil(t, entries)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.InvalidInput.Code, ae.Code)
	require.ErrorContains(t, err, "source is required")
}

func TestObtainRejectsInvalidConfig(t *testing.T) {
	// Arrange
	obtainer := devto.Definition(&responseClient{}).Obtainer
	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"kind": "org", "value": "go"},
	})
	// Assert
	require.Nil(t, entries)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.InvalidInput.Code, ae.Code)
}

func TestObtainReportsRequestFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		client responseClient
		want   string
	}{
		{"HTTP failure", responseClient{status: http.StatusServiceUnavailable}, "DEV.to request failed: status 503"},
		{"network failure", responseClient{err: errors.New("offline")}, "DEV.to request failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			client := tc.client
			obtainer := devto.Definition(&client).Obtainer
			// Act
			entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
				Config: map[string]any{"kind": "tag", "value": "go"},
			})
			// Assert
			require.Nil(t, entries)
			var ae *apperr.AppError
			require.ErrorAs(t, err, &ae)
			require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestObtainRejectsInvalidResponses(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
	}{
		{"malformed JSON", `{`, "invalid DEV.to articles response"},
		{"null array", `null`, "invalid DEV.to articles response"},
		{"object instead of array", `{}`, "invalid DEV.to articles response"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			obtainer := devto.Definition(&responseClient{body: tc.body}).Obtainer
			// Act
			entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
				Config: map[string]any{"kind": "tag", "value": "go"},
			})
			// Assert
			require.Nil(t, entries)
			var ae *apperr.AppError
			require.ErrorAs(t, err, &ae)
			require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestObtainRejectsMalformedArticles(t *testing.T) {
	for _, tc := range []struct {
		name, article string
	}{
		{"missing title", `{"title":"","url":"https://dev.to/ben/post"}`},
		{"blank title", `{"title":" ","url":"https://dev.to/ben/post"}`},
		{"http link", `{"title":"Post","url":"http://dev.to/ben/post"}`},
		{"wrong host", `{"title":"Post","url":"https://example.com/ben/post"}`},
		{"empty path", `{"title":"Post","url":"https://dev.to"}`},
		{"userinfo", `{"title":"Post","url":"https://user@dev.to/ben/post"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			body := `[{"title":"Valid","url":"https://dev.to/ben/valid"},` + tc.article + `]`
			obtainer := devto.Definition(&responseClient{body: body}).Obtainer
			// Act
			entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
				Config: map[string]any{"kind": "tag", "value": "go"},
			})
			// Assert
			require.Nil(t, entries)
			require.ErrorContains(t, err, "invalid DEV.to article")
			var ae *apperr.AppError
			require.ErrorAs(t, err, &ae)
			require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
		})
	}
}
