# Client Retry Guidelines

This document provides guidelines for client applications when handling errors and implementing retry logic for the Booking Rush API.

## Overview

The Booking Rush API handles high-concurrency ticket booking with 10,000+ RPS. Clients must implement proper retry strategies to:
- Avoid overwhelming the system during spikes
- Maximize successful bookings
- Provide good user experience

## HTTP Response Codes

| Code | Meaning | Should Retry? |
|------|---------|---------------|
| 201 | Success - Booking created | No |
| 400 | Bad Request - Invalid input | No |
| 401 | Unauthorized | No |
| 403 | Forbidden | No |
| 404 | Not Found | No |
| 409 | Conflict (Insufficient seats, already confirmed) | Depends |
| 429 | Too Many Requests - Rate limited | Yes |
| 500 | Internal Server Error | Yes (with backoff) |
| 503 | Service Unavailable | Yes (with backoff) |

## Error Codes and Retry Strategy

### Never Retry
These errors indicate permanent failures:

| Error Code | Description | User Action |
|------------|-------------|-------------|
| `INVALID_REQUEST` | Malformed request | Fix request and resubmit |
| `INVALID_QUANTITY` | Quantity <= 0 | Fix quantity |
| `UNAUTHORIZED` | Missing/invalid auth | Re-authenticate |
| `FORBIDDEN` | Not allowed | Check permissions |
| `INSUFFICIENT_SEATS` | Zone sold out | Show "sold out" message |
| `MAX_TICKETS_EXCEEDED` | User already at max | Inform user of limit |
| `ALREADY_CONFIRMED` | Booking already confirmed | Redirect to confirmation |
| `ALREADY_RELEASED` | Booking already released | Start new booking |
| `EXPIRED` | Reservation expired | Start new booking |

### Retry with Same Idempotency Key
These errors should be retried with the **same idempotency key**:

| Error Code | Description | Retry Strategy |
|------------|-------------|----------------|
| `TOO_MANY_REQUESTS` | Rate limited | Wait for Retry-After |
| `INTERNAL_ERROR` | Server error | Exponential backoff |
| `SERVICE_UNAVAILABLE` | System overloaded | Exponential backoff |

## Exponential Backoff Strategy

### Algorithm

```python
def exponential_backoff(attempt, base_delay=1.0, max_delay=30.0, jitter=True):
    """
    Calculate delay for exponential backoff with jitter.

    Args:
        attempt: Current attempt number (0-indexed)
        base_delay: Initial delay in seconds
        max_delay: Maximum delay cap in seconds
        jitter: Add randomization to prevent thundering herd

    Returns:
        Delay in seconds
    """
    delay = min(base_delay * (2 ** attempt), max_delay)

    if jitter:
        # Add random jitter between 0-100% of delay
        delay = delay * (0.5 + random.random())

    return delay
```

### Example Implementation (JavaScript)

```javascript
async function reserveWithRetry(payload, options = {}) {
    const {
        maxRetries = 3,
        baseDelay = 1000,  // 1 second
        maxDelay = 30000,  // 30 seconds
        idempotencyKey = generateIdempotencyKey(),
    } = options;

    for (let attempt = 0; attempt <= maxRetries; attempt++) {
        try {
            const response = await fetch('/bookings/reserve', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Idempotency-Key': idempotencyKey,
                },
                body: JSON.stringify(payload),
            });

            if (response.ok) {
                return response.json();
            }

            const error = await response.json();

            // Non-retryable errors
            if (!isRetryable(response.status, error.code)) {
                throw new BookingError(error.code, error.message);
            }

            // Check Retry-After header
            const retryAfter = response.headers.get('Retry-After');
            let delay;

            if (retryAfter) {
                delay = parseInt(retryAfter, 10) * 1000;
            } else {
                delay = Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
                delay = delay * (0.5 + Math.random()); // Jitter
            }

            if (attempt < maxRetries) {
                await sleep(delay);
            }

        } catch (networkError) {
            // Network errors are retryable
            if (attempt >= maxRetries) {
                throw networkError;
            }

            const delay = Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
            await sleep(delay * (0.5 + Math.random()));
        }
    }

    throw new Error('Max retries exceeded');
}

function isRetryable(status, code) {
    // Rate limiting
    if (status === 429) return true;

    // Server errors
    if (status >= 500 && status < 600) return true;

    // Specific retryable codes
    const retryableCodes = ['INTERNAL_ERROR', 'SERVICE_UNAVAILABLE'];
    return retryableCodes.includes(code);
}

function sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
}
```

### Example Implementation (Go)

```go
func ReserveWithRetry(ctx context.Context, payload *ReserveRequest, opts *RetryOptions) (*Booking, error) {
    if opts == nil {
        opts = DefaultRetryOptions()
    }

    var lastErr error

    for attempt := 0; attempt <= opts.MaxRetries; attempt++ {
        booking, err := doReserve(ctx, payload, opts.IdempotencyKey)
        if err == nil {
            return booking, nil
        }

        apiErr, ok := err.(*APIError)
        if !ok || !apiErr.Retryable {
            return nil, err
        }

        lastErr = err

        if attempt < opts.MaxRetries {
            delay := calculateBackoff(attempt, opts)

            // Use Retry-After header if available
            if apiErr.RetryAfter > 0 {
                delay = time.Duration(apiErr.RetryAfter) * time.Second
            }

            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(delay):
            }
        }
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func calculateBackoff(attempt int, opts *RetryOptions) time.Duration {
    delay := opts.BaseDelay * time.Duration(1<<uint(attempt))
    if delay > opts.MaxDelay {
        delay = opts.MaxDelay
    }

    // Add jitter
    jitter := time.Duration(rand.Float64() * float64(delay))
    return delay/2 + jitter
}
```

