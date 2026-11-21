package http

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/larb26656/my-feed/backend/internal/modules/apiprototype/adapter/memory"
	"github.com/larb26656/my-feed/backend/internal/modules/apiprototype/service"
	httpx "github.com/larb26656/my-feed/backend/internal/platform/http"
)

func TestListQueryBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, raw string
		valid     bool
		size      int
	}{
		{"defaults", "/", true, 10},
		{"filters and sort", "/?titleLike=go&statusEqual=unread&sortBy=title&sortOrder=desc&pageSize=100", true, 100},
		{"empty page size", "/?pageSize=", false, 0},
		{"zero page size", "/?pageSize=0", false, 0},
		{"large page size", "/?pageSize=101", false, 0},
		{"invalid number", "/?pageSize=nope", false, 0},
		{"invalid status", "/?statusEqual=other", false, 0},
		{"invalid sort", "/?sortBy=url", false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", tc.raw, nil)
			query := listBookmarksQuery{CursorPageOption: httpx.NewCursorPageOption()}
			err := c.ShouldBindQuery(&query)
			if (err == nil) != tc.valid {
				t.Fatalf("binding error = %v", err)
			}
			if tc.valid && query.PageSize != tc.size {
				t.Fatalf("page size = %d", query.PageSize)
			}
		})
	}
}

func TestListBookmarksAppliesQuery(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	repo := memory.New()
	svc := service.New(repo)
	for _, item := range []struct {
		title, url string
		read       bool
	}{
		{"Go Alpha", "https://example.com/a", false},
		{"Go Beta", "https://example.com/b", false},
		{"Go Read", "https://example.com/c", true},
		{"Other", "https://example.com/d", false},
	} {
		bookmark, err := svc.Create(context.Background(), item.title, item.url)
		if err != nil {
			t.Fatal(err)
		}
		if item.read {
			if _, err := svc.MarkRead(context.Background(), bookmark.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	router := gin.New()
	New(svc).RegisterRoutes(router.Group(""))

	// Act
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/bookmarks?titleLike=go&statusEqual=unread&sortBy=title&sortOrder=desc&pageSize=1", nil))

	// Assert
	var body struct {
		Data []bookmarkResponse `json:"data"`
		Meta struct {
			PageSize   int    `json:"pageSize"`
			NextCursor string `json:"nextCursor"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || len(body.Data) != 1 || body.Data[0].Title != "Go Beta" || body.Meta.PageSize != 1 || body.Meta.NextCursor != body.Data[0].ID {
		t.Fatalf("unexpected response: status %d, body %s", response.Code, response.Body.String())
	}
}

func TestListBookmarksContinuesFilteredSortedPage(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	svc := service.New(memory.New())
	for _, title := range []string{"Go Alpha", "Go Beta", "Other"} {
		if _, err := svc.Create(context.Background(), title, "https://example.com/"+url.PathEscape(title)); err != nil {
			t.Fatal(err)
		}
	}
	router := gin.New()
	New(svc).RegisterRoutes(router.Group(""))
	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest("GET", "/bookmarks?titleLike=go&sortBy=title&sortOrder=desc&pageSize=1", nil))
	var page struct {
		Meta struct {
			NextCursor string `json:"nextCursor"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &page); err != nil || page.Meta.NextCursor == "" {
		t.Fatalf("first page: %s", first.Body.String())
	}

	// Act
	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest("GET", "/bookmarks?titleLike=go&sortBy=title&sortOrder=desc&pageSize=1&cursor="+url.QueryEscape(page.Meta.NextCursor), nil))

	// Assert
	var body struct {
		Data []bookmarkResponse `json:"data"`
		Meta struct {
			NextCursor string `json:"nextCursor"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if second.Code != 200 || len(body.Data) != 1 || body.Data[0].Title != "Go Alpha" || body.Meta.NextCursor != "" {
		t.Fatalf("second page: %s", second.Body.String())
	}
}
