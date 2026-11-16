package rss_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestObtainAtomEntries(t *testing.T) {
	// Arrange
	body := `<a:feed xmlns:a="http://www.w3.org/2005/Atom" xml:base="https://example.com/base/"><a:entry xml:base="../articles/"><a:title> First </a:title><a:link rel="alternate" type="application/pdf" href="file.pdf"/><a:link type="text/html" href="page"/><a:published>2024-01-02T03:04:05+07:00</a:published></a:entry><a:entry><a:title>Missing link</a:title><a:link rel="self" href="self"/></a:entry></a:feed>`
	adapter := rss.New(&http.Client{Transport: fixtureTransport{body: body}})
	// Act
	entries, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://feeds.example/feed"})
	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "First", entries[0].Title)
	require.Equal(t, "https://example.com/articles/file.pdf", entries[0].Link)
	require.Equal(t, time.Date(2024, 1, 1, 20, 4, 5, 0, time.UTC), *entries[0].PublishedAt)
}

func TestObtainDistinguishesEmptyAndUnsupportedDocuments(t *testing.T) {
	// Arrange
	for _, tc := range []struct {
		name, body string
		valid      bool
	}{
		{"empty Atom", `<feed xmlns="http://www.w3.org/2005/Atom"/>`, true},
		{"empty RSS", `<rss><channel/></rss>`, true},
		{"unrecognized Atom namespace accepted by gofeed", `<feed xmlns="http://example.com/atom"/>`, true},
		{"unsupported root", `<html/>`, false},
		{"malformed XML tolerated by gofeed", `<feed xmlns="http://www.w3.org/2005/Atom"><entry></feed>`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			adapter := rss.New(&http.Client{Transport: fixtureTransport{body: tc.body}})
			// Act
			entries, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://example.com/feed"})
			// Assert
			if tc.valid {
				// Assert
				require.NoError(t, err)
				require.Empty(t, entries)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestObtainAtomSkipsIncompleteEntriesAndUsesGofeedPublishedTime(t *testing.T) {
	// Arrange
	body := `<feed xmlns="http://www.w3.org/2005/Atom"><entry><title> </title><link href="/missing-title"/></entry><entry><title>No alternate</title><id>https://example.com/id</id><link rel="enclosure" href="/image"/></entry><entry><title>Unknown time</title><link href="/page"/><published>bad</published><updated>2024-01-01T00:00:00Z</updated></entry></feed>`
	adapter := rss.New(&http.Client{Transport: fixtureTransport{body: body}})
	// Act
	entries, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://example.com/feed"})
	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "/page", entries[0].Link)
	require.Equal(t, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), *entries[0].PublishedAt)
}

func TestObtainAtomDescriptions(t *testing.T) {
	// Arrange
	body := `<feed xmlns="http://www.w3.org/2005/Atom" xml:base="https://example.test/base/"><entry><title>Summary</title><link href="one"/><summary type="html" xml:base="../docs/">&lt;p&gt;Read &lt;a href="guide"&gt;guide&lt;/a&gt;&lt;/p&gt;</summary><content>ignored</content></entry><entry><title>Literal</title><link href="two"/><summary> </summary><content type="text">&lt;b&gt;literal&lt;/b&gt;</content></entry><entry><title>XHTML</title><link href="three"/><content type="xhtml"><div xmlns="http://www.w3.org/1999/xhtml" xml:base="../"><p>See <a href="page">page</a></p></div></content></entry><entry><title>External</title><link href="four"/><content src="external" type="text/html"/></entry></feed>`
	adapter := rss.New(&http.Client{Transport: fixtureTransport{body: body}})
	// Act
	entries, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://feeds.test/feed"})
	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 4)
	require.Contains(t, entries[0].Description, `href="https://example.test/docs/guide"`)
	require.Contains(t, entries[0].Description, "Read")
	require.Equal(t, "<b>literal</b>", entries[1].Description)
	require.Contains(t, entries[2].Description, `href="https://example.test/base/page"`)
	require.Empty(t, entries[3].Description)
}

func TestObtainAtomDescriptionFallbacks(t *testing.T) {
	// Arrange
	body := `<feed xmlns="http://www.w3.org/2005/Atom"><entry><title>Fallback</title><link href="one"/><summary type="html">&lt;p&gt; &lt;/p&gt;</summary><content type="html">&lt;strong&gt;Content&lt;/strong&gt;</content></entry><entry><title>Binary</title><link href="two"/><content type="image/png">aGVsbG8=</content></entry><entry><title>Missing</title><link href="three"/></entry></feed>`
	adapter := rss.New(&http.Client{Transport: fixtureTransport{body: body}})
	// Act
	entries, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://feeds.test/feed"})
	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 3)
	require.Equal(t, "<p> </p>", entries[0].Description)
	require.Equal(t, "hello", entries[1].Description)
	require.Empty(t, entries[2].Description)
}

