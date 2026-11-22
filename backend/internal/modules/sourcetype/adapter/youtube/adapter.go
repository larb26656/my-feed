package youtube

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	typeapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}
type Config struct{ ChannelID string }

func decodeConfig(values map[string]any) Config {
	id, _ := values["channelId"].(string)
	return Config{ChannelID: id}
}

type Adapter struct{ client HTTPClient }

func New(c HTTPClient) *Adapter {
	if c == nil {
		c = http.DefaultClient
	}
	return &Adapter{c}
}

var (
	channelPattern = regexp.MustCompile(`^UC[A-Za-z0-9_-]{22}$`)
	videoPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

func (a *Adapter) Obtain(ctx context.Context, source *typeapi.Input) ([]typeapi.PreparedEntry, error) {
	if source == nil {
		return nil, apperr.InvalidInput.New("source is required")
	}
	channelID := decodeConfig(source.Config).ChannelID
	if !channelPattern.MatchString(channelID) {
		return nil, apperr.InvalidInput.New("invalid youtube channel ID")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.youtube.com/feeds/videos.xml?channel_id="+url.QueryEscape(channelID), nil)
	if err != nil {
		return nil, err
	}
	res, err := a.client.Do(req)
	if err != nil {
		return nil, apperr.ResourceConflict.New(fmt.Sprintf("youtube feed request failed: %v", err))
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, apperr.ResourceConflict.New(fmt.Sprintf("youtube feed request failed: status %d", res.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return nil, apperr.ResourceConflict.New("youtube feed body read failed")
	}
	var feed struct {
		XMLName xml.Name `xml:"http://www.w3.org/2005/Atom feed"`
		Entries []struct {
			ID        string `xml:"http://www.youtube.com/xml/schemas/2015 videoId"`
			Title     string `xml:"http://www.w3.org/2005/Atom title"`
			Published string `xml:"http://www.w3.org/2005/Atom published"`
			Group     struct {
				Description string `xml:"http://search.yahoo.com/mrss/ description"`
				Thumbnail   struct {
					URL string `xml:"url,attr"`
				} `xml:"http://search.yahoo.com/mrss/ thumbnail"`
			} `xml:"http://search.yahoo.com/mrss/ group"`
		} `xml:"http://www.w3.org/2005/Atom entry"`
	}
	decoder := xml.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(&feed); err != nil {
		return nil, apperr.InvalidInput.New("invalid youtube atom feed document")
	}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, apperr.InvalidInput.New("invalid youtube atom feed document")
		}
		switch token := token.(type) {
		case xml.CharData:
			if strings.TrimSpace(string(token)) != "" {
				return nil, apperr.InvalidInput.New("invalid youtube atom feed document")
			}
		case xml.Comment, xml.ProcInst, xml.Directive:
		default:
			return nil, apperr.InvalidInput.New("invalid youtube atom feed document")
		}
	}
	out := make([]typeapi.PreparedEntry, 0, len(feed.Entries))
	for _, item := range feed.Entries {
		id, title := strings.TrimSpace(item.ID), strings.TrimSpace(item.Title)
		if !videoPattern.MatchString(id) || title == "" {
			return nil, apperr.InvalidInput.New("invalid youtube feed entry")
		}
		e := typeapi.PreparedEntry{Title: title, Link: "https://www.youtube.com/watch?v=" + id, Description: strings.TrimSpace(item.Group.Description)}
		if thumb := strings.TrimSpace(item.Group.Thumbnail.URL); thumb != "" {
			e.Thumbnail = &thumb
		}
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Published)); err == nil {
			t = t.UTC()
			e.PublishedAt = &t
		}
		out = append(out, e)
	}
	return out, nil
}

// Definition describes YouTube input and its obtain behavior.
func Definition(client HTTPClient) typeapi.Definition {
	return typeapi.Definition{Code: "youtube", Label: "YouTube", Schema: map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema", "type": "object", "additionalProperties": false,
		"properties": map[string]any{"channelId": map[string]any{"type": "string", "pattern": `^\s*(https://www\.youtube\.com/channel/)?UC[A-Za-z0-9_-]{22}\s*$`}}, "required": []any{"channelId"},
	}, UI: map[string]any{"channelId": map[string]any{"label": "YouTube channel ID or URL", "widget": "text", "description": "Handles and custom URLs are not supported.", "error": "Enter a channel ID starting with UC or a canonical YouTube channel URL"}}, Normalize: normalize, Obtainer: New(client)}
}

func normalize(_ context.Context, config map[string]any) (map[string]any, string, error) {
	id := strings.TrimPrefix(strings.TrimSpace(decodeConfig(config).ChannelID), "https://www.youtube.com/channel/")
	if !channelPattern.MatchString(id) {
		return nil, "", apperr.InvalidInput.New("invalid YouTube channel ID")
	}
	return map[string]any{"channelId": id}, id, nil
}
