package proxy

import (
	"strings"

	"github.com/gin-gonic/gin"
	pkgmiddleware "github.com/prohmpiriya/booking-rush-10k-rps/pkg/middleware"
)

// Router manages API Gateway routing with JWT middleware
type Router struct {
	proxy     *ReverseProxy
	jwtConfig *pkgmiddleware.JWTConfig
}

// NewRouter creates a new router with proxy and JWT configuration
func NewRouter(proxy *ReverseProxy, jwtSecret string) *Router {
	return &Router{
		proxy: proxy,
		jwtConfig: &pkgmiddleware.JWTConfig{
			Secret:    jwtSecret,
			SkipPaths: []string{"/health", "/ready", "/api/v1/status"},
		},
	}
}

// SetupRoutes configures all routes on the given router group
func (r *Router) SetupRoutes(router *gin.Engine) {
	// Create handlers for public and protected routes
	publicHandler := r.createPublicHandler()
	protectedHandler := r.createProtectedHandler()

	// Setup routes based on configuration
	for _, route := range r.proxy.config.Routes {
		if route.RequireAuth {
			r.setupProtectedRoute(router, route, protectedHandler)
		} else {
			r.setupPublicRoute(router, route, publicHandler)
		}
	}
}

// createPublicHandler creates a handler for public routes (no JWT)
func (r *Router) createPublicHandler() gin.HandlerFunc {
	return r.proxy.Handler()
}

// createProtectedHandler creates a handler for protected routes (with JWT)
func (r *Router) createProtectedHandler() gin.HandlerFunc {
	jwtMiddleware := pkgmiddleware.JWTMiddleware(r.jwtConfig)
	proxyHandler := r.proxy.Handler()

	return func(c *gin.Context) {
		// First apply JWT middleware
		jwtMiddleware(c)

		// If JWT validation failed, context is aborted
		if c.IsAborted() {
			return
		}

		// Then proxy the request
		proxyHandler(c)
	}
}

// setupPublicRoute sets up a public route
func (r *Router) setupPublicRoute(router *gin.Engine, route RouteConfig, handler gin.HandlerFunc) {
	pattern := route.PathPrefix + "/*path"

	if len(route.AllowedMethods) == 0 {
		// All methods allowed
		router.Any(pattern, handler)
	} else {
		for _, method := range route.AllowedMethods {
			switch strings.ToUpper(method) {
			case "GET":
				router.GET(pattern, handler)
			case "POST":
				router.POST(pattern, handler)
			case "PUT":
				router.PUT(pattern, handler)
			case "DELETE":
				router.DELETE(pattern, handler)
			case "PATCH":
				router.PATCH(pattern, handler)
			case "OPTIONS":
				router.OPTIONS(pattern, handler)
			case "HEAD":
				router.HEAD(pattern, handler)
			}
		}
	}
}

// setupProtectedRoute sets up a protected route with JWT middleware
func (r *Router) setupProtectedRoute(router *gin.Engine, route RouteConfig, handler gin.HandlerFunc) {
	pattern := route.PathPrefix + "/*path"

	if len(route.AllowedMethods) == 0 {
		// All methods allowed
		router.Any(pattern, handler)
	} else {
		for _, method := range route.AllowedMethods {
			switch strings.ToUpper(method) {
			case "GET":
				router.GET(pattern, handler)
			case "POST":
				router.POST(pattern, handler)
			case "PUT":
				router.PUT(pattern, handler)
			case "DELETE":
				router.DELETE(pattern, handler)
			case "PATCH":
				router.PATCH(pattern, handler)
			case "OPTIONS":
				router.OPTIONS(pattern, handler)
			case "HEAD":
				router.HEAD(pattern, handler)
			}
		}
	}
}

// SetupRoutesV2 provides a cleaner API for setting up routes
// with better control over route ordering and method handling
func (r *Router) SetupRoutesV2(engine *gin.Engine) {
	// Group routes by path prefix to handle method-based auth correctly
	routeGroups := make(map[string][]RouteConfig)
	for _, route := range r.proxy.config.Routes {
		routeGroups[route.PathPrefix] = append(routeGroups[route.PathPrefix], route)
	}

	jwtMiddleware := pkgmiddleware.JWTMiddleware(r.jwtConfig)
	proxyHandler := r.proxy.Handler()

	for prefix, routes := range routeGroups {
		group := engine.Group(prefix)

		// Separate public and protected routes
		var publicMethods []string
		var protectedMethods []string

		for _, route := range routes {
			if route.RequireAuth {
				if len(route.AllowedMethods) == 0 {
					protectedMethods = append(protectedMethods, "ALL")
				} else {
					protectedMethods = append(protectedMethods, route.AllowedMethods...)
				}
			} else {
				if len(route.AllowedMethods) == 0 {
					publicMethods = append(publicMethods, "ALL")
				} else {
					publicMethods = append(publicMethods, route.AllowedMethods...)
				}
			}
		}

		// Register public methods
		for _, method := range publicMethods {
			switch strings.ToUpper(method) {
			case "ALL":
				group.Any("", proxyHandler)
				group.Any("/*path", proxyHandler)
			case "GET":
				group.GET("", proxyHandler)
				group.GET("/*path", proxyHandler)
			case "POST":
				group.POST("", proxyHandler)
				group.POST("/*path", proxyHandler)
			case "PUT":
				group.PUT("", proxyHandler)
				group.PUT("/*path", proxyHandler)
			case "DELETE":
				group.DELETE("", proxyHandler)
				group.DELETE("/*path", proxyHandler)
			case "PATCH":
				group.PATCH("", proxyHandler)
				group.PATCH("/*path", proxyHandler)
			}
		}

		// Register protected methods (with JWT middleware)
		protectedGroup := group.Group("")
		protectedGroup.Use(jwtMiddleware)

		for _, method := range protectedMethods {
			switch strings.ToUpper(method) {
			case "ALL":
				protectedGroup.Any("", proxyHandler)
				protectedGroup.Any("/*path", proxyHandler)
			case "GET":
				protectedGroup.GET("", proxyHandler)
				protectedGroup.GET("/*path", proxyHandler)
			case "POST":
				protectedGroup.POST("", proxyHandler)
				protectedGroup.POST("/*path", proxyHandler)
			case "PUT":
				protectedGroup.PUT("", proxyHandler)
				protectedGroup.PUT("/*path", proxyHandler)
			case "DELETE":
				protectedGroup.DELETE("", proxyHandler)
				protectedGroup.DELETE("/*path", proxyHandler)
			case "PATCH":
				protectedGroup.PATCH("", proxyHandler)
				protectedGroup.PATCH("/*path", proxyHandler)
			}
		}
	}
}

// MatchHandler returns a handler that uses the proxy's route matching
// This is useful for catch-all routes
func (r *Router) MatchHandler() gin.HandlerFunc {
	jwtMiddleware := pkgmiddleware.JWTMiddleware(r.jwtConfig)
	proxyHandler := r.proxy.Handler()

	return func(c *gin.Context) {
		// Find matching route
		route := r.proxy.findRoute(c.Request.URL.Path, c.Request.Method)
		if route == nil {
			c.JSON(404, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Route not found",
				},
			})
			c.Abort()
			return
		}

		// Apply JWT middleware if required
		if route.RequireAuth {
			jwtMiddleware(c)
			if c.IsAborted() {
				return
			}
		}

		// Proxy the request
		proxyHandler(c)
	}
}
