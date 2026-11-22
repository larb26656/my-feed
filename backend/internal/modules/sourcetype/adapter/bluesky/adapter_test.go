package bluesky_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/bluesky"
	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

const sampleDID = "did:plc:z72i7hdynmk6r22z27h6tvur"

type responseClient struct {
	body   string
	status int
	err    error
	last   *http.Request
	byPath map[string]responseClient
}

func (c *responseClient) Do(r *http.Request) (*http.Response, error) {
	c.last = r
	if handler, ok := c.byPath[r.URL.Path]; ok {
		handler.last = r
		c.last = r
		if handler.err != nil {
			return nil, handler.err
		}
		status := handler.status
		if status == 0 {
			status = http.StatusOK
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(handler.body))}, nil
	}
	if c.err != nil {
		return nil, c.err
	}
	status := c.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(c.body))}, nil
}

func TestNormalizeResolvesHandleVariantsToDID(t *testing.T) {
	for _, tc := range []struct {
		name, input string
	}{
		{"bare handle", "alice.bsky.social"},
		{"at handle", " @Alice.bsky.social "},
		{"profile URL", "https://bsky.app/profile/alice.bsky.social"},
		{"profile URL trailing slash", "https://bsky.app/profile/alice.bsky.social/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			client := &responseClient{body: `{"did":"` + sampleDID + `"}`}
			normalize := bluesky.Definition(client).Normalize
			// Act
			config, key, err := normalize(context.Background(), map[string]any{"did": tc.input})
			// Assert
			require.NoError(t, err)
			require.Equal(t, map[string]any{"did": sampleDID}, config)
			require.Equal(t, sampleDID, key)
			require.Equal(t, "public.api.bsky.app", client.last.URL.Host)
			require.Equal(t, "/xrpc/com.atproto.identity.resolveHandle", client.last.URL.Path)
			require.Equal(t, "alice.bsky.social", client.last.URL.Query().Get("handle"))
		})
	}
}

func TestNormalizeAcceptsDIDWithoutResolve(t *testing.T) {
	// Arrange
	client := &responseClient{err: errors.New("should not call network")}
	normalize := bluesky.Definition(client).Normalize
	// Act
	config, key, err := normalize(context.Background(), map[string]any{"did": " " + sampleDID + " "})
	// Assert
	require.NoError(t, err)
	require.Equal(t, map[string]any{"did": sampleDID}, config)
	require.Equal(t, sampleDID, key)
	require.Nil(t, client.last)
}

func TestNormalizeResolvesProfileURLWithDID(t *testing.T) {
	client := &responseClient{err: errors.New("should not call network")}
	normalize := bluesky.Definition(client).Normalize
	config, key, err := normalize(context.Background(), map[string]any{
		"did": "https://bsky.app/profile/" + sampleDID,
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"did": sampleDID}, config)
	require.Equal(t, sampleDID, key)
	require.Nil(t, client.last)
}

