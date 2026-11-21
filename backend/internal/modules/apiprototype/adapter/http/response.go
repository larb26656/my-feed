package http

import "github.com/larb26656/my-feed/backend/internal/modules/apiprototype/domain"

type bookmarkResponse struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Read  bool   `json:"read"`
}

func response(b domain.Bookmark) bookmarkResponse {
	return bookmarkResponse{ID: b.ID, Title: b.Title, URL: b.URL, Read: b.Read}
}
