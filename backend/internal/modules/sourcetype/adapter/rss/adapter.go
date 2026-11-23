package rss

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	typeapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"

	"github.com/mmcdole/gofeed"

	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type Config struct{ URL string }

func decodeConfig(values map[string]any) Config {
	url, _ := values["url"].(string)
	return Config{URL: url}
}

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

func (a *Adapter) Obtain(ctx context.Context, source *typeapi.Input) ([]typeapi.PreparedEntry, error) {
	if source == nil {
		return nil, apperr.InvalidInput.New("source is required")
	}
	config := decodeConfig(source.Config)
	feedURL := strings.TrimSpace(config.URL)
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
		return nil, apperr.ResourceConflict.New(fmt.Sprintf("rss feed request failed: %v", err))
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, res.Body)
		return nil, apperr.ResourceConflict.New(fmt.Sprintf("rss feed request failed: status %d", res.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, apperr.ResourceConflict.New("rss feed body read failed")
	}
	feed, err := gofeed.NewParser().Parse(bytes.NewReader(body))
	if err != nil {
		return nil, apperr.InvalidInput.New("invalid or unsupported feed document")
	}
	entries := make([]typeapi.PreparedEntry, 0, len(feed.Items))
	for _, item := range feed.Items {
		title, link := strings.TrimSpace(item.Title), strings.TrimSpace(item.Link)
		if title == "" || link == "" {
			continue
		}
		description := strings.TrimSpace(item.Description)
		if description == "" {
			description = strings.TrimSpace(item.Content)
		}
		entry := typeapi.PreparedEntry{Title: title, Link: link, Description: description, PublishedAt: item.PublishedParsed}
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

// Definition keeps the accepted configuration and its obtain behavior together.
func Definition(client HTTPClient) typeapi.Definition {
	return typeapi.Definition{Code: "rss", Label: "RSS feed", Schema: map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "additionalProperties": false,
		"properties": map[string]any{"url": map[string]any{"type": "string", "pattern": `^\s*https?://([^\s/@]+@)?(\[[0-9a-fA-F:.]+\]|[^\s/:?#@\[\]]+)(:[0-9]+)?([/?#][^\s]*)?\s*$`}}, "required": []any{"url"},
	}, UI: map[string]any{"url": map[string]any{"label": "RSS / Atom feed URL", "widget": "url", "error": "Please enter a valid RSS / Atom URL"}}, Normalize: normalize, Obtainer: New(client)}
}

func normalize(_ context.Context, config map[string]any) (map[string]any, string, error) {
	typed := decodeConfig(config)
	feed := strings.TrimSpace(typed.URL)
	parsed, err := url.Parse(feed)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return nil, "", apperr.InvalidInput.New("invalid rss feed url").WithDetail(map[string]any{"field": "config.url"})
	}
	return map[string]any{"url": feed}, feed, nil
}
