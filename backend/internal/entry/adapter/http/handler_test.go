package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	entryhttp "github.com/larb26656/quick-feed/backend/internal/entry/adapter/http"
	"github.com/larb26656/quick-feed/backend/internal/entry/domain"
	"github.com/larb26656/quick-feed/backend/internal/entry/service"
)

type stubEntryAPI struct {
	ingest func(ctx context.Context, in service.IngestEntryInput) (*service.IngestEntryResult, error)
}

func (s stubEntryAPI) SaveLink(context.Context, service.SaveLinkInput) (*domain.Entry, error) {
	panic("unexpected SaveLink")
}
func (s stubEntryAPI) GetEntry(context.Context, uuid.UUID) (*domain.Entry, error) {
	panic("unexpected GetEntry")
}
func (s stubEntryAPI) ListEntries(context.Context, service.ListEntriesInput) (*service.ListEntriesResult, error) {
	panic("unexpected ListEntries")
}
func (s stubEntryAPI) CountSavedEntries(context.Context) (int, error) {
	panic("unexpected CountSavedEntries")
}
func (s stubEntryAPI) MarkEntryRead(context.Context, uuid.UUID) (*domain.Entry, error) {
	panic("unexpected MarkEntryRead")
}
func (s stubEntryAPI) ClearEntryRead(context.Context, uuid.UUID) (*domain.Entry, error) {
	panic("unexpected ClearEntryRead")
}
func (s stubEntryAPI) MarkEntrySaved(context.Context, uuid.UUID) (*domain.Entry, error) {
	panic("unexpected MarkEntrySaved")
}
func (s stubEntryAPI) ClearEntrySaved(context.Context, uuid.UUID) (*domain.Entry, error) {
	panic("unexpected ClearEntrySaved")
}
func (s stubEntryAPI) IngestEntry(ctx context.Context, in service.IngestEntryInput) (*service.IngestEntryResult, error) {
	return s.ingest(ctx, in)
}

func TestIngestEntryReturns201WhenCreated(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	h := entryhttp.NewHandler(stubEntryAPI{
		ingest: func(_ context.Context, in service.IngestEntryInput) (*service.IngestEntryResult, error) {
			require.Equal(t, sourceID, in.SourceID)
			return &service.IngestEntryResult{
				Entry: &domain.Entry{
					ID: id, SourceID: sourceID, Title: in.Title, Link: in.Link,
					CreatedAt: now, UpdatedAt: now,
				},
				Created: true,
			}, nil
		},
	})

	body, err := json.Marshal(map[string]any{
		"sourceId": sourceID.String(),
		"title":    "Hello",
		"link":     "https://example.com/a",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
}

func TestIngestEntryReturns200WhenRefreshed(t *testing.T) {
	id := uuid.New()
	sourceID := uuid.New()
	now := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	h := entryhttp.NewHandler(stubEntryAPI{
		ingest: func(_ context.Context, in service.IngestEntryInput) (*service.IngestEntryResult, error) {
			return &service.IngestEntryResult{
				Entry: &domain.Entry{
					ID: id, SourceID: sourceID, Title: in.Title, Link: in.Link,
					CreatedAt: now, UpdatedAt: now,
				},
				Created: false,
			}, nil
		},
	})

	body, err := json.Marshal(map[string]any{
		"sourceId": sourceID.String(),
		"title":    "Hello",
		"link":     "https://example.com/a",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Routes().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
