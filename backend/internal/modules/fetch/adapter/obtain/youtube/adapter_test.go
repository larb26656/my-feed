package youtube_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/modules/fetch/adapter/obtain/youtube"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
)

type client struct{}

func (client) Do(r *http.Request) (*http.Response, error) {
	if r.URL.String() != "https://www.youtube.com/feeds/videos.xml?channel_id=UC1234567890123456789012" {
		panic(r.URL.String())
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015" xmlns:media="http://search.yahoo.com/mrss/"><entry><yt:videoId>abc123</yt:videoId><title>Video</title><published>2024-01-02T03:04:05Z</published><media:group><media:description>Description</media:description><media:thumbnail url="https://example.com/thumb.jpg"/></media:group></entry></feed>`))}, nil
}
func TestObtainChannelEntries(t *testing.T) {
	// Arrange
	adapter := youtube.New(client{})
	// Act
	entries, err := adapter.Obtain(context.Background(), &sourceapi.SourceRef{YouTubeChannelID: "UC1234567890123456789012"})
	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "https://www.youtube.com/watch?v=abc123", entries[0].Link)
	require.Equal(t, "Video", entries[0].Title)
	require.Equal(t, "Description", entries[0].Description)
	require.Equal(t, "https://example.com/thumb.jpg", *entries[0].Thumbnail)
	require.Equal(t, "2024-01-02T03:04:05Z", entries[0].PublishedAt.Format("2006-01-02T15:04:05Z"))
}

type responseClient struct {
	body   string
	status int
	err    error
}

func (c responseClient) Do(*http.Request) (*http.Response, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &http.Response{StatusCode: c.status, Body: io.NopCloser(strings.NewReader(c.body))}, nil
}
func TestObtainRecognizesEmptyAndRejectsInvalidDocuments(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantError  bool
	}{
		{"empty Atom feed", `<feed xmlns="http://www.w3.org/2005/Atom"/>`, false},
		{"unsupported root", `<rss/>`, true},
		{"wrong namespace", `<feed/>`, true},
		{"malformed XML", `<feed xmlns="http://www.w3.org/2005/Atom"><entry></feed>`, true},
		{"trailing malformed XML", `<feed xmlns="http://www.w3.org/2005/Atom"/><broken`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			adapter := youtube.New(responseClient{body: tc.body, status: 200})
			// Act
			entries, err := adapter.Obtain(context.Background(), &sourceapi.SourceRef{YouTubeChannelID: "UC1234567890123456789012"})
			// Assert
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Empty(t, entries)
			}
		})
	}
}

func TestObtainReportsRequestFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		client responseClient
	}{{"HTTP failure", responseClient{status: 503}}, {"network failure", responseClient{err: errors.New("offline")}}} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			adapter := youtube.New(tc.client)
			// Act
			_, err := adapter.Obtain(context.Background(), &sourceapi.SourceRef{YouTubeChannelID: "UC1234567890123456789012"})
			// Assert
			require.Error(t, err)
		})
	}
}

func TestObtainAcceptsEntriesWithoutOptionalMetadata(t *testing.T) {
	// Arrange
	body := `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015"><entry><yt:videoId>scheduled_1</yt:videoId><title>Scheduled stream</title></entry><entry><yt:videoId>live-2</yt:videoId><title>Live stream</title><published>2024-01-02T10:04:05+07:00</published></entry></feed>`
	// Act
	entries, err := youtube.New(responseClient{body: body, status: 200}).Obtain(context.Background(), &sourceapi.SourceRef{YouTubeChannelID: "UC1234567890123456789012"})
	require.NoError(t, err)
	// Assert
	require.Len(t, entries, 2)
	require.Equal(t, "https://www.youtube.com/watch?v=scheduled_1", entries[0].Link)
	require.Empty(t, entries[0].Description)
	require.Nil(t, entries[0].Thumbnail)
	require.Nil(t, entries[0].PublishedAt)
	require.Equal(t, "2024-01-02T03:04:05Z", entries[1].PublishedAt.Format("2006-01-02T15:04:05Z"))
}

func TestObtainRejectsMalformedEntries(t *testing.T) {
	for _, tc := range []struct{ name, entry string }{
		{"missing video ID", `<title>Video</title>`},
		{"invalid video ID", `<yt:videoId>bad/id</yt:videoId><title>Video</title>`},
		{"missing title", `<yt:videoId>video123</yt:videoId>`},
		{"blank title", `<yt:videoId>video123</yt:videoId><title> </title>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			body := `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:yt="http://www.youtube.com/xml/schemas/2015"><entry><yt:videoId>valid</yt:videoId><title>Valid</title></entry><entry>` + tc.entry + `</entry></feed>`
			adapter := youtube.New(responseClient{body: body, status: 200})
			// Act
			entries, err := adapter.Obtain(context.Background(), &sourceapi.SourceRef{YouTubeChannelID: "UC1234567890123456789012"})
			// Assert
			require.ErrorContains(t, err, "invalid youtube feed entry")
			require.Nil(t, entries)
		})
	}
}
