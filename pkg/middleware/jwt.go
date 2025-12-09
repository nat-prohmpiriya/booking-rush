package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/response"
)

var (
	ErrMissingAuthHeader = errors.New("missing authorization header")
	ErrInvalidAuthFormat = errors.New("invalid authorization header format")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
)

// Context keys for user information
const (
	ContextKeyUserID   = "user_id"
	ContextKeyEmail    = "email"
	ContextKeyRole     = "role"
	ContextKeyTenantID = "tenant_id"
)

// JWTConfig holds configuration for JWT middleware
type JWTConfig struct {
	// Secret key for validating JWT tokens
	Secret string
	// SkipPaths is a list of paths that should skip JWT validation
	SkipPaths []string
}

// JWTMiddleware creates a new JWT validation middleware
func JWTMiddleware(config *JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if path should skip JWT validation
		for _, path := range config.SkipPaths {
			if c.Request.URL.Path == path {
				c.Next()
				return
			}
		}

		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Error("MISSING_TOKEN", "Authorization header is required"))
			return
		}

		// Extract token from "Bearer <token>"
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid authorization header format"))
			return
		}
		tokenString := authHeader[len(bearerPrefix):]

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Token is empty"))
			return
		}

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidToken
			}
			return []byte(config.Secret), nil
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, response.Error("TOKEN_EXPIRED", "Access token has expired"))
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid access token"))
			return
		}

		if !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid access token"))
			return
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Invalid token claims"))
			return
		}

		// Extract and validate required claims
		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Error("INVALID_TOKEN", "Missing user_id in token"))
			return
		}

		email, _ := claims["email"].(string)
		role, _ := claims["role"].(string)
		tenantID, _ := claims["tenant_id"].(string)

		// Inject user context into request
		c.Set(ContextKeyUserID, userID)
		c.Set(ContextKeyEmail, email)
		c.Set(ContextKeyRole, role)
		c.Set(ContextKeyTenantID, tenantID)

		c.Next()
	}
}

// RequireRole creates a middleware that checks if user has required role
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(ContextKeyRole)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Error("UNAUTHORIZED", "User not authenticated"))
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.InternalError("Invalid role type"))
			return
		}

		// Check if user has one of the required roles
		for _, r := range roles {
			if roleStr == r {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, response.Error("FORBIDDEN", "Insufficient permissions"))
	}
}

// GetUserID extracts user ID from gin context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// GetEmail extracts email from gin context
func GetEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get(ContextKeyEmail)
	if !exists {
		return "", false
	}
	e, ok := email.(string)
	return e, ok
}

// GetRole extracts role from gin context
func GetRole(c *gin.Context) (string, bool) {
	role, exists := c.Get(ContextKeyRole)
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}

// GetTenantID extracts tenant ID from gin context
func GetTenantID(c *gin.Context) (string, bool) {
	tenantID, exists := c.Get(ContextKeyTenantID)
	if !exists {
		return "", false
	}
	t, ok := tenantID.(string)
	return t, ok
}