func TestObtainAtomUsesGofeedImageSelection(t *testing.T) {
	// Arrange
	body := `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/" xml:base="https://example.com/base/"><entry xml:base="../entries/"><title>Entry</title><link href="page"/><link rel="enclosure" type="image/jpeg" href="fallback.jpg"/><media:thumbnail xml:base="images/" url="preferred.jpg"/></entry></feed>`
	adapter := rss.New(&http.Client{Transport: fixtureTransport{body: body}})
	// Act
	entries, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://feeds.example/feed"})
	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.NotNil(t, entries[0].Thumbnail)
	require.Equal(t, "https://example.com/entries/fallback.jpg", *entries[0].Thumbnail)
}

func TestObtainAtomImageEnclosureFallback(t *testing.T) {
	// Arrange
	for _, tc := range []struct{ name, media, links, want string }{
		{"empty thumbnail falls back", `<media:thumbnail url=" "/>`, `<link rel="enclosure" type="image/jpeg" xml:base="images/" href="photo.jpg"/>`, "/images/photo.jpg"},
		{"skips non-image and untyped", "", `<link rel="enclosure" type="audio/mpeg" href="audio"/><link rel="enclosure" href="untyped"/><link rel="enclosure" type="IMAGE/PNG" href="image.png"/>`, "image.png"},
		{"no supported images", "", `<link rel="enclosure" type="audio/mpeg" href="audio"/><link rel="enclosure" href="untyped"/>`, ""},
		{"no image", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			body := `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:media="http://search.yahoo.com/mrss/"><entry><title>Entry</title><link href="page"/>` + tc.media + tc.links + `</entry></feed>`
			adapter := rss.New(&http.Client{Transport: fixtureTransport{body: body}})
			// Act
			entries, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://feeds.example/feed"})
			// Assert
			require.NoError(t, err)
			require.Len(t, entries, 1)
			if tc.want == "" {
				require.Nil(t, entries[0].Thumbnail)
			} else {
				require.NotNil(t, entries[0].Thumbnail)
				require.Equal(t, tc.want, *entries[0].Thumbnail)
			}
		})
	}

}

func TestObtainAtomFormattedTitles(t *testing.T) {
	// Arrange
	body := `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:h="http://www.w3.org/1999/xhtml"><entry><title type="html">Hello &lt;strong&gt;world&lt;/strong&gt; &amp;amp; friends</title><link href="one"/></entry><entry><title type="xhtml"><h:div>Another <h:em>title</h:em></h:div></title><link href="two"/></entry></feed>`
	adapter := rss.New(&http.Client{Transport: fixtureTransport{body: body}})
	// Act
	entries, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://example.com/feed"})
	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 2)
	require.Equal(t, "Hello <strong>world</strong> &amp; friends", entries[0].Title)
	require.Equal(t, "<h:div>Another <h:em>title</h:em></h:div>", entries[1].Title)
}

func TestObtainAtomPrefixedXHTMLDescription(t *testing.T) {
	// Arrange
	body := `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:h="http://www.w3.org/1999/xhtml"><entry><title>Title</title><link href="page"/><summary type="xhtml"><h:div xml:base="/docs/"><h:p>Read <h:a href="guide"><h:strong>guide</h:strong></h:a></h:p></h:div></summary></entry></feed>`
	adapter := rss.New(&http.Client{Transport: fixtureTransport{body: body}})
	// Act
	entries, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://example.com/feed"})
	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Contains(t, entries[0].Description, `<h:a href="guide">`)
	require.Contains(t, entries[0].Description, `<h:strong>guide</h:strong>`)
}

func TestObtainAtomKeepsRelativeURLsAfterRedirect(t *testing.T) {
	// Arrange
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed" {
			http.Redirect(w, r, "/redirected/feed", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte(`<feed xmlns="http://www.w3.org/2005/Atom"><entry><title>Title</title><link href="page"/><link rel="enclosure" type="image/png" href="image.png"/><summary type="html">&lt;a href="guide"&gt;Guide&lt;/a&gt;</summary></entry></feed>`))
	}))
	defer server.Close()
	// Act
	entries, err := rss.New(server.Client()).Obtain(context.Background(), &service.SourceRef{RSSURL: server.URL + "/feed"})
	// Assert
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, "page", entries[0].Link)
	require.Equal(t, "image.png", *entries[0].Thumbnail)
	require.Contains(t, entries[0].Description, `href="guide"`)
}

func TestObtainRSSAlternateMediaPrefixIsNotMappedToImage(t *testing.T) {
	// Arrange
	body := `<rss xmlns:picture="http://search.yahoo.com/mrss/"><channel><item><title>Item</title><link>https://example.com/item</link><picture:thumbnail url="https://example.com/image.jpg"/></item></channel></rss>`
	adapter := rss.New(&http.Client{Transport: fixtureTransport{body: body}})

	// Act
	items, err := adapter.Obtain(context.Background(), &service.SourceRef{RSSURL: "https://example.com/feed"})

	// Assert
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Nil(t, items[0].Thumbnail)
}
