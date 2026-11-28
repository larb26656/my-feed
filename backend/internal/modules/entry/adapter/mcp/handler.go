package mcp

import (
	"context"
	"net/http"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/larb26656/my-feed/backend/internal/modules/entry/service"
	"github.com/larb26656/my-feed/backend/internal/platform/apperr"
)

// Handler exposes Entry recording tools over Streamable HTTP MCP.
type Handler struct {
	http.Handler
	svc *service.Service
}

type saveLinkInput struct {
	Title       string  `json:"title" jsonschema:"required,Title of the Entry"`
	Link        string  `json:"link" jsonschema:"Optional URL of the Entry"`
	Description string  `json:"description,omitempty" jsonschema:"Optional description"`
	Thumbnail   *string `json:"thumbnail,omitempty" jsonschema:"Optional thumbnail URL"`
	PublishedAt *string `json:"publishedAt,omitempty" jsonschema:"Optional source-published time (RFC3339)"`
}

type saveLinkOutput struct {
	ID      string  `json:"id" jsonschema:"Stable Entry id"`
	Title   string  `json:"title" jsonschema:"Entry title"`
	Link    *string `json:"link" jsonschema:"Optional Entry URL"`
	Created bool    `json:"created" jsonschema:"True when a new Entry was created; false when metadata was refreshed"`
}

// NewHandler builds a Streamable HTTP MCP handler with the save_link tool.
func NewHandler(entries *service.Service) *Handler {
	h := &Handler{svc: entries}
	server := mcp.NewServer(&mcp.Implementation{Name: "my-feed", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "save_link",
		Description: "Record a link as an Entry on the manual Saved links Source. Does not mark the Entry as Saved (bookmarked).",
		InputSchema: &jsonschema.Schema{
			Type:     "object",
			Required: []string{"title"},
			Properties: map[string]*jsonschema.Schema{
				"title":       {Type: "string", Description: "Title of the Entry"},
				"link":        {Description: "Optional URL of the Entry", AnyOf: []*jsonschema.Schema{{Type: "null"}, {Type: "string"}}},
				"description": {Type: "string", Description: "Optional description"},
				"thumbnail":   {Description: "Optional thumbnail URL", AnyOf: []*jsonschema.Schema{{Type: "null"}, {Type: "string"}}},
				"publishedAt": {Description: "Optional source-published time (RFC3339)", AnyOf: []*jsonschema.Schema{{Type: "null"}, {Type: "string"}}},
			},
		},
	}, h.saveLink)
	h.Handler = mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		JSONResponse: true,
		Stateless:    true,
	})
	return h
}

func (h *Handler) saveLink(ctx context.Context, _ *mcp.CallToolRequest, in saveLinkInput) (*mcp.CallToolResult, saveLinkOutput, error) {
	var publishedAt *time.Time
	if in.PublishedAt != nil && *in.PublishedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, *in.PublishedAt)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, *in.PublishedAt)
			if err != nil {
				return nil, saveLinkOutput{}, apperr.InvalidInput.New("publishedAt must be RFC3339")
			}
		}
		utc := parsed.UTC()
		publishedAt = &utc
	}

	result, err := h.svc.SaveLink(ctx, service.SaveLinkInput{
		Title:       in.Title,
		Link:        in.Link,
		Description: in.Description,
		Thumbnail:   in.Thumbnail,
		PublishedAt: publishedAt,
	})
	if err != nil {
		return nil, saveLinkOutput{}, err
	}
	var link *string
	if result.Entry.Link != "" {
		link = &result.Entry.Link
	}
	return nil, saveLinkOutput{
		ID:      result.Entry.ID.String(),
		Title:   result.Entry.Title,
		Link:    link,
		Created: result.Created,
	}, nil
}
