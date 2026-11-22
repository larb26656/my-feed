package devto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type adapter struct{ client HTTPClient }

type article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	CoverImage  string `json:"cover_image"`
	PublishedAt string `json:"published_at"`
}

var valuePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

func normalize(config map[string]any) (map[string]any, string, error) {
	kind, _ := config["kind"].(string)
	value, _ := config["value"].(string)
	value = strings.ToLower(strings.TrimSpace(value))
	switch kind {
	case "tag":
		value = strings.TrimPrefix(value, "#")
	case "username":
		value = strings.TrimPrefix(value, "@")
	default:
		return nil, "", apperr.InvalidInput.New("invalid DEV.to kind")
	}
	if !valuePattern.MatchString(value) {
		return nil, "", apperr.InvalidInput.New("invalid DEV.to tag or username")
	}
	return map[string]any{"kind": kind, "value": value}, kind + ":" + value, nil
}

func (a adapter) Obtain(ctx context.Context, input *api.Input) ([]api.PreparedEntry, error) {
	if input == nil {
		return nil, apperr.InvalidInput.New("source is required")
	}
	config, _, err := normalize(input.Config)
	if err != nil {
		return nil, err
	}
	query := url.Values{"page": {"1"}, "per_page": {"30"}, config["kind"].(string): {config["value"].(string)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://dev.to/api/articles?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.forem.api-v1+json")
	res, err := a.client.Do(req)
	if err != nil {
		return nil, apperr.ResourceConflict.Wrap(err, "DEV.to request failed")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, apperr.ResourceConflict.New(fmt.Sprintf("DEV.to request failed: status %d", res.StatusCode))
	}
	const maxBody = 8 << 20
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if err != nil {
		return nil, apperr.ResourceConflict.Wrap(err, "DEV.to response read failed")
	}
	if len(body) > maxBody {
		return nil, apperr.ResourceConflict.New("DEV.to response exceeded the size limit")
	}
	var articles []article
	if err := json.Unmarshal(body, &articles); err != nil {
		return nil, apperr.ResourceConflict.Wrap(err, "invalid DEV.to articles response")
	}
	if articles == nil {
		return nil, apperr.ResourceConflict.New("invalid DEV.to articles response")
	}
	entries := make([]api.PreparedEntry, 0, len(articles))
	for _, article := range articles {
		title := strings.TrimSpace(article.Title)
		link := strings.TrimSpace(article.URL)
		parsed, err := url.Parse(link)
		if title == "" || err != nil || parsed.Scheme != "https" || !strings.EqualFold(parsed.Host, "dev.to") || parsed.User != nil || parsed.Path == "" {
			return nil, apperr.ResourceConflict.New("invalid DEV.to article")
		}
		entry := api.PreparedEntry{Title: title, Link: link, Description: strings.TrimSpace(article.Description)}
		if image := strings.TrimSpace(article.CoverImage); image != "" {
			entry.Thumbnail = &image
		}
		if published, err := time.Parse(time.RFC3339, article.PublishedAt); err == nil {
			published = published.UTC()
			entry.PublishedAt = &published
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func Definition(client HTTPClient) api.Definition {
	if client == nil {
		client = http.DefaultClient
	}
	return api.Definition{
		Code:  "devto",
		Label: "DEV.to",
		Schema: map[string]any{
			"$schema":              "https://json-schema.org/draft/2020-12/schema",
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"kind":  map[string]any{"type": "string", "enum": []any{"tag", "username"}},
				"value": map[string]any{"type": "string", "pattern": `^\s*[#@]?[A-Za-z0-9_-]+\s*$`},
			},
			"required": []any{"kind", "value"},
		},
		UI: map[string]any{
			"kind":  map[string]any{"label": "Follow by", "widget": "select"},
			"value": map[string]any{"label": "Tag or username", "widget": "text", "description": "Enter a tag like #go or a username like @ben."},
		},
		Normalize: normalize,
		Obtainer:  adapter{client: client},
	}
}
