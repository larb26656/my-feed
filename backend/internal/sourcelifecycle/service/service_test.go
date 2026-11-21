package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	fetchdomain "github.com/larb26656/my-feed/backend/internal/modules/fetch/domain"
	fetchrepo "github.com/larb26656/my-feed/backend/internal/modules/fetch/repository"
	fetchsvc "github.com/larb26656/my-feed/backend/internal/modules/fetch/service"
	sourcedomain "github.com/larb26656/my-feed/backend/internal/modules/source/domain"
	sourcerepo "github.com/larb26656/my-feed/backend/internal/modules/source/repository"
	sourcesvc "github.com/larb26656/my-feed/backend/internal/modules/source/service"
	"github.com/larb26656/my-feed/backend/internal/platform/database"
	sourcelifecycle "github.com/larb26656/my-feed/backend/internal/sourcelifecycle/service"
)

type transactionTestTransactor struct {
	tx         database.Transaction
	rolledBack bool
}

func (d *transactionTestTransactor) WithinTransaction(ctx context.Context, fn func(context.Context, database.Transaction) error) error {
	err := fn(ctx, d.tx)
	d.rolledBack = err != nil
	return err
}

type transactionTestSourceRepo struct {
	sourcerepo.Repository
	source *sourcedomain.Source
	tx     database.Transaction
}

func (r *transactionTestSourceRepo) WithTx(tx database.Transaction) sourcerepo.Repository {
	r.tx = tx
	return r
}

func (r *transactionTestSourceRepo) GetSourceByID(context.Context, uuid.UUID) (*sourcedomain.Source, error) {
	return r.source, nil
}

func (r *transactionTestSourceRepo) UpdateSource(context.Context, *sourcedomain.Source) error {
	return nil
}

type transactionTestFetchRepo struct {
	fetchrepo.Repository
	job *fetchdomain.FetchJob
	tx  database.Transaction
}

func (r *transactionTestFetchRepo) WithTx(tx database.Transaction) fetchrepo.Repository {
	r.tx = tx
	return r
}

func (r *transactionTestFetchRepo) GetUnfinishedFetchJobBySourceID(context.Context, uuid.UUID) (*fetchdomain.FetchJob, error) {
	return r.job, nil
}

func (r *transactionTestFetchRepo) UpdateFetchJob(context.Context, *fetchdomain.FetchJob) error {
	return nil
}

func TestDeactivateSourceUpdatesSourceAndPendingFetchJobInOneTransaction(t *testing.T) {
	ctx := context.Background()
	source, err := sourcedomain.NewRSSSource("A source", "https://example.com/feed.xml")
	require.NoError(t, err)
	job, err := fetchdomain.NewFetchJob(source.ID)
	require.NoError(t, err)
	tx := new(struct{ marker string })
	db := &transactionTestTransactor{tx: tx}
	sourceRepo := &transactionTestSourceRepo{source: source}
	fetchRepository := &transactionTestFetchRepo{job: job}
	fetchService := fetchsvc.New(fetchRepository, nil, nil, nil, nil, time.Minute, time.Minute)
	api := sourcelifecycle.New(db, sourcesvc.New(sourceRepo), fetchService)

	_, err = api.DeactivateSource(ctx, source.ID)
	require.NoError(t, err)
	require.Same(t, tx, sourceRepo.tx)
	require.Same(t, tx, fetchRepository.tx)
	require.False(t, source.Active)
	require.Equal(t, fetchdomain.StatusFailed, job.Status)
	require.False(t, db.rolledBack)
}

type failingTransactionTestFetchRepo struct {
	*transactionTestFetchRepo
	err error
}

func (r *failingTransactionTestFetchRepo) WithTx(tx database.Transaction) fetchrepo.Repository {
	r.tx = tx
	return r
}

func (r *failingTransactionTestFetchRepo) UpdateFetchJob(context.Context, *fetchdomain.FetchJob) error {
	return r.err
}

func TestDeactivateSourcePropagatesFetchJobFailureThroughTransaction(t *testing.T) {
	ctx := context.Background()
	source, err := sourcedomain.NewRSSSource("A source", "https://example.com/feed.xml")
	require.NoError(t, err)
	job, err := fetchdomain.NewFetchJob(source.ID)
	require.NoError(t, err)
	tx := new(struct{ marker string })
	db := &transactionTestTransactor{tx: tx}
	sourceRepo := &transactionTestSourceRepo{source: source}
	failure := errors.New("fetch job update failed")
	fetchRepository := &failingTransactionTestFetchRepo{transactionTestFetchRepo: &transactionTestFetchRepo{job: job}, err: failure}
	api := sourcelifecycle.New(db, sourcesvc.New(sourceRepo), fetchsvc.New(fetchRepository, nil, nil, nil, nil, time.Minute, time.Minute))

	_, err = api.DeactivateSource(ctx, source.ID)
	require.ErrorIs(t, err, failure)
	require.True(t, db.rolledBack)
}
