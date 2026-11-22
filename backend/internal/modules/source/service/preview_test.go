package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	rss "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/adapter/rss"
	obtainapi "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/api"
	typesvc "github.com/larb26656/my-feed/backend/internal/modules/sourcetype/service"
	"github.com/stretchr/testify/require"
)

type failingObtain struct{ cause error }

func (p failingObtain) Obtain(context.Context, *obtainapi.Input) ([]obtainapi.PreparedEntry, error) {
	return nil, p.cause
}

func TestPreviewPreservesOriginFailure(t *testing.T) {
	cause := errors.New("origin unavailable")
	svc := sourcesvc.New(nil)
	registry, err := typesvc.New(rss.Definition(nil))
	require.NoError(t, err)
	svc.SetSourceTypes(registry)
	svc.SetObtainer(failingObtain{cause})
	_, err = svc.Preview(context.Background(), sourceapi.PreviewInput{Type: "rss", Config: map[string]any{"url": "https://example.test/feed"}})
	require.ErrorIs(t, err, cause)
}

type deadlineObtain struct{ t *testing.T }

func (p deadlineObtain) Obtain(ctx context.Context, _ *obtainapi.Input) ([]obtainapi.PreparedEntry, error) {
	deadline, ok := ctx.Deadline()
	require.True(p.t, ok)
	remaining := time.Until(deadline)
	require.Greater(p.t, remaining, 14*time.Second)
	require.LessOrEqual(p.t, remaining, 15*time.Second)
	return nil, context.DeadlineExceeded
}

func TestPreviewSetsFifteenSecondTimeout(t *testing.T) {
	svc := sourcesvc.New(nil)
	registry, err := typesvc.New(rss.Definition(nil))
	require.NoError(t, err)
	svc.SetSourceTypes(registry)
	svc.SetObtainer(deadlineObtain{t})
	_, err = svc.Preview(context.Background(), sourceapi.PreviewInput{Type: "rss", Config: map[string]any{"url": "https://example.test/feed"}})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
