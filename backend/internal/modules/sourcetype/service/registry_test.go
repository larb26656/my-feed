package service_test

import (
	"testing"

	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"

	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	"github.com/stretchr/testify/require"
)

func TestStartupRejectsInvalidSourceTypeDefinitions(t *testing.T) {
	invalid := rss.Definition(nil)
	invalid.UI["url"] = map[string]any{"label": "URL", "widget": "unsupported"}
	_, err := service.New(invalid)
	require.Error(t, err)
}

func TestStartupRejectsWidgetTypeMismatch(t *testing.T) {
	definition := rss.Definition(nil)
	definition.UI["url"] = map[string]any{"label": "URL", "widget": "checkbox"}
	_, err := service.New(definition)
	require.Error(t, err)
}

func TestStartupRejectsMissingObtainHandler(t *testing.T) {
	definition := rss.Definition(nil)
	definition.Obtainer = nil
	_, err := service.New(definition)
	require.Error(t, err)
}

func TestStartupRejectsInvalidFieldOrder(t *testing.T) {
	for _, order := range [][]string{nil, {"missing"}, {"url", "url"}} {
		definition := rss.Definition(nil)
		definition.FieldOrder = order
		_, err := service.New(definition)
		require.ErrorContains(t, err, "field order")
	}
}
