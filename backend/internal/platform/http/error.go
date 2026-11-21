package httpx

type ErrorResponse struct {
	Success bool       `json:"success"`
	Error   ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  any    `json:"detail,omitempty"`
}

func Fail(
	code string,
	message string,
	details ...any,
) ErrorResponse {
	var detail any
	if len(details) > 0 {
		detail = details[0]
	}

	return ErrorResponse{
		Success: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Detail:  detail,
		},
	}
}
