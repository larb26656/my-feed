package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/larb26656/quick-feed/backend/internal/platform/apperr"
	"github.com/larb26656/quick-feed/backend/internal/source/domain"
	"github.com/larb26656/quick-feed/backend/internal/source/repository"
	"github.com/larb26656/quick-feed/backend/internal/source/service"
)

type fakeRepo struct {
	sources map[uuid.UUID]*domain.Source
	byURL   map[string]uuid.UUID
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		sources: make(map[uuid.UUID]*domain.Source),
		byURL:   make(map[string]uuid.UUID),
	}
}

func (f *fakeRepo) CreateSource(_ context.Context, s *domain.Source) error {
	if s.Type == domain.TypeManual {
		for _, existing := range f.sources {
			if existing.Type == domain.TypeManual {
				return apperr.Conflict("manual source already exists")
			}
		}
	}
	if s.RSSURL != "" {
		if _, ok := f.byURL[s.RSSURL]; ok {
			return apperr.Conflict("rss url already in use")
		}
	}
	cp := *s
	f.sources[s.ID] = &cp
	if s.RSSURL != "" {
		f.byURL[s.RSSURL] = s.ID
	}
	return nil
}

func (f *fakeRepo) GetSourceByID(_ context.Context, id uuid.UUID) (*domain.Source, error) {
	s, ok := f.sources[id]
	if !ok {
		return nil, apperr.NotFound("source not found")
	}
	cp := *s
	return &cp, nil
}

func (f *fakeRepo) ListSources(_ context.Context, filter repository.ListSourcesFilter) ([]*domain.Source, error) {
	out := make([]*domain.Source, 0, len(f.sources))
	for _, s := range f.sources {
		if !filter.IncludeDeactivated && !s.Active {
			continue
		}
		cp := *s
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeRepo) UpdateSource(_ context.Context, s *domain.Source) error {
	if _, ok := f.sources[s.ID]; !ok {
		return apperr.NotFound("source not found")
	}
	cp := *s
	f.sources[s.ID] = &cp
	return nil
}

func (f *fakeRepo) GetManualSource(_ context.Context) (*domain.Source, error) {
	for _, s := range f.sources {
		if s.Type == domain.TypeManual {
			cp := *s
			return &cp, nil
		}
	}
	return nil, apperr.NotFound("source not found")
}

func (f *fakeRepo) WithTx(_ pgx.Tx) repository.Repository {
	return f
}

func TestCreateRSSSourceExposesTypeAndURLOnCreateGetList(t *testing.T) {
	svc := service.New(newFakeRepo())
	feedURL := "https://hn.example/rss"

	created, err := svc.CreateSource(context.Background(), service.CreateSourceInput{
		Name:   "HN RSS",
		Type:   "rss",
		RSSURL: feedURL,
	})
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, created.ID)
	require.Equal(t, "HN RSS", created.Name)
	require.Equal(t, domain.TypeRSS, created.Type)
	require.Equal(t, feedURL, created.RSSURL)

	got, err := svc.GetSource(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, domain.TypeRSS, got.Type)
	require.Equal(t, feedURL, got.RSSURL)

	listed, err := svc.ListSources(context.Background(), service.ListSourcesInput{})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, created.ID, listed[0].ID)
	require.Equal(t, domain.TypeRSS, listed[0].Type)
	require.Equal(t, feedURL, listed[0].RSSURL)
	require.True(t, listed[0].Active)
}

func TestDeactivateRSSOmitsFromDefaultListButGetByIDWorks(t *testing.T) {
	svc := service.New(newFakeRepo())
	created, err := svc.CreateSource(context.Background(), service.CreateSourceInput{
		Name:   "HN RSS",
		Type:   "rss",
		RSSURL: "https://hn.example/rss",
	})
	require.NoError(t, err)

	deactivated, err := svc.DeactivateSource(context.Background(), created.ID)
	require.NoError(t, err)
	require.False(t, deactivated.Active)

	listed, err := svc.ListSources(context.Background(), service.ListSourcesInput{})
	require.NoError(t, err)
	require.Empty(t, listed)

	got, err := svc.GetSource(context.Background(), created.ID)
	require.NoError(t, err)
	require.False(t, got.Active)
	require.Equal(t, created.ID, got.ID)
}

func TestListSourcesCanIncludeDeactivated(t *testing.T) {
	svc := service.New(newFakeRepo())
	created, err := svc.CreateSource(context.Background(), service.CreateSourceInput{
		Name:   "HN RSS",
		Type:   "rss",
		RSSURL: "https://hn.example/rss",
	})
	require.NoError(t, err)
	_, err = svc.DeactivateSource(context.Background(), created.ID)
	require.NoError(t, err)

	listed, err := svc.ListSources(context.Background(), service.ListSourcesInput{
		IncludeDeactivated: true,
	})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, created.ID, listed[0].ID)
	require.False(t, listed[0].Active)
}

