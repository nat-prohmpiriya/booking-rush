package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewReverseProxy(t *testing.T) {
	config := DefaultConfig()
	rp := NewReverseProxy(config)

	if rp == nil {
		t.Fatal("Expected non-nil ReverseProxy")
	}

	if rp.config.DefaultTimeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", rp.config.DefaultTimeout)
	}

	if len(rp.proxies) == 0 {
		t.Error("Expected proxies to be initialized")
	}
}

func TestFindRoute(t *testing.T) {
	config := ProxyConfig{
		Routes: []RouteConfig{
			{
				PathPrefix:  "/api/v1/auth",
				RequireAuth: false,
				Service: ServiceConfig{
					Name:    "auth-service",
					BaseURL: "http://localhost:8081",
				},
			},
			{
				PathPrefix:     "/api/v1/events",
				RequireAuth:    false,
				AllowedMethods: []string{"GET"},
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: "http://localhost:8082",
				},
			},
			{
				PathPrefix:     "/api/v1/events",
				RequireAuth:    true,
				AllowedMethods: []string{"POST", "PUT", "DELETE"},
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: "http://localhost:8082",
				},
			},
			{
				PathPrefix:  "/api/v1/bookings",
				RequireAuth: true,
				Service: ServiceConfig{
					Name:    "booking-service",
					BaseURL: "http://localhost:8083",
				},
			},
		},
	}

	rp := NewReverseProxy(config)

	tests := []struct {
		name         string
		path         string
		method       string
		expectMatch  bool
		expectAuth   bool
		expectPrefix string
	}{
		{
			name:         "auth login - public",
			path:         "/api/v1/auth/login",
			method:       "POST",
			expectMatch:  true,
			expectAuth:   false,
			expectPrefix: "/api/v1/auth",
		},
		{
			name:         "events GET - public",
			path:         "/api/v1/events",
			method:       "GET",
			expectMatch:  true,
			expectAuth:   false,
			expectPrefix: "/api/v1/events",
		},
		{
			name:         "events POST - protected",
			path:         "/api/v1/events",
			method:       "POST",
			expectMatch:  true,
			expectAuth:   true,
			expectPrefix: "/api/v1/events",
		},
		{
			name:         "bookings - protected",
			path:         "/api/v1/bookings/123",
			method:       "GET",
			expectMatch:  true,
			expectAuth:   true,
			expectPrefix: "/api/v1/bookings",
		},
		{
			name:        "unknown path - no match",
			path:        "/api/v1/unknown",
			method:      "GET",
			expectMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := rp.findRoute(tt.path, tt.method)

			if tt.expectMatch {
				if route == nil {
					t.Fatal("Expected route to be found")
				}
				if route.RequireAuth != tt.expectAuth {
					t.Errorf("Expected RequireAuth=%v, got %v", tt.expectAuth, route.RequireAuth)
				}
				if route.PathPrefix != tt.expectPrefix {
					t.Errorf("Expected PathPrefix=%s, got %s", tt.expectPrefix, route.PathPrefix)
				}
			} else {
				if route != nil {
					t.Error("Expected no route match")
				}
			}
		})
	}
}

func TestReverseProxy_Handler_NotFound(t *testing.T) {
	config := ProxyConfig{
		Routes: []RouteConfig{
			{
				PathPrefix: "/api/v1/auth",
				Service: ServiceConfig{
					Name:    "auth-service",
					BaseURL: "http://localhost:8081",
				},
			},
		},
	}

	rp := NewReverseProxy(config)
	handler := rp.Handler()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.GET("/api/v1/unknown/*path", handler)

	req := httptest.NewRequest("GET", "/api/v1/unknown/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["success"] != false {
		t.Error("Expected success=false in response")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DefaultTimeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", config.DefaultTimeout)
	}

	if len(config.Routes) == 0 {
		t.Error("Expected routes to be configured")
	}

	// Check that auth routes exist
	hasAuth := false
	for _, route := range config.Routes {
		if route.PathPrefix == "/api/v1/auth" {
			hasAuth = true
			if route.RequireAuth {
				t.Error("Auth routes should be public")
			}
		}
	}

	if !hasAuth {
		t.Error("Expected auth routes to be configured")
	}
}

func TestConfigFromEnv(t *testing.T) {
	config := ConfigFromEnv(
		"http://auth:8081",
		"http://ticket:8082",
		"http://booking:8083",
		"http://payment:8084",
		"test-secret",
	)

	if config.JWTSecret != "test-secret" {
		t.Errorf("Expected JWT secret 'test-secret', got '%s'", config.JWTSecret)
	}

	// Check service URLs
	authFound := false
	bookingFound := false
	for _, route := range config.Routes {
		if route.Service.Name == "auth-service" && route.Service.BaseURL == "http://auth:8081" {
			authFound = true
		}
		if route.Service.Name == "booking-service" && route.Service.BaseURL == "http://booking:8083" {
			bookingFound = true
		}
	}

	if !authFound {
		t.Error("Expected auth-service with custom URL")
	}
	if !bookingFound {
		t.Error("Expected booking-service with custom URL")
	}
}

