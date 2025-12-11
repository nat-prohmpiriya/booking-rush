package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestDefaultActionMapper(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		expected AuditAction
	}{
		{"POST creates", "POST", "/api/v1/bookings", AuditActionCreate},
		{"PUT updates", "PUT", "/api/v1/bookings/123", AuditActionUpdate},
		{"PATCH updates", "PATCH", "/api/v1/users/456", AuditActionUpdate},
		{"DELETE deletes", "DELETE", "/api/v1/events/789", AuditActionDelete},
		{"GET views", "GET", "/api/v1/events", AuditActionView},
		{"login path", "POST", "/api/v1/auth/login", AuditActionLogin},
		{"logout path", "POST", "/api/v1/auth/logout", AuditActionLogout},
		{"reserve path", "POST", "/api/v1/seats/reserve", AuditActionReserve},
		{"confirm path", "POST", "/api/v1/bookings/confirm", AuditActionConfirm},
		{"cancel path", "POST", "/api/v1/bookings/cancel", AuditActionCancel},
		{"refund path", "POST", "/api/v1/payments/refund", AuditActionRefund},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultActionMapper(tt.method, tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultResourceExtractor(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedType string
		expectedID   string
	}{
		{"simple resource", "/api/v1/bookings/123e4567-e89b-12d3-a456-426614174000", "booking", "123e4567-e89b-12d3-a456-426614174000"},
		{"resource list", "/api/v1/events", "event", ""},
		{"nested resource", "/api/v1/events/123", "event", "123"},
		{"numeric ID", "/api/v1/users/12345", "user", "12345"},
		{"no api prefix", "/bookings/abc", "booking", ""},
		{"deep path", "/api/v1/events/123/tickets", "event", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceType, resourceID := defaultResourceExtractor(tt.path)
			assert.Equal(t, tt.expectedType, resourceType)
			assert.Equal(t, tt.expectedID, resourceID)
		})
	}
}

func TestMaskSensitiveFields(t *testing.T) {
	sensitiveFields := []string{"password", "token", "secret", "api_key"}

	tests := []struct {
		name     string
		input    map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "masks password",
			input: map[string]interface{}{
				"email":    "test@example.com",
				"password": "secret123",
			},
			expected: map[string]interface{}{
				"email":    "test@example.com",
				"password": "[REDACTED]",
			},
		},
		{
			name: "masks nested sensitive fields",
			input: map[string]interface{}{
				"user": map[string]interface{}{
					"name":    "John",
					"api_key": "key123",
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name":    "John",
					"api_key": "[REDACTED]",
				},
			},
		},
		{
			name:     "handles nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveFields(tt.input, sensitiveFields)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestComputeChanges(t *testing.T) {
	tests := []struct {
		name     string
		oldVals  map[string]interface{}
		newVals  map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name:    "detects value change",
			oldVals: map[string]interface{}{"name": "John", "age": 30},
			newVals: map[string]interface{}{"name": "Jane", "age": 30},
			expected: map[string]interface{}{
				"name": map[string]interface{}{"old": "John", "new": "Jane"},
			},
		},
		{
			name:    "detects new field",
			oldVals: map[string]interface{}{"name": "John"},
			newVals: map[string]interface{}{"name": "John", "email": "john@example.com"},
			expected: map[string]interface{}{
				"email": map[string]interface{}{"old": nil, "new": "john@example.com"},
			},
		},
		{
			name:    "detects deleted field",
			oldVals: map[string]interface{}{"name": "John", "phone": "123"},
			newVals: map[string]interface{}{"name": "John"},
			expected: map[string]interface{}{
				"phone": map[string]interface{}{"old": "123", "new": nil},
			},
		},
		{
			name:     "no changes",
			oldVals:  map[string]interface{}{"name": "John"},
			newVals:  map[string]interface{}{"name": "John"},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeChanges(tt.oldVals, tt.newVals)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "from X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name:       "from X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "192.168.1.2"},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.2",
		},
		{
			name:       "from RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.3:12345",
			expected:   "192.168.1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/", nil)
			c.Request.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				c.Request.Header.Set(k, v)
			}

			result := getClientIP(c)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid UUID", "123e4567-e89b-12d3-a456-426614174000", true},
		{"numeric ID", "12345", true},
		{"empty string", "", false},
		{"random string", "abc-def", false},
		{"partial UUID", "123e4567", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuditLogger_Log(t *testing.T) {
	config := &AuditConfig{
		DB:            nil,
		BufferSize:    10,
		FlushInterval: 100 * time.Millisecond,
		BatchSize:     100,
	}

	logger := NewAuditLogger(config)
	logger.SetTestMode(true)
	defer logger.Close()

	entry := &AuditEntry{
		ID:           "test-id",
		Action:       AuditActionCreate,
		ResourceType: "test",
		CreatedAt:    time.Now(),
	}

	// Should not block
	logger.Log(entry)

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	// Verify entry is collected
	entries := logger.GetTestEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "test-id", entries[0].ID)
}

