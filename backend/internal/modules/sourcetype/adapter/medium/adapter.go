package medium

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type adapter struct {
	rss *rss.Adapter
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func normalize(_ context.Context, config map[string]any) (map[string]any, string, error) {
	kind, _ := config["kind"].(string)
	raw, _ := config["identifier"].(string)
	identifier := strings.TrimSpace(raw)
	switch kind {
	case "profile":
		identifier = strings.TrimPrefix(identifier, "@")
	case "publication":
		if strings.HasPrefix(identifier, "@") {
			return nil, "", apperr.InvalidInput.New("invalid Medium publication slug").WithDetail(map[string]any{"field": "config.identifier"})
		}
	default:
		return nil, "", apperr.InvalidInput.New("invalid Medium kind").WithDetail(map[string]any{"field": "config.kind"})
	}
	if !identifierPattern.MatchString(identifier) {
		msg := "invalid Medium username"
		if kind == "publication" {
			msg = "invalid Medium publication slug"
		}
		return nil, "", apperr.InvalidInput.New(msg).WithDetail(map[string]any{"field": "config.identifier"})
	}
	return map[string]any{"kind": kind, "identifier": identifier}, kind + ":" + identifier, nil
}

func (a adapter) Obtain(ctx context.Context, input *api.Input) ([]api.PreparedEntry, error) {
	if input == nil {
		return nil, apperr.InvalidInput.New("source is required")
	}
	config, _, err := normalize(ctx, input.Config)
	if err != nil {
		return nil, err
	}
	identifier := config["identifier"].(string)
	feedURL := "https://medium.com/feed/" + identifier
	if config["kind"] == "profile" {
		feedURL = "https://medium.com/feed/@" + identifier
	}
	return a.rss.Obtain(ctx, &api.Input{
		Type:   api.ProviderRSS,
		Config: map[string]any{"url": feedURL},
	})
}

func Definition(client HTTPClient) api.Definition {
	return api.Definition{
		Code:  "medium",
		Label: "Medium",
		Schema: map[string]any{
			"$schema":              "https://json-schema.org/draft/2020-12/schema",
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"kind": map[string]any{
					"type": "string",
					"enum": []any{"profile", "publication"},
				},
				"identifier": map[string]any{
					"type": "string",
				},
			},
			"required": []any{"kind", "identifier"},
			"allOf": []any{
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{"kind": map[string]any{"const": "profile"}},
						"required":   []any{"kind"},
					},
					"then": map[string]any{
						"properties": map[string]any{
							"identifier": map[string]any{
								"type":    "string",
								"pattern": `^\s*@?[A-Za-z0-9_-]+\s*$`,
							},
						},
					},
				},
				map[string]any{
					"if": map[string]any{
						"properties": map[string]any{"kind": map[string]any{"const": "publication"}},
						"required":   []any{"kind"},
					},
					"then": map[string]any{
						"properties": map[string]any{
							"identifier": map[string]any{
								"type":    "string",
								"pattern": `^\s*[A-Za-z0-9_-]+\s*$`,
							},
						},
					},
				},
			},
		},
		FieldOrder: []string{"kind", "identifier"}, UI: map[string]any{
			"kind": map[string]any{
				"label":  "Kind",
				"widget": "select",
			},
			"identifier": map[string]any{
				"label":       "Identifier",
				"widget":      "text",
				"description": "Profile username like alice or @alice, or a bare publication slug.",
				"error":       "Enter a Medium identifier without a URL or path",
			},
		},
		Normalize: normalize,
		Obtainer:  adapter{rss: rss.New(restrictToMedium(client))},
	}
}

func restrictToMedium(client HTTPClient) HTTPClient {
	base, ok := client.(*http.Client)
	if !ok {
		if client == nil {
			base = http.DefaultClient
		} else {
			return client
		}
	}
	previous := base.CheckRedirect
	clone := *base
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		host := strings.ToLower(req.URL.Hostname())
		if host != "medium.com" && host != "www.medium.com" {
			return fmt.Errorf("medium redirect host not allowed: %s", host)
		}
		if previous != nil {
			return previous(req, via)
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}
	return &clone
}
