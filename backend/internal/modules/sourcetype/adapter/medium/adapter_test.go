package medium_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/medium"
	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

func TestNormalizeTrimsAtAndFingerprintsProfile(t *testing.T) {
	for _, tc := range []struct {
		name, input, wantID, wantKey string
	}{
		{"bare", "alice", "alice", "profile:alice"},
		{"with at", "@alice", "alice", "profile:alice"},
		{"trim whitespace", "  @Alice  ", "Alice", "profile:Alice"},
		{"underscores and hyphens", "my_user-name", "my_user-name", "profile:my_user-name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			normalize := medium.Definition(nil).Normalize

			// Act
			config, key, err := normalize(context.Background(), map[string]any{"kind": "profile", "identifier": tc.input})

			// Assert
			require.NoError(t, err)
			require.Equal(t, map[string]any{"kind": "profile", "identifier": tc.wantID}, config)
			require.Equal(t, tc.wantKey, key)
		})
	}
}

func TestNormalizePublicationBareSlugAndKey(t *testing.T) {
	for _, tc := range []struct {
		name, input, wantID, wantKey string
	}{
		{"bare", "alice", "alice", "publication:alice"},
		{"trim whitespace", "  Alice  ", "Alice", "publication:Alice"},
		{"underscores and hyphens", "my_pub-name", "my_pub-name", "publication:my_pub-name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			normalize := medium.Definition(nil).Normalize

			// Act
			config, key, err := normalize(context.Background(), map[string]any{
				"kind": "publication", "identifier": tc.input,
			})

			// Assert
			require.NoError(t, err)
			require.Equal(t, map[string]any{"kind": "publication", "identifier": tc.wantID}, config)
			require.Equal(t, tc.wantKey, key)
		})
	}
}

func TestNormalizeRejectsPublicationLeadingAt(t *testing.T) {
	// Arrange
	normalize := medium.Definition(nil).Normalize

	// Act
	got, key, err := normalize(context.Background(), map[string]any{
		"kind": "publication", "identifier": "@alice",
	})

	// Assert
	require.Nil(t, got)
	require.Empty(t, key)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.InvalidInput.Code, ae.Code)
}

func TestPrepareRejectsPublicationLeadingAtViaSchema(t *testing.T) {
	// Arrange
	registry, err := service.New(medium.Definition(nil))
	require.NoError(t, err)

	// Act
	got, key, err := registry.Prepare(context.Background(), "medium", map[string]any{
		"kind": "publication", "identifier": "@alice",
	})

	// Assert
	require.Nil(t, got)
	require.Empty(t, key)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.InvalidInput.Code, ae.Code)
}

func TestPrepareAllowsProfileLeadingAt(t *testing.T) {
	// Arrange
	registry, err := service.New(medium.Definition(nil))
	require.NoError(t, err)

	// Act
	config, key, err := registry.Prepare(context.Background(), "medium", map[string]any{
		"kind": "profile", "identifier": "@Alice",
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, map[string]any{"kind": "profile", "identifier": "Alice"}, config)
	require.Equal(t, "profile:Alice", key)
}

func TestNormalizeProfileAndPublicationSameIdentifierDifferentKeys(t *testing.T) {
	// Arrange
	normalize := medium.Definition(nil).Normalize

	// Act
	profile, profileKey, err := normalize(context.Background(), map[string]any{"kind": "profile", "identifier": "alice"})
	require.NoError(t, err)
	publication, publicationKey, err := normalize(context.Background(), map[string]any{"kind": "publication", "identifier": "alice"})

	// Assert
	require.NoError(t, err)
	require.Equal(t, "profile:alice", profileKey)
	require.Equal(t, "publication:alice", publicationKey)
	require.NotEqual(t, profile, publication)
}

func TestNormalizeRejectsInvalidIdentifiers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		kind   string
		value  any
		omitID bool
	}{
		{"empty profile", "profile", "  ", false},
		{"only at profile", "profile", "@", false},
		{"url profile", "profile", "https://medium.com/@alice", false},
		{"domain path profile", "profile", "medium.com/@alice", false},
		{"path segment profile", "profile", "alice/extra", false},
		{"query profile", "profile", "alice?x=1", false},
		{"fragment profile", "profile", "alice#bio", false},
		{"missing identifier profile", "profile", nil, true},
		{"empty publication", "publication", "  ", false},
		{"url publication", "publication", "https://medium.com/alice", false},
		{"domain publication", "publication", "medium.com/alice", false},
		{"path segment publication", "publication", "alice/extra", false},
		{"query publication", "publication", "alice?x=1", false},
		{"fragment publication", "publication", "alice#bio", false},
		{"invalid kind", "channel", "alice", false},
		{"missing kind", "", "alice", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			config := map[string]any{}
			if tc.kind != "" {
				config["kind"] = tc.kind
			}
			if !tc.omitID && tc.value != nil {
				config["identifier"] = tc.value
			}

			// Act
			got, key, err := medium.Definition(nil).Normalize(context.Background(), config)

			// Assert
			require.Nil(t, got)
			require.Empty(t, key)
			var ae *apperr.AppError
			require.ErrorAs(t, err, &ae)
			require.Equal(t, apperr.InvalidInput.Code, ae.Code)
		})
	}
}

