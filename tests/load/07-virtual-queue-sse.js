// k6 Load Test Script for Virtual Queue + Booking Flow (SSE Version)
// Target: 10,000 concurrent users → Virtual Queue (SSE) → 500 TPS booking throughput
// Improvement: SSE reduces polling overhead by 50x (5000 req/s → 100 req/s for 10K users)
//
// Prerequisites: k6 will automatically resolve xk6-sse extension
// Reference: https://github.com/phymbert/xk6-sse

import http from 'k6/http';
import { check, sleep, fail } from 'k6';
import { Rate, Trend, Counter, Gauge } from 'k6/metrics';
import { SharedArray } from 'k6/data';
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import sse from 'k6/x/sse';

// ============================================================================
// Custom Metrics
// ============================================================================
const queueJoinSuccess = new Rate('queue_join_success');
const queuePassReceived = new Rate('queue_pass_received');
const bookingSuccess = new Rate('booking_success');
const bookingFailed = new Rate('booking_failed');
const insufficientSeats = new Counter('insufficient_seats');
const serverErrors = new Counter('server_errors');
const queueWaitTime = new Trend('queue_wait_time');
const bookingDuration = new Trend('booking_duration');
const totalCompleteFlow = new Counter('total_complete_flow');
const currentInQueue = new Gauge('current_in_queue');
const queuePassExpired = new Counter('queue_pass_expired');
const sseConnections = new Counter('sse_connections');
const sseErrors = new Counter('sse_errors');

// ============================================================================
// Configuration
// ============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';
const MAX_SSE_WAIT_TIME = 300; // 5 minutes max wait for SSE stream
const USE_SSE = __ENV.USE_SSE !== 'false'; // Enable SSE by default

// Load test data
let testDataConfig;
try {
    testDataConfig = JSON.parse(open('./seed-data/data.json'));
} catch (e) {
    testDataConfig = {
        eventIds: ['b0000000-0000-0001-0000-000000000001'],
        showIds: ['b0000000-0000-0001-0001-000000000001'],
        zoneIds: ['b0000000-0000-0001-0001-000000000001'],
    };
}

const eventIds = testDataConfig.eventIds || [];
const showIds = testDataConfig.showIds || [];
const zoneIds = testDataConfig.zoneIds || [];

// Load pre-generated tokens
let userTokens = [];
try {
    userTokens = JSON.parse(open('./seed-data/tokens.json'));
    console.log(`Loaded ${userTokens.length} pre-generated tokens`);
} catch (e) {
    console.log('No pre-generated tokens found');
}

const tokens = new SharedArray('tokens', function () {
    return userTokens.length > 0 ? userTokens : [];
});

// Get scenario from environment
const SCENARIO = __ENV.SCENARIO || 'sse_10k';

// ============================================================================
// Scenarios
// ============================================================================
const allScenarios = {
    // Quick test: 100 users with SSE
    sse_smoke: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '30s', target: 100 },
            { duration: '2m', target: 100 },
            { duration: '30s', target: 0 },
        ],
        tags: { scenario: 'sse_smoke' },
        exec: 'virtualQueueSSEFlow',
    },

    // Medium test: 1000 users with SSE
    sse_1k: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '30s', target: 500 },
            { duration: '1m', target: 1000 },
            { duration: '3m', target: 1000 },
            { duration: '30s', target: 0 },
        ],
        tags: { scenario: 'sse_1k' },
        exec: 'virtualQueueSSEFlow',
    },

    // Medium-high test: 3000 users with SSE
    sse_3k: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '30s', target: 1000 },   // Ramp to 1k
            { duration: '1m', target: 2000 },    // Ramp to 2k
            { duration: '1m', target: 3000 },    // Ramp to 3k
            { duration: '4m', target: 3000 },    // Sustain 3k
            { duration: '30s', target: 0 },      // Ramp down
        ],
        tags: { scenario: 'sse_3k' },
        exec: 'virtualQueueSSEFlow',
    },

    // High test: 5000 users with SSE
    sse_5k: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '30s', target: 1500 },   // Ramp to 1.5k
            { duration: '1m', target: 3000 },    // Ramp to 3k
            { duration: '1m', target: 5000 },    // Ramp to 5k
            { duration: '4m', target: 5000 },    // Sustain 5k
            { duration: '30s', target: 0 },      // Ramp down
        ],
        tags: { scenario: 'sse_5k' },
        exec: 'virtualQueueSSEFlow',
    },

    // Main test: 10,000 concurrent users with SSE
    sse_10k: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '1m', target: 2000 },    // Ramp to 2k
            { duration: '1m', target: 5000 },    // Ramp to 5k
            { duration: '1m', target: 10000 },   // Ramp to 10k
            { duration: '5m', target: 10000 },   // Sustain 10k
            { duration: '1m', target: 0 },       // Ramp down
        ],
        tags: { scenario: 'sse_10k' },
        exec: 'virtualQueueSSEFlow',
    },

    // Hybrid: Use polling for comparison
    polling_10k: {
        executor: 'ramping-vus',
        startVUs: 0,
        stages: [
            { duration: '1m', target: 2000 },
            { duration: '1m', target: 5000 },
            { duration: '1m', target: 10000 },
            { duration: '5m', target: 10000 },
            { duration: '1m', target: 0 },
        ],
        tags: { scenario: 'polling_10k' },
        exec: 'virtualQueuePollingFlow',
    },
};

