package bluesky

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

const (
	publicAppView   = "https://public.api.bsky.app"
	authorFeedLimit = "50"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type adapter struct{ client HTTPClient }

var didPattern = regexp.MustCompile(`^did:[a-z]+:[a-zA-Z0-9._:%-]+$`)

func Definition(client HTTPClient) api.Definition {
	if client == nil {
		client = http.DefaultClient
	}
	a := &adapter{client: client}
	return api.Definition{
		Code:  "bluesky",
		Label: "Bluesky",
		Schema: map[string]any{
			"$schema":              "https://json-schema.org/draft/2020-12/schema",
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"did": map[string]any{"type": "string", "minLength": 1},
			},
			"required": []any{"did"},
		},
		UI: map[string]any{
			"did": map[string]any{
				"label":       "Bluesky handle or profile URL",
				"widget":      "text",
				"description": "Enter a handle like @alice.bsky.social or a bsky.app profile URL.",
				"error":       "Enter a Bluesky handle or profile URL",
			},
		},
		Normalize: a.normalize,
		Obtainer:  a,
	}
}

func (a *adapter) normalize(ctx context.Context, config map[string]any) (map[string]any, string, error) {
	raw, _ := config["did"].(string)
	identity, err := parseProfileInput(raw)
	if err != nil {
		return nil, "", err
	}
	if didPattern.MatchString(identity) {
		return map[string]any{"did": identity}, identity, nil
	}
	did, err := a.resolveHandle(ctx, identity)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"did": did}, did, nil
}

func parseProfileInput(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.InvalidInput.New("invalid Bluesky profile")
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		parsed, err := url.Parse(raw)
		if err != nil || !strings.EqualFold(parsed.Host, "bsky.app") {
			return "", apperr.InvalidInput.New("invalid Bluesky profile")
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) < 2 || parts[0] != "profile" || parts[1] == "" {
			return "", apperr.InvalidInput.New("invalid Bluesky profile")
		}
		raw = parts[1]
	}
	raw = strings.TrimPrefix(raw, "@")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", apperr.InvalidInput.New("invalid Bluesky profile")
	}
	if didPattern.MatchString(raw) {
		return raw, nil
	}
	return strings.ToLower(raw), nil
}