func TestAuditLogger_BufferFull(t *testing.T) {
	config := &AuditConfig{
		DB:            nil,
		BufferSize:    2,
		FlushInterval: 1 * time.Hour,
		BatchSize:     100,
	}

	logger := NewAuditLogger(config)
	defer logger.Close()

	// Fill the buffer - should not panic or block
	for i := 0; i < 5; i++ {
		logger.Log(&AuditEntry{ID: "test"})
	}
}

func TestAuditMiddleware_SkipPaths(t *testing.T) {
	config := &AuditConfig{
		DB:            nil,
		BufferSize:    100,
		FlushInterval: 100 * time.Millisecond,
		BatchSize:     100,
		SkipPaths:     []string{"/health", "/metrics"},
		SkipMethods:   []string{"GET", "HEAD", "OPTIONS"},
	}

	logger := NewAuditLogger(config)
	logger.SetTestMode(true)
	defer logger.Close()

	router := gin.New()
	router.Use(AuditMiddleware(logger))
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.POST("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Test skipped path (GET /health)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test skipped method (GET /api/v1/test)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/test", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for potential flush
	time.Sleep(200 * time.Millisecond)

	// Should have no entries because both were skipped
	entries := logger.GetTestEntries()
	assert.Len(t, entries, 0, "No entries should be logged for skipped paths/methods")
}

func TestAuditMiddleware_CapturesUserInfo(t *testing.T) {
	config := &AuditConfig{
		DB:                nil,
		BufferSize:        100,
		FlushInterval:     100 * time.Millisecond,
		BatchSize:         100,
		SkipPaths:         []string{},
		SkipMethods:       []string{},
		ActionMapper:      defaultActionMapper,
		ResourceExtractor: defaultResourceExtractor,
	}

	logger := NewAuditLogger(config)
	logger.SetTestMode(true)
	defer logger.Close()

	router := gin.New()

	// Simulate JWT middleware
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyUserID, "user-123")
		c.Set(ContextKeyEmail, "test@example.com")
		c.Set(ContextKeyRole, "admin")
		c.Set(ContextKeyTenantID, "tenant-456")
		c.Next()
	})

	router.Use(AuditMiddleware(logger))
	router.POST("/api/v1/bookings", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/bookings", nil)
	req.Header.Set("X-Request-ID", "req-123")
	req.Header.Set("X-Trace-ID", "trace-456")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	entries := logger.GetTestEntries()
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, "user-123", *entry.UserID)
	assert.Equal(t, "test@example.com", entry.UserEmail)
	assert.Equal(t, "admin", entry.UserRole)
	assert.Equal(t, "tenant-456", *entry.TenantID)
	assert.Equal(t, AuditActionCreate, entry.Action)
	assert.Equal(t, "booking", entry.ResourceType)
	assert.Equal(t, "req-123", entry.RequestID)
	assert.Equal(t, "trace-456", entry.TraceID)
	assert.Equal(t, "TestAgent/1.0", entry.UserAgent)
}

