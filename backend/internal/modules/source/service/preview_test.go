package service_test

import (
	"context"
	"errors"
	"testing"

	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/obtain/api"
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	"github.com/stretchr/testify/require"
)

type failingObtain struct{ cause error }

func (p failingObtain) Obtain(context.Context, *obtainapi.Input) ([]obtainapi.PreparedEntry, error) {
	return nil, p.cause
}

func TestPreviewPreservesOriginFailure(t *testing.T) {
	cause := errors.New("origin unavailable")
	svc := sourcesvc.New(nil)
	svc.SetObtainer(failingObtain{cause})
	_, err := svc.Preview(context.Background(), sourceapi.PreviewInput{Type: "rss", RSSURL: "https://example.test/feed"})
	require.ErrorIs(t, err, cause)
}