const selectedScenarios = SCENARIO === 'all'
    ? allScenarios
    : { [SCENARIO]: allScenarios[SCENARIO] };

export const options = {
    scenarios: selectedScenarios,
    thresholds: {
        'queue_join_success': ['rate>0.95'],        // 95% should join queue
        'queue_pass_received': ['rate>0.80'],       // 80% should get pass
        'booking_success': ['rate>0.90'],           // 90% of those with pass should book
        'http_req_failed': ['rate<0.10'],           // <10% HTTP errors
        'booking_duration': ['p(95)<2000'],         // 95% booking < 2s
    },
};

// ============================================================================
// Main Flow with SSE: Join Queue → Stream Position → Book
// Uses k6/experimental/sse for real-time event handling
// ============================================================================
export function virtualQueueSSEFlow() {
    // Get random user token
    let token, userId;
    if (tokens.length > 0) {
        const tokenIndex = __VU % tokens.length;
        const tokenData = tokens[tokenIndex];
        token = tokenData.token;
        userId = tokenData.user_id;
    } else {
        fail('No tokens available. Run token generation first.');
    }

    const eventId = randomItem(eventIds);
    const showId = randomItem(showIds);
    const zoneId = randomItem(zoneIds);

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
    };

    // ========================================
    // Step 1: Join Queue
    // ========================================
    const joinIdempotencyKey = `queue-join-${userId}-${eventId}-${Date.now()}`;
    const joinPayload = JSON.stringify({
        event_id: eventId,
    });

    const joinHeaders = {
        ...headers,
        'X-Idempotency-Key': joinIdempotencyKey,
    };

    const joinResponse = http.post(`${BASE_URL}/queue/join`, joinPayload, {
        headers: joinHeaders,
        tags: { name: 'JoinQueue' },
    });

    const joinSuccess = check(joinResponse, {
        'join queue status 200/201': (r) => r.status === 200 || r.status === 201,
    });

    queueJoinSuccess.add(joinSuccess);

    if (!joinSuccess) {
        if (joinResponse.status !== 409) {
            serverErrors.add(1);
            return;
        }
    }

    // ========================================
    // Step 2: SSE Stream for Queue Pass (using k6/experimental/sse)
    // ========================================
    let queuePass = null;
    let lastPosition = 0;
    const queueStartTime = Date.now();

    sseConnections.add(1);

    const sseUrl = `${BASE_URL}/queue/position/${eventId}/stream`;
    const sseParams = {
        headers: {
            'Authorization': `Bearer ${token}`,
        },
        tags: { name: 'SSEStream' },
    };

    // Use k6/x/sse for real-time event handling
    // Reference: https://github.com/phymbert/xk6-sse
    const response = sse.open(sseUrl, sseParams, function (client) {
        // Handle connection open
        client.on('open', function () {
            // Connection opened successfully
        });

        // Handle all SSE events (event.name contains event type: 'position', 'error', etc.)
        client.on('event', function (event) {
            try {
                // Server sends: event: position\ndata: {...}\n\n
                // event.name = 'position', event.data = JSON string
                if (event.name === 'position') {
                    const data = JSON.parse(event.data);
                    lastPosition = data.position || 0;

                    // Check if we got queue pass
                    if (data.queue_pass) {
                        queuePass = data.queue_pass;
                        client.close(); // Got queue pass, close connection
                    }

                    // Also check is_ready flag
                    if (data.is_ready && data.queue_pass) {
                        queuePass = data.queue_pass;
                        client.close();
                    }
                } else if (event.name === 'error') {
                    // Server sent error event
                    sseErrors.add(1);
                    client.close();
                }
            } catch (e) {
                // Parse error, continue listening
            }
        });

        // Handle connection errors
        client.on('error', function (e) {
            sseErrors.add(1);
            client.close();
        });
    });

    // Check if SSE connection was successful
    if (response.status !== 200) {
        sseErrors.add(1);
    }

    const queueEndTime = Date.now();
    const waitTime = (queueEndTime - queueStartTime) / 1000;
    queueWaitTime.add(waitTime);

    // If no queue pass from SSE, try fallback polling once
    if (!queuePass) {
        const positionResponse = http.get(
            `${BASE_URL}/queue/position/${eventId}`,
            { headers: headers, tags: { name: 'GetPositionFallback' } }
        );
        if (positionResponse.status === 200) {
            try {
                const posData = JSON.parse(positionResponse.body);
                queuePass = posData.queue_pass || posData.data?.queue_pass;
            } catch (e) {}
        }
    }

    if (!queuePass) {
        queuePassReceived.add(false);
        return;
    }

    queuePassReceived.add(true);

    // ========================================
    // Step 3: Reserve with Queue Pass
    // ========================================
    const quantity = randomIntBetween(1, 2);
    const idempotencyKey = `${userId}-${zoneId}-${Date.now()}-${randomIntBetween(1, 1000000)}`;

    const bookingPayload = JSON.stringify({
        event_id: eventId,
        show_id: showId,
        zone_id: zoneId,
        quantity: quantity,
        unit_price: 100.00,
        queue_pass: queuePass,
    });

    const bookingHeaders = {
        ...headers,
        'X-Queue-Pass': queuePass,
        'X-Idempotency-Key': idempotencyKey,
    };

    const bookingStartTime = Date.now();
    const bookingResponse = http.post(`${BASE_URL}/bookings/reserve`, bookingPayload, {
        headers: bookingHeaders,
        tags: { name: 'ReserveWithPass' },
    });
    const bookingDur = Date.now() - bookingStartTime;
    bookingDuration.add(bookingDur);

    const bookingOk = check(bookingResponse, {
        'booking status 201': (r) => r.status === 201,
        'has booking_id': (r) => {
            try {
                const body = JSON.parse(r.body);
                return body.booking_id || body.data?.booking_id;
            } catch (e) {
                return false;
            }
        },
    });

    bookingSuccess.add(bookingOk);
    bookingFailed.add(!bookingOk);

    if (bookingOk) {
        totalCompleteFlow.add(1);
    } else {
        if (bookingResponse.status === 409) {
            insufficientSeats.add(1);
        } else if (bookingResponse.status === 401 || bookingResponse.status === 403) {
            queuePassExpired.add(1);
        } else if (bookingResponse.status >= 500) {
            serverErrors.add(1);
        }
    }

    sleep(randomIntBetween(100, 500) / 1000);
}

