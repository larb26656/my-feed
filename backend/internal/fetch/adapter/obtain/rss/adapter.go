package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/larb26656/my-feed/backend/internal/fetch/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

// HTTPClient is the narrow HTTP seam used to fetch a feed document.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Adapter obtains Entry-ready fields from an RSS feed URL. It does not write the database.
type Adapter struct {
	client HTTPClient
}

func New(client HTTPClient) *Adapter {
	if client == nil {
		client = http.DefaultClient
	}
	return &Adapter{client: client}
}

func (a *Adapter) Obtain(ctx context.Context, source *service.SourceRef) ([]service.PreparedEntry, error) {
	if source == nil {
		return nil, apperr.Validation("source is required")
	}
	feedURL := strings.TrimSpace(source.RSSURL)
	if feedURL == "" {
		return nil, apperr.Validation("rss feed url is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, apperr.Validation("invalid rss feed url")
	}
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml, */*")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, apperr.FailedPrecondition(fmt.Sprintf("rss feed request failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, res.Body)
		return nil, apperr.FailedPrecondition(fmt.Sprintf("rss feed request failed: status %d", res.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, apperr.FailedPrecondition("rss feed body read failed")
	}
	return parseRSS(body)
}

type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	Description string         `xml:"description"`
	PubDate     string         `xml:"pubDate"`
	Enclosure   *rssEnclosure  `xml:"enclosure"`
	MediaThumb  *mediaThumbnail `xml:"http://search.yahoo.com/mrss/ thumbnail"`
}

type rssEnclosure struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type mediaThumbnail struct {
	URL string `xml:"url,attr"`
}

func parseRSS(body []byte) ([]service.PreparedEntry, error) {
	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, apperr.Validation("invalid rss feed document")
	}

	out := make([]service.PreparedEntry, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		title := strings.TrimSpace(item.Title)
		link := strings.TrimSpace(item.Link)
		if title == "" || link == "" {
			continue
		}
		entry := service.PreparedEntry{
			Title:       title,
			Link:        link,
			Description: strings.TrimSpace(item.Description),
			Thumbnail:   thumbnailFromItem(item),
		}
		if published := parsePubDate(item.PubDate); published != nil {
			entry.PublishedAt = published
		}
		out = append(out, entry)
	}
	return out, nil
}

func thumbnailFromItem(item rssItem) *string {
	if item.MediaThumb != nil {
		if u := strings.TrimSpace(item.MediaThumb.URL); u != "" {
			return &u
		}
	}
	if item.Enclosure != nil {
		u := strings.TrimSpace(item.Enclosure.URL)
		if u == "" {
			return nil
		}
		typ := strings.ToLower(strings.TrimSpace(item.Enclosure.Type))
		if typ == "" || strings.HasPrefix(typ, "image/") {
			return &u
		}
	}
	return nil
}

func parsePubDate(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	layouts := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			utc := t.UTC()
			return &utc
		}
	}
	return nil
}
