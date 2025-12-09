package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	pkgredis "github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Rate limit per second per IP (0 = unlimited)
	RequestsPerSecond int
	// Burst size (token bucket capacity)
	BurstSize int
	// Whether to use Redis for distributed rate limiting
	UseRedis bool
	// Redis client (required if UseRedis is true)
	RedisClient *pkgredis.Client
	// Key prefix for Redis
	KeyPrefix string
	// Cleanup interval for local rate limiter
	CleanupInterval time.Duration
	// Entry TTL for local rate limiter
	EntryTTL time.Duration
}

// DefaultRateLimitConfig returns sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 1000,         // 1000 req/s per IP
		BurstSize:         100,          // Allow burst of 100
		UseRedis:          false,        // Local by default for speed
		KeyPrefix:         "ratelimit:", // Redis key prefix
		CleanupInterval:   time.Minute,  // Cleanup stale entries every minute
		EntryTTL:          time.Minute,  // Entries expire after 1 minute of inactivity
	}
}

// rateLimitEntry tracks rate limit state for an IP
type rateLimitEntry struct {
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// LocalRateLimiter implements in-memory token bucket rate limiting
type LocalRateLimiter struct {
	config  RateLimitConfig
	entries sync.Map
	stop    chan struct{}

	// Metrics
	totalAllowed  uint64
	totalRejected uint64
}

// NewLocalRateLimiter creates a new local rate limiter
func NewLocalRateLimiter(config RateLimitConfig) *LocalRateLimiter {
	rl := &LocalRateLimiter{
		config: config,
		stop:   make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request should be allowed
func (rl *LocalRateLimiter) Allow(key string) bool {
	now := time.Now()

	// Get or create entry
	entry, _ := rl.entries.LoadOrStore(key, &rateLimitEntry{
		tokens:     float64(rl.config.BurstSize),
		lastUpdate: now,
	})
	e := entry.(*rateLimitEntry)

	e.mu.Lock()
	defer e.mu.Unlock()

	// Calculate tokens to add based on time elapsed
	elapsed := now.Sub(e.lastUpdate).Seconds()
	tokensToAdd := elapsed * float64(rl.config.RequestsPerSecond)
	e.tokens = min(float64(rl.config.BurstSize), e.tokens+tokensToAdd)
	e.lastUpdate = now

	// Check if we have tokens available
	if e.tokens >= 1 {
		e.tokens--
		atomic.AddUint64(&rl.totalAllowed, 1)
		return true
	}

	atomic.AddUint64(&rl.totalRejected, 1)
	return false
}

// GetStats returns rate limiter statistics
func (rl *LocalRateLimiter) GetStats() (allowed, rejected uint64) {
	return atomic.LoadUint64(&rl.totalAllowed), atomic.LoadUint64(&rl.totalRejected)
}

// cleanup periodically removes stale entries
func (rl *LocalRateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-rl.config.EntryTTL)
			rl.entries.Range(func(key, value interface{}) bool {
				e := value.(*rateLimitEntry)
				e.mu.Lock()
				if e.lastUpdate.Before(cutoff) {
					rl.entries.Delete(key)
				}
				e.mu.Unlock()
				return true
			})
		case <-rl.stop:
			return
		}
	}
}

// Stop stops the cleanup goroutine
func (rl *LocalRateLimiter) Stop() {
	close(rl.stop)
}

// RedisRateLimiter implements Redis-based distributed rate limiting
type RedisRateLimiter struct {
	config RateLimitConfig
	script string
}

// NewRedisRateLimiter creates a new Redis rate limiter
func NewRedisRateLimiter(config RateLimitConfig) *RedisRateLimiter {
	// Lua script for atomic token bucket rate limiting
	script := `
local key = KEYS[1]
local rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local now = tonumber(ARGV[3])
local requested = 1

local data = redis.call("HMGET", key, "tokens", "last_update")
local tokens = tonumber(data[1]) or burst
local last_update = tonumber(data[2]) or now

-- Calculate tokens to add
local elapsed = now - last_update
local tokens_to_add = elapsed * rate
tokens = math.min(burst, tokens + tokens_to_add)

-- Check if request is allowed
if tokens >= requested then
    tokens = tokens - requested
    redis.call("HMSET", key, "tokens", tokens, "last_update", now)
    redis.call("EXPIRE", key, 60)
    return {1, tokens}
else
    redis.call("HMSET", key, "tokens", tokens, "last_update", now)
    redis.call("EXPIRE", key, 60)
    return {0, tokens}
end
`
	return &RedisRateLimiter{
		config: config,
		script: script,
	}
}