// ============================================================================
// Polling Flow (for comparison)
// ============================================================================
export function virtualQueuePollingFlow() {
    let token, userId;
    if (tokens.length > 0) {
        const tokenIndex = __VU % tokens.length;
        const tokenData = tokens[tokenIndex];
        token = tokenData.token;
        userId = tokenData.user_id;
    } else {
        fail('No tokens available.');
    }

    const eventId = randomItem(eventIds);
    const showId = randomItem(showIds);
    const zoneId = randomItem(zoneIds);

    const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
    };

    // Join Queue
    const joinIdempotencyKey = `queue-join-${userId}-${eventId}-${Date.now()}`;
    const joinResponse = http.post(`${BASE_URL}/queue/join`,
        JSON.stringify({ event_id: eventId }),
        { headers: { ...headers, 'X-Idempotency-Key': joinIdempotencyKey }, tags: { name: 'JoinQueue' } }
    );

    const joinSuccess = check(joinResponse, {
        'join queue status 200/201': (r) => r.status === 200 || r.status === 201,
    });
    queueJoinSuccess.add(joinSuccess);

    if (!joinSuccess && joinResponse.status !== 409) {
        serverErrors.add(1);
        return;
    }

    // Poll for Queue Pass
    let queuePass = null;
    const queueStartTime = Date.now();
    let pollCount = 0;
    const maxPolls = 150; // 5 min / 2s = 150 polls

    while (!queuePass && pollCount < maxPolls) {
        sleep(2); // Poll every 2 seconds
        pollCount++;

        const posResponse = http.get(
            `${BASE_URL}/queue/position/${eventId}`,
            { headers: headers, tags: { name: 'GetPosition' } }
        );

        if (posResponse.status === 200) {
            try {
                const posData = JSON.parse(posResponse.body);
                queuePass = posData.queue_pass || posData.data?.queue_pass;
                if (queuePass) break;
            } catch (e) {}
        }
    }

    const waitTime = (Date.now() - queueStartTime) / 1000;
    queueWaitTime.add(waitTime);

    if (!queuePass) {
        queuePassReceived.add(false);
        return;
    }
    queuePassReceived.add(true);

    // Book
    const quantity = randomIntBetween(1, 2);
    const idempotencyKey = `${userId}-${zoneId}-${Date.now()}-${randomIntBetween(1, 1000000)}`;

    const bookingStartTime = Date.now();
    const bookingResponse = http.post(`${BASE_URL}/bookings/reserve`,
        JSON.stringify({
            event_id: eventId,
            show_id: showId,
            zone_id: zoneId,
            quantity: quantity,
            unit_price: 100.00,
            queue_pass: queuePass,
        }),
        { headers: { ...headers, 'X-Queue-Pass': queuePass, 'X-Idempotency-Key': idempotencyKey }, tags: { name: 'ReserveWithPass' } }
    );
    bookingDuration.add(Date.now() - bookingStartTime);

    const bookingOk = check(bookingResponse, {
        'booking status 201': (r) => r.status === 201,
    });

    bookingSuccess.add(bookingOk);
    bookingFailed.add(!bookingOk);

    if (bookingOk) totalCompleteFlow.add(1);
    else if (bookingResponse.status >= 500) serverErrors.add(1);

    sleep(randomIntBetween(100, 500) / 1000);
}

