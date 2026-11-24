package service

import "github.com/larb26656/my-feed/backend/internal/modules/fetch/domain"

type ListFetchJobsResult struct {
	Jobs       []*domain.FetchJob
	NextCursor *string
}
