package http

import httpx "github.com/larb26656/my-feed/backend/internal/platform/http"

type createBookmarkRequest struct {
	Title string `json:"title" binding:"required"`
	URL   string `json:"url" binding:"required,url"`
}

type listBookmarkCriteria struct {
	TitleLike   string `form:"titleLike" binding:"omitempty,max=200"`
	StatusEqual string `form:"statusEqual" binding:"omitempty,oneof=read unread"`
}

type listBookmarkSort struct {
	SortBy    string `form:"sortBy" binding:"omitempty,oneof=createdAt title"`
	SortOrder string `form:"sortOrder" binding:"omitempty,oneof=asc desc"`
}

type listBookmarksQuery struct {
	httpx.CursorPageOption
	listBookmarkCriteria
	listBookmarkSort
}
