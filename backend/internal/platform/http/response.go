package httpx

type SuccessResponse[T any] struct {
	Success bool `json:"success"`
	Data    T    `json:"data"`
	Meta    any  `json:"meta,omitempty"`
}

func Success[T any](data T) SuccessResponse[T] {
	return SuccessResponse[T]{
		Success: true,
		Data:    data,
	}
}

func SuccessWithMeta[T any](
	data T,
	meta any,
) SuccessResponse[T] {
	return SuccessResponse[T]{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
}
