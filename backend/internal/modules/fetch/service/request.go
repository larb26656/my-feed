package service

import "github.com/larb26656/my-feed/backend/internal/modules/fetch/repository"

type ListFetchJobsInput struct {
	Cursor   *string
	PageSize *int
	Criteria repository.FetchJobListCriteria
}
