package rss

import (
	"bytes"
	"context"
	"fmt"
	"github.com/larb26656/my-feed/backend/internal/fetch/domain"
	"io"
	"net/http"
	"strings"

	"github.com/mmcdole/gofeed"

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
		return nil, apperr.InvalidInput.New("source is required")
	}
	feedURL := strings.TrimSpace(source.RSSURL)
	if feedURL == "" {
		return nil, apperr.InvalidInput.New("rss feed url is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, apperr.InvalidInput.New("invalid rss feed url")
	}
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml, */*")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, domain.ErrConflict.New(fmt.Sprintf("rss feed request failed: %v", err))
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, res.Body)
		return nil, domain.ErrConflict.New(fmt.Sprintf("rss feed request failed: status %d", res.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, domain.ErrConflict.New("rss feed body read failed")
	}
	feed, err := gofeed.NewParser().Parse(bytes.NewReader(body))
	if err != nil {
		return nil, apperr.InvalidInput.New("invalid or unsupported feed document")
	}
	entries := make([]service.PreparedEntry, 0, len(feed.Items))
	for _, item := range feed.Items {
		title, link := strings.TrimSpace(item.Title), strings.TrimSpace(item.Link)
		if title == "" || link == "" {
			continue
		}
		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = strings.TrimSpace(item.Content)
		}
		entry := service.PreparedEntry{Title: title, Link: link, Description: description, PublishedAt: item.PublishedParsed}
		if item.Image != nil && strings.TrimSpace(item.Image.URL) != "" {
			thumbnail := strings.TrimSpace(item.Image.URL)
			entry.Thumbnail = &thumbnail
		} else {
			for _, enclosure := range item.Enclosures {
				if strings.HasPrefix(strings.ToLower(strings.TrimSpace(enclosure.Type)), "image/") && strings.TrimSpace(enclosure.URL) != "" {
					thumbnail := strings.TrimSpace(enclosure.URL)
					entry.Thumbnail = &thumbnail
					break
				}
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
