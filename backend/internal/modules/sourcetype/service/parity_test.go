package service_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/youtube"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	"github.com/larb26656/my-feed/backend/internal/testkit/customtype"
	"github.com/stretchr/testify/require"
)

func TestSharedConfigCases(t *testing.T) {
	data, err := os.ReadFile("../../../../../testdata/source-config-cases.json")
	require.NoError(t, err)
	var cases []struct {
		Type   string         `json:"type"`
		Config map[string]any `json:"config"`
		Valid  bool           `json:"valid"`
	}
	require.NoError(t, json.Unmarshal(data, &cases))
	custom := customtype.Definition(t)
	registry, err := service.New(rss.Definition(nil), youtube.Definition(nil), custom)
	require.NoError(t, err)
	for _, tc := range cases {
		_, _, err := registry.Prepare(tc.Type, tc.Config)
		require.Equal(t, tc.Valid, err == nil, "%s: %v", tc.Type, tc.Config)
	}
}
