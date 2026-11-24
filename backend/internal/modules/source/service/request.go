package service

import (
	sourceapi "github.com/larb26656/my-feed/backend/internal/modules/source/api"
	"github.com/larb26656/my-feed/backend/internal/modules/source/repository"
)

type (
	CreateSourceInput    = sourceapi.CreateSourceInput
	ListSourcesInput     struct{ IncludeDeactivated bool }
	ListSourcesPageInput struct {
		Criteria repository.SourceListCriteria
		Sort     repository.SourceListSort
		Page     repository.SourcePage
	}
)

type UpdateSourceInput struct {
	Config map[string]any
	Name   string
}
