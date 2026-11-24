package repository

import sortoption "github.com/larb26656/my-feed/backend/internal/platform/sort"

type (
	ListSourcesFilter  struct{ IncludeDeactivated bool }
	SourceListCriteria struct {
		NameLike, TypeEqual string
		ActiveEqual         *bool
	}
)

type (
	SourceListSort struct{ NameSort, TypeSort, ActiveSort, CreatedAtSort sortoption.Direction }
	SourcePage     struct{ Page, PageSize int }
)
