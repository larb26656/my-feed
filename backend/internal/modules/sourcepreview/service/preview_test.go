package service_test

import (
	"context"
	"errors"
	"testing"

	fetchapi "github.com/larb26656/my-feed/backend/internal/modules/fetch/api"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	previewapi "github.com/larb26656/my-feed/backend/internal/modules/sourcepreview/api"
	"github.com/larb26656/my-feed/backend/internal/modules/sourcepreview/service"
	"github.com/stretchr/testify/require"
)

type failingObtain struct{ cause error }

func (p failingObtain) Obtain(context.Context, *sourceapi.SourceRef) ([]fetchapi.PreparedEntry, error) {
	return nil, p.cause
}

func TestPreviewPreservesOriginFailure(t *testing.T) {
	cause := errors.New("origin unavailable")
	_, err := service.New(sourcesvc.New(nil), failingObtain{cause}).Preview(context.Background(), previewapi.Input{Type: "rss", RSSURL: "https://example.test/feed"})
	require.ErrorIs(t, err, cause)
}
