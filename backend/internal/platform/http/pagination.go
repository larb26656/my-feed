package httpx

type PageOption struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"pageSize" binding:"min=1,max=100"`
}

func NewPageOption() PageOption { return PageOption{Page: 1, PageSize: 10} }

type CursorPageOption struct {
	Cursor   string `form:"cursor"`
	PageSize int    `form:"pageSize" binding:"min=1,max=100"`
}

func NewCursorPageOption() CursorPageOption { return CursorPageOption{PageSize: 10} }

type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}

type CursorMeta struct {
	NextCursor string `json:"nextCursor,omitempty"`
	PageSize   int    `json:"pageSize"`
}

func Page[T any](
	data []T,
	page int,
	pageSize int,
	total int64,
) SuccessResponse[[]T] {
	var totalPages int64

	if pageSize > 0 {
		totalPages = (total + int64(pageSize) - 1) / int64(pageSize)
	}

	return SuccessWithMeta(data, PageMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	})
}

func Cursor[T any](
	data []T,
	pageSize int,
	nextCursor string,
) SuccessResponse[[]T] {
	return SuccessWithMeta(data, CursorMeta{
		NextCursor: nextCursor,
		PageSize:   pageSize,
	})
}
