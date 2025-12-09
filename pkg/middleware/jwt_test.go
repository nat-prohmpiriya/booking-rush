package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-jwt-middleware"

func init() {
	gin.SetMode(gin.TestMode)
}

func generateTestToken(claims jwt.MapClaims, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secret))
	return tokenString
}

func setupTestRouter(config *JWTConfig) *gin.Engine {
	router := gin.New()
	router.Use(JWTMiddleware(config))
	router.GET("/protected", func(c *gin.Context) {
		userID, _ := GetUserID(c)
		email, _ := GetEmail(c)
		role, _ := GetRole(c)
		tenantID, _ := GetTenantID(c)
		c.JSON(http.StatusOK, gin.H{
			"user_id":   userID,
			"email":     email,
			"role":      role,
			"tenant_id": tenantID,
		})
	})
	router.GET("/skip", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "skipped"})
	})
	return router
}

func TestJWTMiddleware(t *testing.T) {
	config := &JWTConfig{
		Secret:    testSecret,
		SkipPaths: []string{"/skip"},
	}

	t.Run("valid token", func(t *testing.T) {
		router := setupTestRouter(config)
		token := generateTestToken(jwt.MapClaims{
			"user_id":   "user-123",
			"email":     "test@example.com",
			"role":      "user",
			"tenant_id": "tenant-456",
			"exp":       time.Now().Add(time.Hour).Unix(),
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("missing authorization header", func(t *testing.T) {
		router := setupTestRouter(config)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("invalid authorization header format", func(t *testing.T) {
		router := setupTestRouter(config)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("empty token after Bearer", func(t *testing.T) {
		router := setupTestRouter(config)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer ")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		router := setupTestRouter(config)
		token := generateTestToken(jwt.MapClaims{
			"user_id": "user-123",
			"email":   "test@example.com",
			"role":    "user",
			"exp":     time.Now().Add(-time.Hour).Unix(), // Expired
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("invalid secret", func(t *testing.T) {
		router := setupTestRouter(config)
		token := generateTestToken(jwt.MapClaims{
			"user_id": "user-123",
			"email":   "test@example.com",
			"role":    "user",
			"exp":     time.Now().Add(time.Hour).Unix(),
		}, "wrong-secret")

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		router := setupTestRouter(config)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer not-a-valid-jwt-token")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("missing user_id in claims", func(t *testing.T) {
		router := setupTestRouter(config)
		token := generateTestToken(jwt.MapClaims{
			"email": "test@example.com",
			"role":  "user",
			"exp":   time.Now().Add(time.Hour).Unix(),
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("skip path", func(t *testing.T) {
		router := setupTestRouter(config)

		req := httptest.NewRequest(http.MethodGet, "/skip", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("claims extracted correctly", func(t *testing.T) {
		router := setupTestRouter(config)
		token := generateTestToken(jwt.MapClaims{
			"user_id":   "user-789",
			"email":     "claims@example.com",
			"role":      "admin",
			"tenant_id": "tenant-abc",
			"exp":       time.Now().Add(time.Hour).Unix(),
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		body := w.Body.String()
		if body == "" {
			t.Error("expected response body, got empty")
		}
		// Basic check that claims are in response
		if !contains(body, "user-789") {
			t.Errorf("expected user_id in response, got %s", body)
		}
		if !contains(body, "claims@example.com") {
			t.Errorf("expected email in response, got %s", body)
		}
		if !contains(body, "admin") {
			t.Errorf("expected role in response, got %s", body)
		}
		if !contains(body, "tenant-abc") {
			t.Errorf("expected tenant_id in response, got %s", body)
		}
	})
}

func TestRequireRole(t *testing.T) {
	config := &JWTConfig{Secret: testSecret}

	setupRouterWithRole := func(roles ...string) *gin.Engine {
		router := gin.New()
		router.Use(JWTMiddleware(config))
		router.GET("/admin", RequireRole(roles...), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin access"})
		})
		return router
	}

	t.Run("allowed role", func(t *testing.T) {
		router := setupRouterWithRole("admin", "superadmin")
		token := generateTestToken(jwt.MapClaims{
			"user_id": "user-123",
			"email":   "admin@example.com",
			"role":    "admin",
			"exp":     time.Now().Add(time.Hour).Unix(),
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("disallowed role", func(t *testing.T) {
		router := setupRouterWithRole("admin", "superadmin")
		token := generateTestToken(jwt.MapClaims{
			"user_id": "user-123",
			"email":   "user@example.com",
			"role":    "user",
			"exp":     time.Now().Add(time.Hour).Unix(),
		}, testSecret)

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
		}
	})

	t.Run("no authentication", func(t *testing.T) {
		router := gin.New()
		router.GET("/admin", RequireRole("admin"), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin access"})
		})

		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("GetUserID", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(ContextKeyUserID, "test-user-id")

		id, ok := GetUserID(c)
		if !ok {
			t.Error("expected ok to be true")
		}
		if id != "test-user-id" {
			t.Errorf("expected 'test-user-id', got '%s'", id)
		}
	})

	t.Run("GetUserID not set", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())

		_, ok := GetUserID(c)
		if ok {
			t.Error("expected ok to be false")
		}
	})

	t.Run("GetEmail", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(ContextKeyEmail, "test@example.com")

		email, ok := GetEmail(c)
		if !ok {
			t.Error("expected ok to be true")
		}
		if email != "test@example.com" {
			t.Errorf("expected 'test@example.com', got '%s'", email)
		}
	})

	t.Run("GetRole", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(ContextKeyRole, "admin")

		role, ok := GetRole(c)
		if !ok {
			t.Error("expected ok to be true")
		}
		if role != "admin" {
			t.Errorf("expected 'admin', got '%s'", role)
		}
	})

	t.Run("GetTenantID", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set(ContextKeyTenantID, "tenant-123")

		tenantID, ok := GetTenantID(c)
		if !ok {
			t.Error("expected ok to be true")
		}
		if tenantID != "tenant-123" {
			t.Errorf("expected 'tenant-123', got '%s'", tenantID)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
