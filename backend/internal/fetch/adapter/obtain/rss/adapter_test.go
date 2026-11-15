package rss_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/my-feed/backend/internal/fetch/adapter/obtain/rss"
	"github.com/larb26656/my-feed/backend/internal/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type fixtureTransport struct {
	body string
	code int
}

func (f fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	code := f.code
	if code == 0 {
		code = http.StatusOK
	}
	return &http.Response{
		StatusCode: code,
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func loadFixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return string(b)
}

func TestObtainMapsRSSItemsFromFeedFixture(t *testing.T) {
	body := loadFixture(t, "sample.xml")
	client := &http.Client{Transport: fixtureTransport{body: body}}
	adapter := rss.New(client)

	items, err := adapter.Obtain(context.Background(), &service.SourceRef{
		ID:     uuid.New(),
		Type:   service.SourceTypeRSS,
		Active: true,
		RSSURL: "https://feeds.example/fixture.xml",
	})
	require.NoError(t, err)
	require.Len(t, items, 2)

	require.Equal(t, "First Item", items[0].Title)
	require.Equal(t, "https://example.com/first", items[0].Link)
	require.Equal(t, "First description", items[0].Description)
	require.NotNil(t, items[0].Thumbnail)
	require.Equal(t, "https://example.com/first.jpg", *items[0].Thumbnail)
	require.NotNil(t, items[0].PublishedAt)
	require.True(t, items[0].PublishedAt.UTC().Equal(time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)))

	require.Equal(t, "Second Item", items[1].Title)
	require.Equal(t, "https://example.com/second", items[1].Link)
	require.Nil(t, items[1].Thumbnail)
}

func TestObtainRejectsMissingFeedURL(t *testing.T) {
	adapter := rss.New(&http.Client{Transport: fixtureTransport{body: ""}})
	_, err := adapter.Obtain(context.Background(), &service.SourceRef{
		ID:   uuid.New(),
		Type: service.SourceTypeRSS,
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestObtainSurfacesHTTPFailure(t *testing.T) {
	client := &http.Client{Transport: fixtureTransport{body: "nope", code: http.StatusBadGateway}}
	adapter := rss.New(client)
	_, err := adapter.Obtain(context.Background(), &service.SourceRef{
		ID:     uuid.New(),
		Type:   service.SourceTypeRSS,
		RSSURL: "https://feeds.example/fixture.xml",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)
}

func TestObtainSurfacesInvalidFeedDocument(t *testing.T) {
	client := &http.Client{Transport: fixtureTransport{body: "not xml at all"}}
	adapter := rss.New(client)
	_, err := adapter.Obtain(context.Background(), &service.SourceRef{
		ID:     uuid.New(),
		Type:   service.SourceTypeRSS,
		RSSURL: "https://feeds.example/fixture.xml",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}