## Rate Limit Headers

The API returns rate limit information in response headers:

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests per second |
| `X-RateLimit-Remaining` | Requests remaining in current window |
| `X-RateLimit-Reset` | Unix timestamp when window resets |
| `Retry-After` | Seconds to wait before retrying (on 429) |

### Example Headers

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1699876543
Retry-After: 1
Content-Type: application/json

{
    "success": false,
    "error": {
        "code": "TOO_MANY_REQUESTS",
        "message": "Rate limit exceeded. Please retry after 1 second(s)."
    }
}
```

## When to Stop Retrying

### Stop Immediately
- `INSUFFICIENT_SEATS` - Zone is sold out
- `MAX_TICKETS_EXCEEDED` - User at limit
- `EXPIRED` - Reservation expired
- Authentication/authorization errors

### Stop After N Retries
- `TOO_MANY_REQUESTS` - After 3-5 retries
- `INTERNAL_ERROR` - After 3 retries
- `SERVICE_UNAVAILABLE` - After 3 retries

### Recommended Limits

```javascript
const RETRY_LIMITS = {
    TOO_MANY_REQUESTS: 5,    // Can retry more for rate limits
    INTERNAL_ERROR: 3,        // Fewer retries for server errors
    SERVICE_UNAVAILABLE: 3,
    NETWORK_ERROR: 3,
};
```

## Idempotency Keys

Always include an idempotency key to prevent duplicate bookings:

```javascript
// Generate unique idempotency key per booking attempt
function generateIdempotencyKey() {
    return `${userId}-${eventId}-${Date.now()}-${Math.random().toString(36).substring(7)}`;
}

// IMPORTANT: Reuse the SAME key for retries
const idempotencyKey = generateIdempotencyKey();
for (let i = 0; i < maxRetries; i++) {
    // Use SAME idempotencyKey for all retry attempts
    await reserve({ ...payload, idempotencyKey });
}
```

## Best Practices

### 1. Always Use Idempotency Keys
Prevents duplicate bookings on network failures or timeouts.

### 2. Implement Circuit Breaker
After multiple failures, stop retrying temporarily:

```javascript
class CircuitBreaker {
    constructor(threshold = 5, timeout = 30000) {
        this.failures = 0;
        this.threshold = threshold;
        this.timeout = timeout;
        this.lastFailure = null;
        this.state = 'CLOSED';
    }

    async call(fn) {
        if (this.state === 'OPEN') {
            if (Date.now() - this.lastFailure > this.timeout) {
                this.state = 'HALF-OPEN';
            } else {
                throw new Error('Circuit breaker is open');
            }
        }

        try {
            const result = await fn();
            this.onSuccess();
            return result;
        } catch (error) {
            this.onFailure();
            throw error;
        }
    }

    onSuccess() {
        this.failures = 0;
        this.state = 'CLOSED';
    }

    onFailure() {
        this.failures++;
        this.lastFailure = Date.now();
        if (this.failures >= this.threshold) {
            this.state = 'OPEN';
        }
    }
}
```

### 3. Show User Feedback
- Display "Booking..." spinner during retry
- Show countdown for rate limit waits
- Offer "Try another zone" for sold out

### 4. Pre-check Availability
Before booking, check zone availability to avoid unnecessary requests:

```javascript
async function checkAndReserve(eventId, zoneId, quantity) {
    // Check availability first (cached, cheap)
    const availability = await getZoneAvailability(eventId, zoneId);

    if (availability < quantity) {
        throw new Error('INSUFFICIENT_SEATS');
    }

    // Proceed with booking
    return reserveWithRetry({ eventId, zoneId, quantity });
}
```

### 5. Handle Timeout Properly
Set appropriate timeouts and handle them:

```javascript
const controller = new AbortController();
const timeout = setTimeout(() => controller.abort(), 10000); // 10s timeout

try {
    const response = await fetch('/bookings/reserve', {
        signal: controller.signal,
        // ...
    });
} catch (error) {
    if (error.name === 'AbortError') {
        // Timeout - safe to retry with same idempotency key
    }
} finally {
    clearTimeout(timeout);
}
```

## Summary

| Scenario | Action |
|----------|--------|
| Rate limited (429) | Wait for Retry-After, retry with same key |
| Server error (5xx) | Exponential backoff, retry with same key |
| Sold out (409) | Stop, show "sold out" message |
| Network timeout | Retry with same idempotency key |
| Invalid request (4xx) | Stop, fix request |
| Max retries reached | Show error, offer alternatives |

## Contact

For API support or questions about retry strategies, contact the platform team.
