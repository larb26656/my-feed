package customtype

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	typeapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	"github.com/stretchr/testify/require"
)

type Config struct {
	Text    string  `json:"text"`
	URL     string  `json:"url"`
	Count   float64 `json:"count"`
	Choice  int     `json:"choice"`
	Enabled bool    `json:"enabled"`
}
type (
	ObtainInput struct{ Config Config }
	obtainer    struct{}
)

func (obtainer) Obtain(_ context.Context, in *typeapi.Input) ([]typeapi.PreparedEntry, error) {
	input := ObtainInput{}
	raw, err := json.Marshal(in.Config)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &input.Config); err != nil {
		return nil, err
	}
	return []typeapi.PreparedEntry{{Title: input.Config.Text, Link: "https://example.test/custom"}}, nil
}

func Definition(t testing.TB) typeapi.Definition {
	t.Helper()
	_, here, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(here), "../../../../testdata/source-custom-schema.json"))
	require.NoError(t, err)
	var schema map[string]any
	require.NoError(t, json.Unmarshal(raw, &schema))
	return typeapi.Definition{Code: "custom", Label: "Custom", Schema: schema, FieldOrder: []string{"text", "url", "count", "choice", "enabled"}, UI: map[string]any{"text": map[string]any{"label": "Text", "widget": "text"}, "url": map[string]any{"label": "URL", "widget": "url"}, "count": map[string]any{"label": "Count", "widget": "number"}, "choice": map[string]any{"label": "Choice", "widget": "select"}, "enabled": map[string]any{"label": "Enabled", "widget": "checkbox"}}, Normalize: func(_ context.Context, c map[string]any) (map[string]any, string, error) {
		var config Config
		raw, err := json.Marshal(c)
		if err != nil {
			return nil, "", err
		}
		if err = json.Unmarshal(raw, &config); err != nil {
			return nil, "", err
		}
		return c, config.Text, nil
	}, Obtainer: obtainer{}}
}

func Input(text string) map[string]any {
	return map[string]any{"text": text, "url": "https://example.test", "count": 3, "choice": 2, "enabled": true}
}
