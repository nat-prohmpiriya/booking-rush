# Thundering Herd Rejection Efficiency Testing

This document describes tests for verifying that the Booking Service efficiently rejects requests under high load.

## Overview

The "thundering herd" problem occurs when many clients simultaneously compete for limited resources. This test suite verifies:

1. **Rate limiting** returns 429 responses quickly (< 5ms)
2. **Sold out zones** reject requests immediately via Lua script (< 1ms at Redis level)
3. **No resource exhaustion** under 20k RPS spike
4. **Clear error messages** for client retry logic

## Test Files

| File | Purpose |
|------|---------|
| `thundering_herd.js` | k6 load test script for thundering herd scenarios |
| `setup_thundering_herd.js` | Redis seed data setup script |
| `thundering_herd_seed.json` | Generated test data (after running setup) |
| `tests/integration/thundering_herd_test.go` | Go integration tests |

## Scenarios

### 1. Rate Limit 429 Response Speed

**Scenario:** High RPS triggers rate limiting

**Expected Behavior:**
- 429 responses returned < 5ms (local Redis)
- `X-RateLimit-*` headers present
- `Retry-After` header present

**Test:**
```bash
k6 run thundering_herd.js --env SCENARIO=rate_limit_429
```

### 2. Sold Out Rejection Speed

**Scenario:** Requests to a sold-out zone

**Expected Behavior:**
- Lua script checks availability and returns immediately
- No DB connections used for rejection
- Response time < 5ms (local Redis)

**Test:**
```bash
k6 run thundering_herd.js --env SCENARIO=sold_out_rejection
```

**Implementation:**
```lua
-- reserve_seats.lua:59-61
if available < quantity then
    return {0, "INSUFFICIENT_STOCK", "Not enough seats available..."}
end
```

### 3. 20k RPS Spike Test

**Scenario:** Traffic spikes from 1k to 20k RPS

**Expected Behavior:**
- System handles spike gracefully
- No 503 errors (resource exhaustion)
- Rate limiting kicks in to protect system

**Test:**
```bash
k6 run thundering_herd.js --env SCENARIO=spike_20k
```

### 4. Thundering Herd on Limited Inventory

**Scenario:** 100 concurrent requests for 10 seats

**Expected Behavior:**
- Exactly 10 winners
- 90 requests get INSUFFICIENT_SEATS
- No negative inventory

**Test:**
```bash
k6 run thundering_herd.js --env SCENARIO=thundering_herd
```

## Running Tests

### k6 Load Tests

```bash
# Setup test data in Redis
cd tests/load
REDIS_HOST=localhost REDIS_PORT=6379 node setup_thundering_herd.js

# Run all scenarios
k6 run thundering_herd.js --env SCENARIO=all

# Run specific scenario
k6 run thundering_herd.js --env SCENARIO=rate_limit_429
k6 run thundering_herd.js --env SCENARIO=sold_out_rejection
k6 run thundering_herd.js --env SCENARIO=spike_20k
k6 run thundering_herd.js --env SCENARIO=thundering_herd

# Reset test data
REDIS_HOST=localhost node setup_thundering_herd.js reset
```

### Go Integration Tests

```bash
# Run all thundering herd tests
INTEGRATION_TEST=true \
TEST_REDIS_HOST=localhost \
TEST_REDIS_PASSWORD=<password> \
go test ./tests/integration/... -v -run "TestThunderingHerd"

# Run specific test
INTEGRATION_TEST=true TEST_REDIS_HOST=localhost \
go test ./tests/integration/... -v -run TestThunderingHerd_SoldOutRejectionSpeed
```

## Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| `rejection_429_duration_ms` | P95/P99 of 429 response time | P95 < 5ms |
| `rejection_429_under_5ms` | % of 429 responses under 5ms | > 95% |
| `sold_out_duration_ms` | P95/P99 of sold out rejection | P95 < 5ms |
| `sold_out_under_5ms` | % of sold out responses under 5ms | > 95% |
| `rate_limit_headers_present` | % of responses with rate limit headers | > 99% |
| `retry_after_header_present` | % of 429s with Retry-After header | > 90% |
| `successful_requests` | Number of successful bookings | N/A |
| `rate_limited_requests` | Number of rate limited requests | N/A |
| `server_errors` | Number of 5xx errors | 0 |

## Expected Results

| Scenario | Expected Outcome |
|----------|------------------|
| Rate Limit 429 | < 5ms response, headers present |
| Sold Out | < 5ms rejection, clear error message |
| 20k RPS Spike | No 503 errors, graceful degradation |
| Thundering Herd | Exactly N winners for N seats |

## Architecture Protection Layers

### 1. API Gateway Rate Limiting (429)

```go
// middleware/rate_limiter.go
// Token bucket algorithm with in-memory or Redis backend
// Returns 429 with proper headers when limit exceeded

c.Header("X-RateLimit-Limit", strconv.Itoa(config.RequestsPerSecond))
c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
c.Header("Retry-After", strconv.Itoa(retryAfter))
```

### 2. Redis Lua Script (Instant Rejection)

```lua
-- reserve_seats.lua
-- Checks availability BEFORE any writes
local available = redis.call("GET", zone_availability_key)
if available < quantity then
    return {0, "INSUFFICIENT_STOCK", "Not enough seats available..."}
end
```

### 3. Concurrency Limiter

```go
// middleware/rate_limiter.go
// ConcurrencyLimiter prevents too many concurrent requests
if !limiter.Acquire() {
    c.AbortWithStatusJSON(429, ...)
}
defer limiter.Release()
```

## Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| 429 Response Time | < 5ms (P95) | With local Redis |
| Sold Out Rejection | < 1ms (Redis level) | Lua script returns immediately |
| No Resource Exhaustion | 0 goroutine leaks | Under 20k RPS |
| Error Message Quality | Clear, actionable | Client can decide retry/stop |

## Rate Limit Headers

The API returns the following headers:

```
X-RateLimit-Limit: 1000        # Max requests per second
X-RateLimit-Remaining: 42      # Requests remaining
X-RateLimit-Reset: 1699876543  # Unix timestamp of reset
Retry-After: 1                 # Seconds to wait (on 429)
```

## Client Retry Guidelines

See [CLIENT_RETRY_GUIDELINES.md](../../docs/CLIENT_RETRY_GUIDELINES.md) for:
- Exponential backoff algorithm
- When to retry vs stop
- Idempotency key usage

## Troubleshooting

### High 429 Response Times

1. Check Redis connection pool size
2. Verify Redis is not overloaded
3. Consider local rate limiter instead of Redis-based

### Resource Exhaustion (503 errors)

1. Check goroutine count with pprof
2. Increase concurrency limit
3. Add circuit breaker

### Missing Rate Limit Headers

1. Verify rate limiter middleware is applied
2. Check middleware order (rate limiter should be early)

## Test Environment Notes

For remote Redis (e.g., 100.104.0.42):
- Network latency will dominate response times (~30-150ms)
- Tests auto-detect remote Redis and relax timing expectations
- The < 5ms target is for production with local Redis
