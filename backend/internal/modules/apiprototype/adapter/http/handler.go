// Package http exposes a disposable API for discussing response conventions.
package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/modules/apiprototype/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/bookmarks", h.create)
	r.GET("/bookmarks", h.list)
	r.GET("/bookmarks/:id", h.get)
	r.POST("/bookmarks/:id/read", h.markRead)
}

func (h *Handler) create(c *gin.Context) {
	var req createBookmarkRequest

	if !httpx.BindJSON(c, &req) {
		return
	}

	item, err := h.svc.Create(c.Request.Context(), req.Title, req.URL)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, httpx.Success(response(item)))
}

func (h *Handler) get(c *gin.Context) {
	item, err := h.svc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, httpx.Success(response(item)))
}

func (h *Handler) markRead(c *gin.Context) {
	item, err := h.svc.MarkRead(c.Request.Context(), c.Param("id"))
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, httpx.Success(response(item)))
}

func (h *Handler) list(c *gin.Context) {
	query := listBookmarksQuery{CursorPageOption: httpx.NewCursorPageOption()}
	if err := c.ShouldBindQuery(&query); err != nil {
		_ = c.Error(service.ErrInvalid)
		return
	}

	items, cursor, err := h.svc.List(c.Request.Context(), service.ListInput{
		Cursor: query.Cursor, PageSize: query.PageSize,
		TitleLike: query.TitleLike, StatusEqual: query.StatusEqual,
		SortBy: query.SortBy, SortOrder: query.SortOrder,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	result := make([]bookmarkResponse, 0, len(items))
	for _, item := range items {
		result = append(result, response(item))
	}

	c.JSON(http.StatusOK, httpx.Cursor(result, query.PageSize, cursor))
}
