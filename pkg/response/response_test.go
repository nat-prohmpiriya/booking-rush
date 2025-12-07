package response

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestSuccess(t *testing.T) {
	data := map[string]string{"name": "test"}
	resp := Success(data)

	if !resp.Success {
		t.Error("Expected success to be true")
	}
	if resp.Data == nil {
		t.Error("Expected data to be set")
	}
	if resp.Error != nil {
		t.Error("Expected error to be nil")
	}
	if resp.Meta != nil {
		t.Error("Expected meta to be nil")
	}
}

func TestSuccess_JSONFormat(t *testing.T) {
	data := map[string]string{"id": "123"}
	resp := Success(data)

	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if parsed["success"] != true {
		t.Errorf("Expected success=true, got %v", parsed["success"])
	}
	if _, ok := parsed["error"]; ok {
		t.Error("Expected error field to be omitted")
	}
	if _, ok := parsed["meta"]; ok {
		t.Error("Expected meta field to be omitted")
	}
}

func TestError(t *testing.T) {
	resp := Error(ErrCodeNotFound, "User not found")

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Data != nil {
		t.Error("Expected data to be nil")
	}
	if resp.Error == nil {
		t.Fatal("Expected error to be set")
	}
	if resp.Error.Code != ErrCodeNotFound {
		t.Errorf("Expected code %s, got %s", ErrCodeNotFound, resp.Error.Code)
	}
	if resp.Error.Message != "User not found" {
		t.Errorf("Expected message 'User not found', got '%s'", resp.Error.Message)
	}
}

func TestError_JSONFormat(t *testing.T) {
	resp := Error(ErrCodeBadRequest, "Invalid input")

	jsonBytes, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if parsed["success"] != false {
		t.Errorf("Expected success=false, got %v", parsed["success"])
	}

	errorObj, ok := parsed["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error object")
	}
	if errorObj["code"] != ErrCodeBadRequest {
		t.Errorf("Expected code %s, got %v", ErrCodeBadRequest, errorObj["code"])
	}
	if errorObj["message"] != "Invalid input" {
		t.Errorf("Expected message 'Invalid input', got %v", errorObj["message"])
	}
}

func TestErrorWithDetails(t *testing.T) {
	details := map[string]string{
		"email":    "invalid format",
		"password": "too short",
	}
	resp := ErrorWithDetails(ErrCodeValidationFailed, "Validation failed", details)

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error == nil {
		t.Fatal("Expected error to be set")
	}
	if resp.Error.Details == nil {
		t.Fatal("Expected details to be set")
	}
	if resp.Error.Details["email"] != "invalid format" {
		t.Errorf("Expected email error, got %v", resp.Error.Details["email"])
	}
}

func TestPaginated(t *testing.T) {
	data := []string{"item1", "item2"}
	resp := Paginated(data, 1, 10, 25)

	if !resp.Success {
		t.Error("Expected success to be true")
	}
	if resp.Data == nil {
		t.Error("Expected data to be set")
	}
	if resp.Meta == nil {
		t.Fatal("Expected meta to be set")
	}
	if resp.Meta.Page != 1 {
		t.Errorf("Expected page 1, got %d", resp.Meta.Page)
	}
	if resp.Meta.PerPage != 10 {
		t.Errorf("Expected per_page 10, got %d", resp.Meta.PerPage)
	}
	if resp.Meta.Total != 25 {
		t.Errorf("Expected total 25, got %d", resp.Meta.Total)
	}
	if resp.Meta.TotalPages != 3 {
		t.Errorf("Expected total_pages 3, got %d", resp.Meta.TotalPages)
	}
}

func TestPaginated_TotalPagesCalculation(t *testing.T) {
	tests := []struct {
		name          string
		total         int64
		perPage       int
		expectedPages int
	}{
		{"exact division", 20, 10, 2},
		{"with remainder", 25, 10, 3},
		{"less than page", 5, 10, 1},
		{"zero items", 0, 10, 0},
		{"single item", 1, 10, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := Paginated(nil, 1, tt.perPage, tt.total)
			if resp.Meta.TotalPages != tt.expectedPages {
				t.Errorf("Expected %d pages, got %d", tt.expectedPages, resp.Meta.TotalPages)
			}
		})
	}
}

func TestPaginatedFromParams(t *testing.T) {
	params := PaginationParams{Page: 2, PerPage: 15}
	resp := PaginatedFromParams(nil, params, 100)

	if resp.Meta.Page != 2 {
		t.Errorf("Expected page 2, got %d", resp.Meta.Page)
	}
	if resp.Meta.PerPage != 15 {
		t.Errorf("Expected per_page 15, got %d", resp.Meta.PerPage)
	}
}

func TestDefaultPagination(t *testing.T) {
	params := DefaultPagination()

	if params.Page != 1 {
		t.Errorf("Expected default page 1, got %d", params.Page)
	}
	if params.PerPage != 20 {
		t.Errorf("Expected default per_page 20, got %d", params.PerPage)
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{ErrCodeBadRequest, http.StatusBadRequest},
		{ErrCodeUnauthorized, http.StatusUnauthorized},
		{ErrCodeForbidden, http.StatusForbidden},
		{ErrCodeNotFound, http.StatusNotFound},
		{ErrCodeTooManyRequests, http.StatusTooManyRequests},
		{ErrCodeInternalError, http.StatusInternalServerError},
		{ErrCodeInsufficientStock, http.StatusConflict},
		{"UNKNOWN_CODE", http.StatusInternalServerError}, // default
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			status := GetHTTPStatus(tt.code)
			if status != tt.expected {
				t.Errorf("Expected status %d, got %d", tt.expected, status)
			}
		})
	}
}

func TestCommonErrorResponses(t *testing.T) {
	tests := []struct {
		name    string
		fn      func(string) *Response
		message string
		code    string
	}{
		{"BadRequest", BadRequest, "bad input", ErrCodeBadRequest},
		{"Unauthorized", Unauthorized, "", ErrCodeUnauthorized},
		{"Forbidden", Forbidden, "", ErrCodeForbidden},
		{"NotFound", NotFound, "", ErrCodeNotFound},
		{"InternalError", InternalError, "", ErrCodeInternalError},
		{"InsufficientStock", InsufficientStock, "", ErrCodeInsufficientStock},
		{"TooManyRequests", TooManyRequests, "", ErrCodeTooManyRequests},
		{"ServiceUnavailable", ServiceUnavailable, "", ErrCodeServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := tt.fn(tt.message)
			if resp.Success {
				t.Error("Expected success to be false")
			}
			if resp.Error == nil {
				t.Fatal("Expected error to be set")
			}
			if resp.Error.Code != tt.code {
				t.Errorf("Expected code %s, got %s", tt.code, resp.Error.Code)
			}
			if resp.Error.Message == "" {
				t.Error("Expected message to be set (with default)")
			}
		})
	}
}

func TestValidationFailed(t *testing.T) {
	details := map[string]string{
		"field1": "error1",
		"field2": "error2",
	}
	resp := ValidationFailed(details)

	if resp.Success {
		t.Error("Expected success to be false")
	}
	if resp.Error.Code != ErrCodeValidationFailed {
		t.Errorf("Expected code %s, got %s", ErrCodeValidationFailed, resp.Error.Code)
	}
	if len(resp.Error.Details) != 2 {
		t.Errorf("Expected 2 details, got %d", len(resp.Error.Details))
	}
}