func TestAuditMiddleware_SetContextValues(t *testing.T) {
	config := &AuditConfig{
		DB:                nil,
		BufferSize:        100,
		FlushInterval:     100 * time.Millisecond,
		BatchSize:         100,
		SkipPaths:         []string{},
		SkipMethods:       []string{},
		ActionMapper:      defaultActionMapper,
		ResourceExtractor: defaultResourceExtractor,
	}

	logger := NewAuditLogger(config)
	logger.SetTestMode(true)
	defer logger.Close()

	router := gin.New()
	router.Use(AuditMiddleware(logger))
	router.PUT("/api/v1/users/:id", func(c *gin.Context) {
		// Handler sets audit context
		SetAuditResourceType(c, "user")
		SetAuditResourceID(c, c.Param("id"))
		SetAuditOldValues(c, map[string]interface{}{"name": "Old Name"})
		SetAuditNewValues(c, map[string]interface{}{"name": "New Name"})
		SetAuditMetadata(c, map[string]interface{}{"source": "web"})
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/users/user-789", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	entries := logger.GetTestEntries()
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.Equal(t, "user", entry.ResourceType)
	assert.Equal(t, "user-789", *entry.ResourceID)
	assert.Equal(t, map[string]interface{}{"name": "Old Name"}, entry.OldValues)
	assert.Equal(t, map[string]interface{}{"name": "New Name"}, entry.NewValues)
	assert.Equal(t, map[string]interface{}{"source": "web"}, entry.Metadata)
	assert.NotNil(t, entry.Changes)
	assert.Contains(t, entry.Changes, "name")
}

func TestAuditMiddleware_SkipAudit(t *testing.T) {
	config := &AuditConfig{
		DB:                nil,
		BufferSize:        100,
		FlushInterval:     100 * time.Millisecond,
		BatchSize:         100,
		SkipPaths:         []string{},
		SkipMethods:       []string{},
		ActionMapper:      defaultActionMapper,
		ResourceExtractor: defaultResourceExtractor,
	}

	logger := NewAuditLogger(config)
	logger.SetTestMode(true)
	defer logger.Close()

	router := gin.New()
	router.Use(AuditMiddleware(logger))
	router.POST("/api/v1/internal", func(c *gin.Context) {
		SkipAudit(c)
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/internal", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for potential flush
	time.Sleep(200 * time.Millisecond)

	// Should be empty because SkipAudit was called
	entries := logger.GetTestEntries()
	assert.Len(t, entries, 0, "No entries should be logged when SkipAudit is called")
}

func TestAuditMiddleware_CapturesRequestBody(t *testing.T) {
	config := &AuditConfig{
		DB:                nil,
		BufferSize:        100,
		FlushInterval:     100 * time.Millisecond,
		BatchSize:         100,
		SkipPaths:         []string{},
		SkipMethods:       []string{},
		ActionMapper:      defaultActionMapper,
		ResourceExtractor: defaultResourceExtractor,
		EnableRequestBody: true,
		MaxBodySize:       10 * 1024,
		SensitiveFields:   []string{"password"},
	}

	logger := NewAuditLogger(config)
	logger.SetTestMode(true)
	defer logger.Close()

	router := gin.New()
	router.Use(AuditMiddleware(logger))
	router.POST("/api/v1/users", func(c *gin.Context) {
		// Verify body is still readable
		var body map[string]interface{}
		err := c.BindJSON(&body)
		require.NoError(t, err)
		assert.Equal(t, "test@example.com", body["email"])
		c.String(http.StatusOK, "OK")
	})

	requestBody := map[string]interface{}{
		"email":    "test@example.com",
		"password": "secret123",
		"name":     "Test User",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/users", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	entries := logger.GetTestEntries()
	require.Len(t, entries, 1)

	entry := entries[0]
	assert.NotNil(t, entry.NewValues)
	assert.Equal(t, "test@example.com", entry.NewValues["email"])
	assert.Equal(t, "[REDACTED]", entry.NewValues["password"])
	assert.Equal(t, "Test User", entry.NewValues["name"])
}

func TestAuditLogger_Close(t *testing.T) {
	config := &AuditConfig{
		DB:            nil,
		BufferSize:    100,
		FlushInterval: 100 * time.Millisecond, // Short interval to allow flush before close
		BatchSize:     100,
	}

	logger := NewAuditLogger(config)
	logger.SetTestMode(true)

	// Add some entries
	for i := 0; i < 5; i++ {
		logger.Log(&AuditEntry{ID: "test"})
	}

	// Wait for flush to happen before close
	time.Sleep(200 * time.Millisecond)

	// Close should not panic and should be idempotent
	err := logger.Close()
	assert.NoError(t, err)

	err = logger.Close()
	assert.NoError(t, err)

	// Check that entries were flushed
	entries := logger.GetTestEntries()
	assert.Len(t, entries, 5)
}

func TestDefaultAuditConfig(t *testing.T) {
	config := DefaultAuditConfig(nil)

	assert.Nil(t, config.DB)
	assert.Equal(t, 1000, config.BufferSize)
	assert.Equal(t, 5*time.Second, config.FlushInterval)
	assert.Equal(t, 100, config.BatchSize)
	assert.Contains(t, config.SkipPaths, "/health")
	assert.Contains(t, config.SkipMethods, "GET")
	assert.NotNil(t, config.ActionMapper)
	assert.NotNil(t, config.ResourceExtractor)
	assert.False(t, config.EnableRequestBody)
	assert.False(t, config.EnableResponseBody)
	assert.Equal(t, 10*1024, config.MaxBodySize)
	assert.Contains(t, config.SensitiveFields, "password")
}

func TestAuditEntry_Fields(t *testing.T) {
	userID := "user-123"
	tenantID := "tenant-456"
	resourceID := "resource-789"

	entry := &AuditEntry{
		ID:           "entry-id",
		TenantID:     &tenantID,
		UserID:       &userID,
		UserEmail:    "test@example.com",
		UserRole:     "admin",
		Action:       AuditActionCreate,
		ResourceType: "booking",
		ResourceID:   &resourceID,
		IPAddress:    "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
		RequestID:    "req-123",
		TraceID:      "trace-456",
		OldValues:    map[string]interface{}{"field": "old"},
		NewValues:    map[string]interface{}{"field": "new"},
		Changes:      map[string]interface{}{"field": map[string]interface{}{"old": "old", "new": "new"}},
		Metadata:     map[string]interface{}{"extra": "data"},
		CreatedAt:    time.Now(),
	}

	// Test JSON marshaling
	data, err := json.Marshal(entry)
	assert.NoError(t, err)

	var decoded AuditEntry
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, entry.ID, decoded.ID)
	assert.Equal(t, *entry.UserID, *decoded.UserID)
	assert.Equal(t, entry.Action, decoded.Action)
}

func TestAuditMiddleware_IPExtraction(t *testing.T) {
	config := &AuditConfig{
		DB:                nil,
		BufferSize:        100,
		FlushInterval:     100 * time.Millisecond,
		BatchSize:         100,
		SkipPaths:         []string{},
		SkipMethods:       []string{},
		ActionMapper:      defaultActionMapper,
		ResourceExtractor: defaultResourceExtractor,
	}

	logger := NewAuditLogger(config)
	logger.SetTestMode(true)
	defer logger.Close()

	router := gin.New()
	router.Use(AuditMiddleware(logger))
	router.POST("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1")
	req.RemoteAddr = "127.0.0.1:12345"
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	entries := logger.GetTestEntries()
	require.Len(t, entries, 1)

	// Should use X-Forwarded-For (first IP)
	assert.Equal(t, "192.168.1.100", entries[0].IPAddress)
}

func TestAuditLogger_BatchFlush(t *testing.T) {
	config := &AuditConfig{
		DB:            nil,
		BufferSize:    100,
		FlushInterval: 1 * time.Hour, // Long interval
		BatchSize:     5,             // Small batch size to trigger batch flush
	}

	logger := NewAuditLogger(config)
	logger.SetTestMode(true)
	defer logger.Close()

	// Add batch size entries
	for i := 0; i < 5; i++ {
		logger.Log(&AuditEntry{ID: "test"})
	}

	// Wait a bit for batch processing
	time.Sleep(100 * time.Millisecond)

	// Should have flushed
	entries := logger.GetTestEntries()
	assert.Len(t, entries, 5)
}

func TestAuditActions(t *testing.T) {
	// Test all action constants
	actions := []AuditAction{
		AuditActionCreate,
		AuditActionUpdate,
		AuditActionDelete,
		AuditActionLogin,
		AuditActionLogout,
		AuditActionReserve,
		AuditActionConfirm,
		AuditActionCancel,
		AuditActionRefund,
		AuditActionView,
	}

	for _, action := range actions {
		assert.NotEmpty(t, string(action))
	}
}

func TestAuditContextKeys(t *testing.T) {
	// Test all context key constants
	keys := []string{
		ContextKeyAuditResourceType,
		ContextKeyAuditResourceID,
		ContextKeyAuditOldValues,
		ContextKeyAuditNewValues,
		ContextKeyAuditMetadata,
	}

	for _, key := range keys {
		assert.NotEmpty(t, key)
	}
}