// ============================================================================
// Lifecycle
// ============================================================================
export function setup() {
    console.log('='.repeat(60));
    console.log('Virtual Queue Load Test (SSE Version)');
    console.log('='.repeat(60));
    console.log(`Base URL: ${BASE_URL}`);
    console.log(`Scenario: ${SCENARIO}`);
    console.log(`SSE Enabled: ${USE_SSE}`);
    console.log(`Test Data: ${eventIds.length} events, ${zoneIds.length} zones`);
    console.log(`Tokens: ${tokens.length}`);
    console.log(`Max SSE Wait: ${MAX_SSE_WAIT_TIME}s`);
    console.log('='.repeat(60));

    const healthCheck = http.get('http://localhost:8080/health');
    if (healthCheck.status !== 200) {
        console.warn('Health check failed!');
    }

    return { startTime: new Date().toISOString() };
}

export function teardown(data) {
    console.log('='.repeat(60));
    console.log('Virtual Queue Load Test (SSE) Complete');
    console.log(`Started: ${data.startTime}`);
    console.log(`Ended: ${new Date().toISOString()}`);
    console.log('='.repeat(60));
    console.log('Key Metrics to Check:');
    console.log('  - queue_join_success: Should be > 95%');
    console.log('  - queue_pass_received: Should be > 80%');
    console.log('  - booking_success: Should be > 90%');
    console.log('  - sse_connections: Total SSE connections made');
    console.log('  - sse_errors: Should be 0 or very low');
    console.log('='.repeat(60));
}

export default function () {
    virtualQueueSSEFlow();
}
