package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func TestNewRouter(t *testing.T) {
	config := DefaultConfig()
	rp := NewReverseProxy(config)
	router := NewRouter(rp, "test-secret")

	if router == nil {
		t.Fatal("Expected non-nil Router")
	}

	if router.jwtConfig.Secret != "test-secret" {
		t.Errorf("Expected JWT secret 'test-secret', got '%s'", router.jwtConfig.Secret)
	}
}

func TestRouter_MatchHandler_PublicRoute(t *testing.T) {
	// Create mock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer backend.Close()

	config := ProxyConfig{
		Routes: []RouteConfig{
			{
				PathPrefix:  "/api/v1/auth",
				RequireAuth: false,
				Service: ServiceConfig{
					Name:    "auth-service",
					BaseURL: backend.URL,
				},
			},
		},
	}

	rp := NewReverseProxy(config)
	router := NewRouter(rp, "test-secret")
	handler := router.MatchHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	c.Request = req

	handler(c)

	// Should succeed without token (public route)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for public route, got %d", w.Code)
	}
}

func TestRouter_MatchHandler_ProtectedRoute_NoToken(t *testing.T) {
	config := ProxyConfig{
		Routes: []RouteConfig{
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
	router := NewRouter(rp, "test-secret")
	handler := router.MatchHandler()

	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)

	engine.NoRoute(handler)

	req := httptest.NewRequest("GET", "/api/v1/bookings", nil)
	engine.ServeHTTP(w, req)

	// Should fail without token
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for protected route without token, got %d", w.Code)
	}
}

func TestRouter_MatchHandler_ProtectedRoute_WithToken(t *testing.T) {
	// Create mock backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_id":    r.Header.Get("X-User-ID"),
			"user_email": r.Header.Get("X-User-Email"),
		})
	}))
	defer backend.Close()

	jwtSecret := "test-secret-key"

	config := ProxyConfig{
		Routes: []RouteConfig{
			{
				PathPrefix:  "/api/v1/bookings",
				RequireAuth: true,
				Service: ServiceConfig{
					Name:    "booking-service",
					BaseURL: backend.URL,
				},
			},
		},
	}

	rp := NewReverseProxy(config)
	router := NewRouter(rp, jwtSecret)
	handler := router.MatchHandler()

	// Create valid JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user-123",
		"email":   "test@example.com",
		"role":    "user",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(jwtSecret))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/api/v1/bookings", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	c.Request = req

	handler(c)

	// Should succeed with valid token
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for protected route with valid token, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["user_id"] != "user-123" {
		t.Errorf("Expected user_id 'user-123', got '%s'", resp["user_id"])
	}
}

func TestRouter_MatchHandler_NotFound(t *testing.T) {
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
	router := NewRouter(rp, "test-secret")
	handler := router.MatchHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/api/v1/unknown", nil)
	c.Request = req

	handler(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestRouter_MatchHandler_ExpiredToken(t *testing.T) {
	jwtSecret := "test-secret-key"

	config := ProxyConfig{
		Routes: []RouteConfig{
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
	router := NewRouter(rp, jwtSecret)
	handler := router.MatchHandler()

	// Create expired JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user-123",
		"email":   "test@example.com",
		"exp":     time.Now().Add(-time.Hour).Unix(), // Expired
	})
	tokenString, _ := token.SignedString([]byte(jwtSecret))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/api/v1/bookings", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	c.Request = req

	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for expired token, got %d", w.Code)
	}
}

func TestRouter_MatchHandler_InvalidToken(t *testing.T) {
	jwtSecret := "test-secret-key"

	config := ProxyConfig{
		Routes: []RouteConfig{
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
	router := NewRouter(rp, jwtSecret)
	handler := router.MatchHandler()

	// Create token with wrong secret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "user-123",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString([]byte("wrong-secret"))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/api/v1/bookings", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	c.Request = req

	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid token, got %d", w.Code)
	}
}

func TestRouter_MatchHandler_MalformedToken(t *testing.T) {
	jwtSecret := "test-secret-key"

	config := ProxyConfig{
		Routes: []RouteConfig{
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
	router := NewRouter(rp, jwtSecret)
	handler := router.MatchHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/api/v1/bookings", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	c.Request = req

	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for malformed token, got %d", w.Code)
	}
}

func TestRouter_MatchHandler_MissingBearer(t *testing.T) {
	jwtSecret := "test-secret-key"

	config := ProxyConfig{
		Routes: []RouteConfig{
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
	router := NewRouter(rp, jwtSecret)
	handler := router.MatchHandler()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/api/v1/bookings", nil)
	req.Header.Set("Authorization", "some-token") // Missing "Bearer " prefix
	c.Request = req

	handler(c)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing Bearer prefix, got %d", w.Code)
	}
}

func TestRouteByMethod(t *testing.T) {
	// Test that same path can have different auth requirements per method
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	config := ProxyConfig{
		Routes: []RouteConfig{
			// GET /api/v1/events is public
			{
				PathPrefix:     "/api/v1/events",
				RequireAuth:    false,
				AllowedMethods: []string{"GET"},
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: backend.URL,
				},
			},
			// POST /api/v1/events requires auth
			{
				PathPrefix:     "/api/v1/events",
				RequireAuth:    true,
				AllowedMethods: []string{"POST"},
				Service: ServiceConfig{
					Name:    "ticket-service",
					BaseURL: backend.URL,
				},
			},
		},
	}

	rp := NewReverseProxy(config)
	router := NewRouter(rp, "test-secret")
	handler := router.MatchHandler()

	// Test GET - should succeed without token
	t.Run("GET without token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("GET", "/api/v1/events", nil)
		c.Request = req
		handler(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected GET to succeed without token, got status %d", w.Code)
		}
	})

	// Test POST - should fail without token
	t.Run("POST without token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := httptest.NewRequest("POST", "/api/v1/events", nil)
		c.Request = req
		handler(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected POST to fail without token, got status %d", w.Code)
		}
	})
}
