package httpx

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPageOptionBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, query string
		valid       bool
		page, size  int
	}{
		{"defaults", "", true, 1, 10},
		{"explicit page", "?page=2&pageSize=100", true, 2, 100},
		{"zero page", "?page=0", false, 0, 0},
		{"empty page", "?page=", false, 0, 0},
		{"invalid size", "?pageSize=101", false, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/"+tc.query, nil)
			option := NewPageOption()
			err := c.ShouldBindQuery(&option)
			if (err == nil) != tc.valid {
				t.Fatalf("binding error: %v", err)
			}
			if tc.valid && (option.Page != tc.page || option.PageSize != tc.size) {
				t.Fatalf("unexpected page option: %+v", option)
			}
		})
	}
}

func TestCursorPageOptionBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, query, cursor string
		valid               bool
		size                int
	}{
		{"defaults", "", "", true, 10},
		{"explicit cursor", "?cursor=abc&pageSize=50", "abc", true, 50},
		{"cursor only", "?cursor=next-token", "next-token", true, 10},
		{"min size", "?pageSize=1", "", true, 1},
		{"max size", "?pageSize=100", "", true, 100},
		{"zero size", "?pageSize=0", "", false, 0},
		{"empty size", "?pageSize=", "", false, 0},
		{"invalid size", "?pageSize=101", "", false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/"+tc.query, nil)
			option := NewCursorPageOption()
			err := c.ShouldBindQuery(&option)
			if (err == nil) != tc.valid {
				t.Fatalf("binding error: %v", err)
			}
			if tc.valid && (option.Cursor != tc.cursor || option.PageSize != tc.size) {
				t.Fatalf("unexpected cursor page option: %+v", option)
			}
		})
	}
}
