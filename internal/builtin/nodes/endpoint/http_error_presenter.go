package endpoint

import (
	"errors"
	"net/http"
	"strings"

	"github.com/neohetj/matrix/pkg/types"
)

// defaultPublicErrorMessage 把 HTTP 状态码映射为不包含内部实现细节的兜底文案。
func defaultPublicErrorMessage(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return "invalid request"
	case http.StatusUnauthorized:
		return "authentication required"
	case http.StatusForbidden:
		return "permission denied"
	case http.StatusNotFound:
		return "resource not found"
	case http.StatusMethodNotAllowed:
		return "method not allowed"
	case http.StatusConflict:
		return "request conflict"
	case http.StatusRequestEntityTooLarge:
		return "request too large"
	case http.StatusUnsupportedMediaType:
		return "unsupported media type"
	case http.StatusTooManyRequests:
		return "too many requests"
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		return "service unavailable"
	case http.StatusGatewayTimeout:
		return "gateway timeout"
	default:
		if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
			return "request failed"
		}
		return "internal server error"
	}
}

// newPublicErrorResponse 只读取显式标记为安全的 UserMessage，不序列化 Cause 或 FailureInfo.Error。
func newPublicErrorResponse(statusCode int, err error) ErrorResponse {
	message := defaultPublicErrorMessage(statusCode)

	var serviceErr *types.ServiceError
	if errors.As(err, &serviceErr) {
		if publicMessage := strings.TrimSpace(serviceErr.UserMessage); publicMessage != "" {
			message = publicMessage
		}
	}

	return ErrorResponse{
		Code:    statusCode,
		Message: message,
	}
}