func (a *adapter) resolveHandle(ctx context.Context, handle string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, publicAppView+"/xrpc/com.atproto.identity.resolveHandle?handle="+url.QueryEscape(handle), nil)
	if err != nil {
		return "", err
	}
	res, err := a.client.Do(req)
	if err != nil {
		return "", apperr.InvalidInput.Wrap(err, "Bluesky profile unavailable")
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", apperr.InvalidInput.Wrap(err, "Bluesky profile unavailable")
	}
	if res.StatusCode != http.StatusOK {
		return "", apperr.InvalidInput.New("Bluesky profile unavailable")
	}
	var payload struct {
		DID string `json:"did"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || !didPattern.MatchString(payload.DID) {
		return "", apperr.InvalidInput.New("Bluesky profile unavailable")
	}
	return payload.DID, nil
}

func (a *adapter) Obtain(ctx context.Context, input *api.Input) ([]api.PreparedEntry, error) {
	if input == nil {
		return nil, apperr.InvalidInput.New("source is required")
	}
	did, _ := input.Config["did"].(string)
	did = strings.TrimSpace(did)
	if !didPattern.MatchString(did) {
		return nil, apperr.InvalidInput.New("invalid Bluesky DID")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, publicAppView+"/xrpc/app.bsky.feed.getAuthorFeed?actor="+url.QueryEscape(did)+"&limit="+authorFeedLimit, nil)
	if err != nil {
		return nil, err
	}
	res, err := a.client.Do(req)
	if err != nil {
		return nil, apperr.ResourceConflict.Wrap(err, "Bluesky feed request failed")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, apperr.ResourceConflict.New(fmt.Sprintf("Bluesky feed request failed: status %d", res.StatusCode))
	}
	const maxBody = 8 << 20
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if err != nil {
		return nil, apperr.ResourceConflict.Wrap(err, "Bluesky feed response read failed")
	}
	if len(body) > maxBody {
		return nil, apperr.ResourceConflict.New("Bluesky feed response exceeded the size limit")
	}
	var payload struct {
		Feed []feedItem `json:"feed"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, apperr.ResourceConflict.Wrap(err, "invalid Bluesky feed response")
	}
	entries := make([]api.PreparedEntry, 0, len(payload.Feed))
	for _, item := range payload.Feed {
		if !item.eligible() {
			continue
		}
		entry, ok := item.toEntry()
		if !ok {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

type feedItem struct {
	Post   postView       `json:"post"`
	Reason map[string]any `json:"reason"`
	Reply  map[string]any `json:"reply"`
}

type postView struct {
	URI    string `json:"uri"`
	Author struct {
		DID    string `json:"did"`
		Handle string `json:"handle"`
	} `json:"author"`
	Record postRecord `json:"record"`
	Embed  *embedView `json:"embed"`
}

type postRecord struct {
	Text      string         `json:"text"`
	CreatedAt string         `json:"createdAt"`
	Reply     map[string]any `json:"reply"`
}

type embedView struct {
	Type   string `json:"$type"`
	Images []struct {
		Thumb string `json:"thumb"`
	} `json:"images"`
	Thumbnail string `json:"thumbnail"`
	External  *struct {
		Thumb string `json:"thumb"`
	} `json:"external"`
	Media *embedView `json:"media"`
}

func (item feedItem) eligible() bool {
	if len(item.Reason) > 0 {
		return false
	}
	if len(item.Reply) > 0 || len(item.Post.Record.Reply) > 0 {
		return false
	}
	return true
}

func (item feedItem) toEntry() (api.PreparedEntry, bool) {
	authorDID, rkey, ok := parseAtURI(item.Post.URI)
	if !ok {
		return api.PreparedEntry{}, false
	}
	text := item.Post.Record.Text
	title := firstLine(text)
	if title == "" {
		if handle := strings.TrimSpace(item.Post.Author.Handle); handle != "" {
			title = "Post from @" + handle
		}
	}
	entry := api.PreparedEntry{
		Title:       title,
		Link:        "https://bsky.app/profile/" + authorDID + "/post/" + rkey,
		Description: text,
	}
	if thumb := item.Post.thumbnail(); thumb != "" {
		entry.Thumbnail = &thumb
	}
	if published, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Post.Record.CreatedAt)); err == nil {
		published = published.UTC()
		entry.PublishedAt = &published
	}
	return entry, true
}

func (p postView) thumbnail() string {
	return embedThumbnail(p.Embed)
}

func embedThumbnail(embed *embedView) string {
	if embed == nil {
		return ""
	}
	switch embed.Type {
	case "app.bsky.embed.images#view":
		for _, img := range embed.Images {
			if thumb := strings.TrimSpace(img.Thumb); thumb != "" {
				return thumb
			}
		}
	case "app.bsky.embed.video#view":
		return strings.TrimSpace(embed.Thumbnail)
	case "app.bsky.embed.external#view":
		if embed.External != nil {
			return strings.TrimSpace(embed.External.Thumb)
		}
	case "app.bsky.embed.recordWithMedia#view":
		return embedThumbnail(embed.Media)
	}
	return ""
}

func firstLine(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		return strings.TrimSpace(text[:i])
	}
	return strings.TrimSpace(text)
}

func parseAtURI(uri string) (did, rkey string, ok bool) {
	const prefix = "at://"
	if !strings.HasPrefix(uri, prefix) {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(uri, prefix), "/")
	if len(parts) != 3 || parts[1] != "app.bsky.feed.post" || !didPattern.MatchString(parts[0]) || parts[2] == "" {
		return "", "", false
	}
	return parts[0], parts[2], true
}