func TestNormalizeRejectsInvalidOrUnavailableProfiles(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config map[string]any
		client responseClient
		want   string
	}{
		{"empty", map[string]any{"did": "  "}, responseClient{}, "invalid Bluesky profile"},
		{"bad URL host", map[string]any{"did": "https://example.com/profile/alice"}, responseClient{}, "invalid Bluesky profile"},
		{"unavailable handle", map[string]any{"did": "missing.bsky.social"}, responseClient{status: http.StatusBadRequest, body: `{"error":"InvalidRequest"}`}, "Bluesky profile unavailable"},
		{"network failure", map[string]any{"did": "alice.bsky.social"}, responseClient{err: errors.New("offline")}, "Bluesky profile unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := tc.client
			normalize := bluesky.Definition(&client).Normalize
			config, key, err := normalize(context.Background(), tc.config)
			require.Nil(t, config)
			require.Empty(t, key)
			var ae *apperr.AppError
			require.ErrorAs(t, err, &ae)
			require.Equal(t, apperr.InvalidInput.Code, ae.Code)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestObtainSkipsRepliesAndRepostsAndMapsTextPosts(t *testing.T) {
	// Arrange
	body := `{
  "feed": [
    {
      "post": {
        "uri": "at://` + sampleDID + `/app.bsky.feed.post/repost1",
        "author": {"did": "did:plc:other", "handle": "other.bsky.social"},
        "record": {"$type": "app.bsky.feed.post", "text": "Someone else", "createdAt": "2026-01-03T00:00:00.000Z"}
      },
      "reason": {"$type": "app.bsky.feed.defs#reasonRepost", "by": {"did": "` + sampleDID + `"}}
    },
    {
      "post": {
        "uri": "at://` + sampleDID + `/app.bsky.feed.post/reply1",
        "author": {"did": "` + sampleDID + `", "handle": "alice.bsky.social"},
        "record": {
          "$type": "app.bsky.feed.post",
          "text": "A reply",
          "createdAt": "2026-01-02T00:00:00.000Z",
          "reply": {"root": {"uri": "at://did:plc:root/app.bsky.feed.post/1"}, "parent": {"uri": "at://did:plc:root/app.bsky.feed.post/1"}}
        }
      },
      "reply": {"root": {"uri": "at://did:plc:root/app.bsky.feed.post/1"}, "parent": {"uri": "at://did:plc:root/app.bsky.feed.post/1"}}
    },
    {
      "post": {
        "uri": "at://` + sampleDID + `/app.bsky.feed.post/3quote",
        "author": {"did": "` + sampleDID + `", "handle": "alice.bsky.social"},
        "record": {
          "$type": "app.bsky.feed.post",
          "text": "Quoted take\nSecond line",
          "createdAt": "2026-01-01T12:00:00.000Z",
          "embed": {"$type": "app.bsky.embed.record"}
        }
      }
    },
    {
      "post": {
        "uri": "at://` + sampleDID + `/app.bsky.feed.post/3original",
        "author": {"did": "` + sampleDID + `", "handle": "alice.bsky.social"},
        "record": {
          "$type": "app.bsky.feed.post",
          "text": "Hello Bluesky",
          "createdAt": "2026-01-01T00:00:00.000Z"
        }
      }
    }
  ]
}`
	client := &responseClient{body: body}
	obtainer := bluesky.Definition(client).Obtainer
	// Act
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"did": sampleDID},
	})
	// Assert
	require.NoError(t, err)
	require.Equal(t, "public.api.bsky.app", client.last.URL.Host)
	require.Equal(t, "/xrpc/app.bsky.feed.getAuthorFeed", client.last.URL.Path)
	require.Equal(t, sampleDID, client.last.URL.Query().Get("actor"))
	require.Equal(t, "50", client.last.URL.Query().Get("limit"))
	require.Len(t, entries, 2)
	require.Equal(t, "Quoted take", entries[0].Title)
	require.Equal(t, "Quoted take\nSecond line", entries[0].Description)
	require.Equal(t, "https://bsky.app/profile/"+sampleDID+"/post/3quote", entries[0].Link)
	require.Equal(t, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC), entries[0].PublishedAt.UTC())
	require.Nil(t, entries[0].Thumbnail)
	require.Equal(t, "Hello Bluesky", entries[1].Title)
	require.Equal(t, "https://bsky.app/profile/"+sampleDID+"/post/3original", entries[1].Link)
}

func TestObtainAcceptsEmptyEligibleFeed(t *testing.T) {
	body := `{"feed":[{
      "post": {
        "uri": "at://` + sampleDID + `/app.bsky.feed.post/reply1",
        "author": {"did": "` + sampleDID + `", "handle": "alice.bsky.social"},
        "record": {"$type": "app.bsky.feed.post", "text": "reply", "createdAt": "2026-01-01T00:00:00.000Z", "reply": {"root": {"uri": "at://x"}, "parent": {"uri": "at://x"}}}
      },
      "reply": {"root": {"uri": "at://x"}, "parent": {"uri": "at://x"}}
    }]}`
	obtainer := bluesky.Definition(&responseClient{body: body}).Obtainer
	entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
		Config: map[string]any{"did": sampleDID},
	})
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestObtainRejectsMissingSourceAndInvalidDID(t *testing.T) {
	obtainer := bluesky.Definition(&responseClient{}).Obtainer
	entries, err := obtainer.Obtain(context.Background(), nil)
	require.Nil(t, entries)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.InvalidInput.Code, ae.Code)

	entries, err = obtainer.Obtain(context.Background(), &obtainapi.Input{Config: map[string]any{"did": "alice.bsky.social"}})
	require.Nil(t, entries)
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.InvalidInput.Code, ae.Code)
}

func TestObtainReportsFeedFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		client responseClient
		want   string
	}{
		{"HTTP failure", responseClient{status: http.StatusServiceUnavailable}, "Bluesky feed request failed: status 503"},
		{"network failure", responseClient{err: errors.New("offline")}, "Bluesky feed request failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := tc.client
			obtainer := bluesky.Definition(&client).Obtainer
			entries, err := obtainer.Obtain(context.Background(), &obtainapi.Input{
				Config: map[string]any{"did": sampleDID},
			})
			require.Nil(t, entries)
			var ae *apperr.AppError
			require.ErrorAs(t, err, &ae)
			require.Equal(t, apperr.ResourceConflict.Code, ae.Code)
			require.ErrorContains(t, err, tc.want)
		})
	}
}