// Allow checks if a request should be allowed using Redis
func (rl *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	now := float64(time.Now().UnixNano()) / 1e9

	result := rl.config.RedisClient.Eval(ctx, rl.script,
		[]string{rl.config.KeyPrefix + key},
		float64(rl.config.RequestsPerSecond),
		float64(rl.config.BurstSize),
		now,
	)

	if result.Err() != nil {
		return false, result.Err()
	}

	values, err := result.Slice()
	if err != nil {
		return false, err
	}

	if len(values) < 1 {
		return false, fmt.Errorf("unexpected result length")
	}

	allowed, _ := values[0].(int64)
	return allowed == 1, nil
}

// RateLimiter creates a rate limiting middleware
func RateLimiter(config RateLimitConfig) gin.HandlerFunc {
	var localLimiter *LocalRateLimiter
	var redisLimiter *RedisRateLimiter

	if config.UseRedis && config.RedisClient != nil {
		redisLimiter = NewRedisRateLimiter(config)
	} else {
		localLimiter = NewLocalRateLimiter(config)
	}

	return func(c *gin.Context) {
		// Get client IP as rate limit key
		clientIP := c.ClientIP()

		var allowed bool
		var remaining int
		var err error

		startTime := time.Now()

		if redisLimiter != nil {
			allowed, err = redisLimiter.Allow(c.Request.Context(), clientIP)
			if err != nil {
				// Fallback to allowing on Redis errors (fail open)
				allowed = true
			}
		} else {
			allowed = localLimiter.Allow(clientIP)
		}

		// Calculate remaining tokens (approximation for headers)
		remaining = config.BurstSize - 1
		if !allowed {
			remaining = 0
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerSecond))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Second).Unix(), 10))

		if !allowed {
			// Calculate retry after (1 second default)
			retryAfter := 1
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			// Track rejection latency
			latency := time.Since(startTime)
			c.Header("X-RateLimit-Latency", latency.String())

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOO_MANY_REQUESTS",
					"message": "Rate limit exceeded. Please retry after " + strconv.Itoa(retryAfter) + " second(s).",
				},
			})
			return
		}

		c.Next()
	}
}

// RateLimiterWithDefault creates a rate limiting middleware with default config
func RateLimiterWithDefault() gin.HandlerFunc {
	return RateLimiter(DefaultRateLimitConfig())
}

// GlobalRateLimiter implements global (non-per-IP) rate limiting for spike protection
type GlobalRateLimiter struct {
	maxConcurrent int64
	currentCount  int64
	mu            sync.Mutex
}

// NewGlobalRateLimiter creates a new global rate limiter
func NewGlobalRateLimiter(maxConcurrent int64) *GlobalRateLimiter {
	return &GlobalRateLimiter{
		maxConcurrent: maxConcurrent,
	}
}

// Acquire tries to acquire a slot
func (g *GlobalRateLimiter) Acquire() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.currentCount >= g.maxConcurrent {
		return false
	}
	g.currentCount++
	return true
}

// Release releases a slot
func (g *GlobalRateLimiter) Release() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.currentCount > 0 {
		g.currentCount--
	}
}

// CurrentCount returns the current concurrent request count
func (g *GlobalRateLimiter) CurrentCount() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.currentCount
}

// ConcurrencyLimiter creates a middleware that limits concurrent requests
func ConcurrencyLimiter(maxConcurrent int64) gin.HandlerFunc {
	limiter := NewGlobalRateLimiter(maxConcurrent)

	return func(c *gin.Context) {
		if !limiter.Acquire() {
			c.Header("Retry-After", "1")
			c.Header("X-Concurrency-Limit", strconv.FormatInt(maxConcurrent, 10))
			c.Header("X-Concurrency-Current", strconv.FormatInt(limiter.CurrentCount(), 10))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "TOO_MANY_REQUESTS",
					"message": "Server is at capacity. Please retry in a moment.",
				},
			})
			return
		}

		defer limiter.Release()
		c.Next()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