func TestGetRequireAuthRoutes(t *testing.T) {
	config := ProxyConfig{
		Routes: []RouteConfig{
			{PathPrefix: "/api/v1/auth", RequireAuth: false},
			{PathPrefix: "/api/v1/events", RequireAuth: false, AllowedMethods: []string{"GET"}},
			{PathPrefix: "/api/v1/events", RequireAuth: true, AllowedMethods: []string{"POST"}},
			{PathPrefix: "/api/v1/bookings", RequireAuth: true},
		},
	}

	rp := NewReverseProxy(config)
	authRoutes := rp.GetRequireAuthRoutes()

	if len(authRoutes) != 2 {
		t.Errorf("Expected 2 auth routes, got %d", len(authRoutes))
	}

	for _, route := range authRoutes {
		if !route.RequireAuth {
			t.Error("Expected all returned routes to require auth")
		}
	}
}

func TestGetPublicRoutes(t *testing.T) {
	config := ProxyConfig{
		Routes: []RouteConfig{
			{PathPrefix: "/api/v1/auth", RequireAuth: false},
			{PathPrefix: "/api/v1/events", RequireAuth: false, AllowedMethods: []string{"GET"}},
			{PathPrefix: "/api/v1/events", RequireAuth: true, AllowedMethods: []string{"POST"}},
			{PathPrefix: "/api/v1/bookings", RequireAuth: true},
		},
	}

	rp := NewReverseProxy(config)
	publicRoutes := rp.GetPublicRoutes()

	if len(publicRoutes) != 2 {
		t.Errorf("Expected 2 public routes, got %d", len(publicRoutes))
	}

	for _, route := range publicRoutes {
		if route.RequireAuth {
			t.Error("Expected all returned routes to be public")
		}
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"deadline exceeded", context.DeadlineExceeded, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeoutError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isConnectionError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestReverseProxyWithBackend tests the proxy with a mock backend
func TestReverseProxyWithBackend(t *testing.T) {
	// Create mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return request info
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := map[string]interface{}{
			"path":       r.URL.Path,
			"method":     r.Method,
			"user_id":    r.Header.Get("X-User-ID"),
			"user_email": r.Header.Get("X-User-Email"),
			"request_id": r.Header.Get("X-Request-ID"),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer backend.Close()

	config := ProxyConfig{
		DefaultTimeout: 5 * time.Second,
		Routes: []RouteConfig{
			{
				PathPrefix:  "/api/v1/test",
				RequireAuth: false,
				Service: ServiceConfig{
					Name:    "test-service",
					BaseURL: backend.URL,
					Timeout: 5 * time.Second,
				},
			},
		},
	}

	rp := NewReverseProxy(config)
	handler := rp.Handler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/api/v1/test/hello", nil)
	req.Header.Set("X-Request-ID", "test-request-123")
	c.Request = req

	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["path"] != "/api/v1/test/hello" {
		t.Errorf("Expected path '/api/v1/test/hello', got '%s'", resp["path"])
	}

	if resp["method"] != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", resp["method"])
	}
}

// TestReverseProxyUserContextHeaders tests that user context is passed to backend
func TestReverseProxyUserContextHeaders(t *testing.T) {
	var receivedHeaders http.Header

	// Create mock backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	config := ProxyConfig{
		Routes: []RouteConfig{
			{
				PathPrefix: "/api/v1/test",
				Service: ServiceConfig{
					Name:    "test-service",
					BaseURL: backend.URL,
				},
			},
		},
	}

	rp := NewReverseProxy(config)
	handler := rp.Handler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	c.Request = req

	// Set user context (simulating JWT middleware)
	c.Set("user_id", "user-123")
	c.Set("email", "test@example.com")
	c.Set("role", "admin")
	c.Set("tenant_id", "tenant-456")

	handler(c)

	// Verify headers were passed
	if receivedHeaders.Get("X-User-ID") != "user-123" {
		t.Errorf("Expected X-User-ID header 'user-123', got '%s'", receivedHeaders.Get("X-User-ID"))
	}
	if receivedHeaders.Get("X-User-Email") != "test@example.com" {
		t.Errorf("Expected X-User-Email header 'test@example.com', got '%s'", receivedHeaders.Get("X-User-Email"))
	}
	if receivedHeaders.Get("X-User-Role") != "admin" {
		t.Errorf("Expected X-User-Role header 'admin', got '%s'", receivedHeaders.Get("X-User-Role"))
	}
	if receivedHeaders.Get("X-Tenant-ID") != "tenant-456" {
		t.Errorf("Expected X-Tenant-ID header 'tenant-456', got '%s'", receivedHeaders.Get("X-Tenant-ID"))
	}
}

// TestReverseProxyStripPrefix tests path prefix stripping
func TestReverseProxyStripPrefix(t *testing.T) {
	var receivedPath string

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	config := ProxyConfig{
		Routes: []RouteConfig{
			{
				PathPrefix:  "/api/v1/test",
				StripPrefix: "/api/v1",
				Service: ServiceConfig{
					Name:    "test-service",
					BaseURL: backend.URL,
				},
			},
		},
	}

	rp := NewReverseProxy(config)
	handler := rp.Handler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/api/v1/test/hello", nil)
	c.Request = req

	handler(c)

	if receivedPath != "/test/hello" {
		t.Errorf("Expected stripped path '/test/hello', got '%s'", receivedPath)
	}
}