func TestReactivateRSSReturnsToActiveList(t *testing.T) {
	svc := service.New(newFakeRepo())
	created, err := svc.CreateSource(context.Background(), service.CreateSourceInput{
		Name:   "HN RSS",
		Type:   "rss",
		RSSURL: "https://hn.example/rss",
	})
	require.NoError(t, err)
	_, err = svc.DeactivateSource(context.Background(), created.ID)
	require.NoError(t, err)

	reactivated, err := svc.ReactivateSource(context.Background(), created.ID)
	require.NoError(t, err)
	require.True(t, reactivated.Active)

	listed, err := svc.ListSources(context.Background(), service.ListSourcesInput{})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, created.ID, listed[0].ID)
	require.True(t, listed[0].Active)
}

func TestDeactivateManualSourceRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo)

	manual, err := domain.NewManualSource("Saved links")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(context.Background(), manual))

	_, err = svc.DeactivateSource(context.Background(), manual.ID)
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)

	got, err := svc.GetSource(context.Background(), manual.ID)
	require.NoError(t, err)
	require.True(t, got.Active)
}

func TestReactivateManualSourceRejected(t *testing.T) {
	repo := newFakeRepo()
	svc := service.New(repo)

	manual, err := domain.NewManualSource("Saved links")
	require.NoError(t, err)
	require.NoError(t, repo.CreateSource(context.Background(), manual))

	_, err = svc.ReactivateSource(context.Background(), manual.ID)
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeFailedPrecondition, ae.Code)

	got, err := svc.GetSource(context.Background(), manual.ID)
	require.NoError(t, err)
	require.True(t, got.Active)
}

func TestDeactivateMissingSourceNotFound(t *testing.T) {
	svc := service.New(newFakeRepo())
	_, err := svc.DeactivateSource(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestGetSourceMissingIDNotFound(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.GetSource(context.Background(), uuid.New())
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeNotFound, ae.Code)
}

func TestCreateSourceRejectsManualType(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.CreateSource(context.Background(), service.CreateSourceInput{
		Name: "Saved links",
		Type: "manual",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestCreateRSSSourceRejectsMissingURL(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.CreateSource(context.Background(), service.CreateSourceInput{
		Name:   "HN RSS",
		Type:   "rss",
		RSSURL: "  ",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)
}

func TestCreateRSSSourceRejectsDuplicateURL(t *testing.T) {
	svc := service.New(newFakeRepo())
	feedURL := "https://hn.example/rss"

	_, err := svc.CreateSource(context.Background(), service.CreateSourceInput{
		Name:   "HN RSS",
		Type:   "rss",
		RSSURL: feedURL,
	})
	require.NoError(t, err)

	_, err = svc.CreateSource(context.Background(), service.CreateSourceInput{
		Name:   "HN again",
		Type:   "rss",
		RSSURL: feedURL,
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeConflict, ae.Code)
}

func TestEnsureSavedLinksCreatesSingletonVisibleOnListAndGet(t *testing.T) {
	svc := service.New(newFakeRepo())

	ensured, err := svc.EnsureSavedLinks(context.Background())
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, ensured.ID)
	require.Equal(t, "Saved links", ensured.Name)
	require.Equal(t, domain.TypeManual, ensured.Type)
	require.Empty(t, ensured.RSSURL)

	got, err := svc.GetSource(context.Background(), ensured.ID)
	require.NoError(t, err)
	require.Equal(t, ensured.ID, got.ID)
	require.Equal(t, "Saved links", got.Name)
	require.Equal(t, domain.TypeManual, got.Type)

	listed, err := svc.ListSources(context.Background(), service.ListSourcesInput{})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, ensured.ID, listed[0].ID)
	require.Equal(t, domain.TypeManual, listed[0].Type)
}

func TestEnsureSavedLinksIsIdempotent(t *testing.T) {
	svc := service.New(newFakeRepo())

	first, err := svc.EnsureSavedLinks(context.Background())
	require.NoError(t, err)

	second, err := svc.EnsureSavedLinks(context.Background())
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)

	listed, err := svc.ListSources(context.Background(), service.ListSourcesInput{})
	require.NoError(t, err)
	require.Len(t, listed, 1)
}

func TestEnsureSavedLinksDoesNotAllowReaderManualCreate(t *testing.T) {
	svc := service.New(newFakeRepo())

	_, err := svc.EnsureSavedLinks(context.Background())
	require.NoError(t, err)

	_, err = svc.CreateSource(context.Background(), service.CreateSourceInput{
		Name: "Saved links",
		Type: "manual",
	})
	require.Error(t, err)
	var ae *apperr.Error
	require.ErrorAs(t, err, &ae)
	require.Equal(t, apperr.CodeValidation, ae.Code)

	listed, err := svc.ListSources(context.Background(), service.ListSourcesInput{})
	require.NoError(t, err)
	require.Len(t, listed, 1)
}