type capturingTransport struct {
	requests []string
	handler  func(*http.Request) (*http.Response, error)
}

func (c *capturingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.requests = append(c.requests, r.URL.String())
	return c.handler(r)
}

func TestObtainRequestsProfileFeedURLAndMapsEntries(t *testing.T) {
	// Arrange
	body := `<rss version="2.0"><channel><title>Alice</title><item><title>First</title><link>https://medium.com/@alice/first</link><description>Hello</description></item></channel></rss>`
	transport := &capturingTransport{handler: func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	}}
	obtainer := medium.Definition(&http.Client{Transport: transport}).Obtainer

	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"kind": "profile", "identifier": "@Alice"},
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, []string{"https://medium.com/feed/@Alice"}, transport.requests)
	require.Len(t, entries, 1)
	require.Equal(t, "First", entries[0].Title)
	require.Equal(t, "https://medium.com/@alice/first", entries[0].Link)
	require.Equal(t, "Hello", entries[0].Description)
}

func TestObtainRequestsPublicationFeedURL(t *testing.T) {
	// Arrange
	body := `<rss version="2.0"><channel><title>Pub</title><item><title>First</title><link>https://medium.com/alice/first</link><description>Hello</description></item></channel></rss>`
	transport := &capturingTransport{handler: func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	}}
	obtainer := medium.Definition(&http.Client{Transport: transport}).Obtainer

	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"kind": "publication", "identifier": "Alice"},
	})

	// Assert
	require.NoError(t, err)
	require.Equal(t, []string{"https://medium.com/feed/Alice"}, transport.requests)
	require.Len(t, entries, 1)
	require.Equal(t, "First", entries[0].Title)
}

func TestObtainRejectsCustomDomainRedirectWithoutFollowUp(t *testing.T) {
	// Arrange
	transport := &capturingTransport{handler: func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://custom.example/feed"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	}}
	obtainer := medium.Definition(&http.Client{Transport: transport}).Obtainer

	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"kind": "profile", "identifier": "alice"},
	})

	// Assert
	require.Nil(t, entries)
	require.Error(t, err)
	require.Equal(t, []string{"https://medium.com/feed/@alice"}, transport.requests)
}

func TestObtainPublicationRejectsCustomDomainRedirectWithoutFollowUp(t *testing.T) {
	// Arrange
	transport := &capturingTransport{handler: func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": []string{"https://custom.example/feed"}},
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    r,
		}, nil
	}}
	obtainer := medium.Definition(&http.Client{Transport: transport}).Obtainer

	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"kind": "publication", "identifier": "alice"},
	})

	// Assert
	require.Nil(t, entries)
	require.Error(t, err)
	require.Equal(t, []string{"https://medium.com/feed/alice"}, transport.requests)
}

func TestObtainAcceptsEmptyReadableFeed(t *testing.T) {
	// Arrange
	body := `<rss version="2.0"><channel><title>Empty</title></channel></rss>`
	obtainer := medium.Definition(&http.Client{Transport: &capturingTransport{handler: func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	}}}).Obtainer

	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"kind": "profile", "identifier": "alice"},
	})

	// Assert
	require.NoError(t, err)
	require.Empty(t, entries)
}
