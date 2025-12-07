package response

import (
	"net/http"
)

// Response represents the standard API response structure
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorInfo represents error details in the response
type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// Meta represents metadata for paginated responses
type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginationParams represents pagination input parameters
type PaginationParams struct {
	Page    int
	PerPage int
}

// DefaultPagination returns default pagination values
func DefaultPagination() PaginationParams {
	return PaginationParams{
		Page:    1,
		PerPage: 20,
	}
}

// --- Error Code Constants ---

// Common error codes
const (
	// Client errors (4xx)
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "NOT_FOUND"
	ErrCodeMethodNotAllowed    = "METHOD_NOT_ALLOWED"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeUnprocessableEntity = "UNPROCESSABLE_ENTITY"
	ErrCodeTooManyRequests     = "TOO_MANY_REQUESTS"

	// Server errors (5xx)
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"

	// Business logic errors
	ErrCodeValidationFailed  = "VALIDATION_FAILED"
	ErrCodeInsufficientStock = "INSUFFICIENT_STOCK"
	ErrCodeBookingExpired    = "BOOKING_EXPIRED"
	ErrCodePaymentFailed     = "PAYMENT_FAILED"
	ErrCodeDuplicateEntry    = "DUPLICATE_ENTRY"
	ErrCodeMaxLimitReached   = "MAX_LIMIT_REACHED"
	ErrCodeResourceLocked    = "RESOURCE_LOCKED"
)

// --- HTTP Status Code Mapping ---

// ErrorCodeToHTTPStatus maps error codes to HTTP status codes
var ErrorCodeToHTTPStatus = map[string]int{
	ErrCodeBadRequest:          http.StatusBadRequest,
	ErrCodeUnauthorized:        http.StatusUnauthorized,
	ErrCodeForbidden:           http.StatusForbidden,
	ErrCodeNotFound:            http.StatusNotFound,
	ErrCodeMethodNotAllowed:    http.StatusMethodNotAllowed,
	ErrCodeConflict:            http.StatusConflict,
	ErrCodeUnprocessableEntity: http.StatusUnprocessableEntity,
	ErrCodeTooManyRequests:     http.StatusTooManyRequests,
	ErrCodeInternalError:       http.StatusInternalServerError,
	ErrCodeServiceUnavailable:  http.StatusServiceUnavailable,
	ErrCodeValidationFailed:    http.StatusBadRequest,
	ErrCodeInsufficientStock:   http.StatusConflict,
	ErrCodeBookingExpired:      http.StatusGone,
	ErrCodePaymentFailed:       http.StatusPaymentRequired,
	ErrCodeDuplicateEntry:      http.StatusConflict,
	ErrCodeMaxLimitReached:     http.StatusTooManyRequests,
	ErrCodeResourceLocked:      http.StatusLocked,
}

// GetHTTPStatus returns the HTTP status code for an error code
func GetHTTPStatus(code string) int {
	if status, ok := ErrorCodeToHTTPStatus[code]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// --- Response Builders ---

// Success creates a success response with data
func Success(data interface{}) *Response {
	return &Response{
		Success: true,
		Data:    data,
	}
}

// SuccessWithMeta creates a success response with data and metadata
func SuccessWithMeta(data interface{}, meta *Meta) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
}

// Error creates an error response
func Error(code string, message string) *Response {
	return &Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}

// ErrorWithDetails creates an error response with additional details
func ErrorWithDetails(code string, message string, details map[string]string) *Response {
	return &Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// Paginated creates a paginated success response
func Paginated(data interface{}, page, perPage int, total int64) *Response {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return &Response{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// PaginatedFromParams creates a paginated response using PaginationParams
func PaginatedFromParams(data interface{}, params PaginationParams, total int64) *Response {
	return Paginated(data, params.Page, params.PerPage, total)
}

// --- Common Error Responses ---

// BadRequest creates a bad request error response
func BadRequest(message string) *Response {
	return Error(ErrCodeBadRequest, message)
}

// Unauthorized creates an unauthorized error response
func Unauthorized(message string) *Response {
	if message == "" {
		message = "Authentication required"
	}
	return Error(ErrCodeUnauthorized, message)
}

// Forbidden creates a forbidden error response
func Forbidden(message string) *Response {
	if message == "" {
		message = "Access denied"
	}
	return Error(ErrCodeForbidden, message)
}

// NotFound creates a not found error response
func NotFound(message string) *Response {
	if message == "" {
		message = "Resource not found"
	}
	return Error(ErrCodeNotFound, message)
}

// InternalError creates an internal server error response
func InternalError(message string) *Response {
	if message == "" {
		message = "An internal error occurred"
	}
	return Error(ErrCodeInternalError, message)
}

// ValidationFailed creates a validation error response with field details
func ValidationFailed(details map[string]string) *Response {
	return ErrorWithDetails(ErrCodeValidationFailed, "Validation failed", details)
}

// InsufficientStock creates an insufficient stock error response
func InsufficientStock(message string) *Response {
	if message == "" {
		message = "Insufficient stock available"
	}
	return Error(ErrCodeInsufficientStock, message)
}

// TooManyRequests creates a rate limit error response
func TooManyRequests(message string) *Response {
	if message == "" {
		message = "Too many requests, please try again later"
	}
	return Error(ErrCodeTooManyRequests, message)
}

// ServiceUnavailable creates a service unavailable error response
func ServiceUnavailable(message string) *Response {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	return Error(ErrCodeServiceUnavailable, message)
}
