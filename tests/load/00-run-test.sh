#!/bin/bash

# k6 Load Test Runner
# Usage: ./04-run-test.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

AUTH_URL="http://localhost:8080/api/v1/auth/login"

# DB connection
DB_CONTAINER="booking-rush-postgres"
DB_USER="postgres"
DB_PASSWORD="postgres"

# Function to generate tokens
generate_tokens() {
  echo ""
  echo "=== Generating JWT Tokens ==="

  read -p "Number of tokens to generate [500]: " num_tokens
  num_tokens=${num_tokens:-500}

  echo "Generating $num_tokens tokens..."
  cd "$SCRIPT_DIR/seed-data"

  NUM_TOKENS=$num_tokens node generate-tokens.js

  if [ $? -eq 0 ]; then
    TOKEN_COUNT=$(jq 'length' tokens.json 2>/dev/null)
    echo ""
    echo "=== Token Generation Complete ==="
    echo "  File: seed-data/tokens.json"
    echo "  Count: $TOKEN_COUNT tokens"
    echo "  Note: Tokens expire in ~15 minutes"
  else
    echo "ERROR: Token generation failed"
    return 1
  fi

  cd "$SCRIPT_DIR"
  echo ""
}

# Function to reset data
reset_data() {
  echo ""
  echo "=== Resetting Data ==="

  # Clear Redis
  echo "[1/4] Clearing Redis..."
  docker exec booking-rush-redis redis-cli -a redis123 --no-auth-warning FLUSHDB > /dev/null
  echo "Redis cleared"

  # Clear load test bookings from DB
  echo "[2/4] Clearing load test bookings from DB..."
  docker exec $DB_CONTAINER psql -U $DB_USER -d booking_db -c \
    "DELETE FROM bookings WHERE user_id::text LIKE 'a0000000-%';" > /dev/null 2>&1
  docker exec $DB_CONTAINER psql -U $DB_USER -d booking_db -c \
    "DELETE FROM saga_instances WHERE booking_id::text LIKE 'a0000000-%';" > /dev/null 2>&1
  echo "DB cleared"

  # Reset zone available_seats
  echo "[3/4] Resetting zone seats in DB..."
  docker exec $DB_CONTAINER psql -U $DB_USER -d ticket_db -c \
    "UPDATE seat_zones SET available_seats = total_seats WHERE id::text LIKE 'b0000000-%';" > /dev/null 2>&1
  echo "Zones reset"

  # Get admin token for sync (using heredoc to avoid shell escaping issues with !)
  echo "[4/4] Syncing inventory to Redis..."
  ADMIN_TOKEN=$(cat << 'EOF' | curl -s http://localhost:8080/api/v1/auth/login -H "Content-Type: application/json" -d @- | jq -r '.data.access_token'
{"email":"organizer@test.com","password":"Test123!"}
EOF
)

  if [ -z "$ADMIN_TOKEN" ] || [ "$ADMIN_TOKEN" = "null" ]; then
    echo "ERROR: Failed to get admin token"
    return 1
  fi

  SYNC_RESULT=$(curl -s -X POST http://localhost:8080/api/v1/admin/sync-inventory \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json")
  echo "$SYNC_RESULT" | jq -r '"Synced: \(.zones_synced) zones"'

  echo "=== Reset Complete ==="
  echo ""
}

# Function to run test
run_test() {
  local scenario=$1

  echo ""
  # Ask about bypass gateway
  read -p "Bypass Gateway? (y/n) [n]: " bypass_choice
  BYPASS_GATEWAY="false"
  if [ "$bypass_choice" = "y" ] || [ "$bypass_choice" = "Y" ]; then
    BYPASS_GATEWAY="true"
    echo "  → Testing directly to booking:8083 (bypass gateway)"
  else
    echo "  → Testing via gateway:8080"
  fi
  echo ""

  echo "Getting auth token..."
  TOKEN=$(cat << 'EOF' | curl -s "$AUTH_URL" -H "Content-Type: application/json" -d @- | jq -r '.data.access_token'
{"email":"loadtest1@test.com","password":"Test123!"}
EOF
)

  if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "ERROR: Failed to get token"
    return 1
  fi

  echo "Token: ${TOKEN:0:30}..."
  echo ""

  # Create results folder
  mkdir -p results

  # Generate filename with timestamp
  TIMESTAMP=$(date +%Y%m%d-%H%M%S)
  SUFFIX=""
  if [ "$BYPASS_GATEWAY" = "true" ]; then
    SUFFIX="-direct"
  fi
  RESULT_FILE="results/${scenario}${SUFFIX}-${TIMESTAMP}"

  echo "Running scenario: $scenario"
  echo "Bypass Gateway: $BYPASS_GATEWAY"
  echo "Results will be saved to: ${RESULT_FILE}.json"
  echo ""

  K6_WEB_DASHBOARD=true k6 run \
    --env AUTH_TOKEN="$TOKEN" \
    --env SCENARIO="$scenario" \
    --env BYPASS_GATEWAY="$BYPASS_GATEWAY" \
    --out json="${RESULT_FILE}.json" \
    --summary-export="${RESULT_FILE}-summary.json" \
    01-booking-reserve.js

  echo ""
  echo "=== Results saved ==="
  echo "  Full:    ${RESULT_FILE}.json"
  echo "  Summary: ${RESULT_FILE}-summary.json"
}

# Function to run Virtual Queue test
run_queue_test() {
  local scenario=$1

  echo ""
  echo "=== Virtual Queue Load Test ==="
  echo "This test simulates:"
  echo "  - 10,000 concurrent users joining queue"
  echo "  - Queue releases 500 users at a time"
  echo "  - Users with queue pass can book"
  echo ""

  # Check if REQUIRE_QUEUE_PASS is enabled
  echo "Checking REQUIRE_QUEUE_PASS setting..."
  QUEUE_CHECK=$(curl -s http://localhost:8080/health | jq -r '.queue_pass_required // "unknown"')
  echo "  Queue Pass Required: $QUEUE_CHECK"
  echo ""

  # Create results folder
  mkdir -p results

  # Generate filename with timestamp
  TIMESTAMP=$(date +%Y%m%d-%H%M%S)
  RESULT_FILE="results/${scenario}-${TIMESTAMP}"

  echo "Running scenario: $scenario"
  echo "Results will be saved to: ${RESULT_FILE}.json"
  echo ""

  K6_WEB_DASHBOARD=true k6 run \
    --env SCENARIO="$scenario" \
    --out json="${RESULT_FILE}.json" \
    --summary-export="${RESULT_FILE}-summary.json" \
    06-virtual-queue.js

  echo ""
  echo "=== Results saved ==="
  echo "  Full:    ${RESULT_FILE}.json"
  echo "  Summary: ${RESULT_FILE}-summary.json"
  echo ""
  echo "=== Key Metrics to Verify ==="
  echo "  - queue_join_success > 95%"
  echo "  - queue_pass_received > 80%"
  echo "  - booking_success > 90%"
  echo "  - Zero overselling (check DB)"
}

# Function to monitor system during test
monitor_system() {
  echo ""
  echo "=== System Monitor ==="
  echo "Press Ctrl+C to stop"
  echo ""

  read -p "Refresh interval (seconds) [2]: " interval
  interval=${interval:-2}

  while true; do
    clear
    echo "╔══════════════════════════════════════════════════════════════════════════════╗"
    echo "║                    📊 SYSTEM MONITOR - $(date '+%H:%M:%S')                            ║"
    echo "╠══════════════════════════════════════════════════════════════════════════════╣"

    # Docker CPU/Memory
    echo "║ 🐳 DOCKER CONTAINERS (Top CPU)                                               ║"
    echo "╟──────────────────────────────────────────────────────────────────────────────╢"
    printf "║ %-40s %8s %15s ║\n" "CONTAINER" "CPU" "MEMORY"
    echo "╟──────────────────────────────────────────────────────────────────────────────╢"

    docker stats --no-stream --format "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" 2>/dev/null | \
      sort -t$'\t' -k2 -rn 2>/dev/null | head -10 | \
      while IFS=$'\t' read -r name cpu mem; do
        printf "║ %-40s %8s %15s ║\n" "${name:0:40}" "$cpu" "${mem%% *}"
      done

    echo "╠══════════════════════════════════════════════════════════════════════════════╣"

    # Redis Stats
    REDIS_CLIENTS=$(docker exec booking-rush-redis redis-cli -a redis123 --no-auth-warning INFO clients 2>/dev/null | grep connected_clients | cut -d: -f2 | tr -d '\r')
    REDIS_MEM=$(docker exec booking-rush-redis redis-cli -a redis123 --no-auth-warning INFO memory 2>/dev/null | grep used_memory_human | cut -d: -f2 | tr -d '\r')
    REDIS_KEYS=$(docker exec booking-rush-redis redis-cli -a redis123 --no-auth-warning DBSIZE 2>/dev/null | grep -oE '[0-9]+')
    REDIS_OPS=$(docker exec booking-rush-redis redis-cli -a redis123 --no-auth-warning INFO stats 2>/dev/null | grep instantaneous_ops_per_sec | cut -d: -f2 | tr -d '\r')

    echo "║ 🔴 REDIS                                                                      ║"
    echo "╟──────────────────────────────────────────────────────────────────────────────╢"
    printf "║   Connections: %-10s  Memory: %-10s  Keys: %-10s  Ops/s: %-6s ║\n" \
      "${REDIS_CLIENTS:-0}/20000" "${REDIS_MEM:-0}" "${REDIS_KEYS:-0}" "${REDIS_OPS:-0}"

    echo "╠══════════════════════════════════════════════════════════════════════════════╣"

    # PostgreSQL Stats
    PG_CONN=$(docker exec booking-rush-postgres psql -U postgres -t -c "SELECT count(*) FROM pg_stat_activity;" 2>/dev/null | tr -d ' ')
    PG_ACTIVE=$(docker exec booking-rush-postgres psql -U postgres -t -c "SELECT count(*) FROM pg_stat_activity WHERE state='active';" 2>/dev/null | tr -d ' ')
    PG_IDLE=$(docker exec booking-rush-postgres psql -U postgres -t -c "SELECT count(*) FROM pg_stat_activity WHERE state='idle';" 2>/dev/null | tr -d ' ')
    PG_TPS=$(docker exec booking-rush-postgres psql -U postgres -t -c "SELECT sum(xact_commit + xact_rollback) FROM pg_stat_database;" 2>/dev/null | tr -d ' ')

    echo "║ 🐘 POSTGRESQL                                                                 ║"
    echo "╟──────────────────────────────────────────────────────────────────────────────╢"
    printf "║   Connections: %-10s  Active: %-10s  Idle: %-10s  TXN: %-8s ║\n" \
      "${PG_CONN:-0}/1000" "${PG_ACTIVE:-0}" "${PG_IDLE:-0}" "${PG_TPS:-0}"

    echo "╠══════════════════════════════════════════════════════════════════════════════╣"

    # Booking Service Goroutines (check for leaks)
    GOROUTINES=""
    for i in 1 2 3 4 5; do
      G=$(curl -s "http://localhost:908${i}/debug/pprof/goroutine?debug=1" 2>/dev/null | head -1 | grep -oE '[0-9]+' | head -1)
      if [ -n "$G" ]; then
        GOROUTINES="$GOROUTINES booking-$i:$G"
      fi
    done

    echo "║ 🔧 BOOKING SERVICE GOROUTINES                                                 ║"
    echo "╟──────────────────────────────────────────────────────────────────────────────╢"
    printf "║  %s\n" "$GOROUTINES"
    printf "║   (Normal: <100, Warning: >1000, Critical: >10000)                          ║\n"

    echo "╠══════════════════════════════════════════════════════════════════════════════╣"

    # Network I/O
    echo "║ 📡 NETWORK I/O (Top)                                                          ║"
    echo "╟──────────────────────────────────────────────────────────────────────────────╢"
    docker stats --no-stream --format "{{.Name}}\t{{.NetIO}}" 2>/dev/null | \
      grep -E "(nginx|gateway|booking|redis|postgres)" | head -5 | \
      while IFS=$'\t' read -r name netio; do
        printf "║   %-35s %s\n" "${name:0:35}" "$netio"
      done
    printf "║                                                                              ║\n"

    echo "╚══════════════════════════════════════════════════════════════════════════════╝"
    echo ""
    echo "Refreshing every ${interval}s... (Ctrl+C to stop)"

    sleep "$interval"
  done
}

# Function to run SSE Virtual Queue test (reduces polling by 50x)
run_sse_test() {
  local scenario=$1

  echo ""
  echo "=== Virtual Queue SSE Load Test ==="
  echo "This test uses Server-Sent Events (SSE) to reduce polling:"
  echo "  - Polling: 10K users × 2s poll = 5,000 req/sec overhead"
  echo "  - SSE: 10K connections × 1 req = 10K connections (sustained)"
  echo "  - Result: 50x reduction in request overhead"
  echo ""

  # Check if REQUIRE_QUEUE_PASS is enabled
  echo "Checking REQUIRE_QUEUE_PASS setting..."
  QUEUE_CHECK=$(curl -s http://localhost:8080/health | jq -r '.queue_pass_required // "unknown"')
  echo "  Queue Pass Required: $QUEUE_CHECK"
  echo ""

  # Create results folder
  mkdir -p results

  # Generate filename with timestamp
  TIMESTAMP=$(date +%Y%m%d-%H%M%S)
  RESULT_FILE="results/${scenario}-${TIMESTAMP}"

  echo "Running scenario: $scenario"
  echo "Results will be saved to: ${RESULT_FILE}.json"
  echo ""

  K6_WEB_DASHBOARD=true k6 run \
    --env SCENARIO="$scenario" \
    --out json="${RESULT_FILE}.json" \
    --summary-export="${RESULT_FILE}-summary.json" \
    07-virtual-queue-sse.js

  echo ""
  echo "=== Results saved ==="
  echo "  Full:    ${RESULT_FILE}.json"
  echo "  Summary: ${RESULT_FILE}-summary.json"
  echo ""
  echo "=== Key Metrics to Verify ==="
  echo "  - queue_join_success > 95%"
  echo "  - queue_pass_received > 80%"
  echo "  - booking_success > 90%"
  echo "  - sse_connections (total SSE streams)"
  echo "  - sse_errors (should be 0)"
}

# Main menu
echo "=== k6 Load Test Runner ==="
echo ""
echo "Select option:"
echo "  1) smoke       - 1 VU, 30s (quick test)"
echo "  2) ramp_up     - 0→1000 VUs, 9 min"
echo "  3) sustained   - 5000 RPS, 5 min"
echo "  4) spike       - 1k→10k RPS, 3 min"
echo "  5) stress_10k  - 10000 RPS, 5 min"
echo "  6) all         - Run all scenarios (~25 min)"
echo "  ---"
echo "  7) reset       - Reset all (Redis + DB bookings + zones)"
echo "  8) tokens      - Generate JWT tokens (run before test!)"
echo "  9) monitor     - Real-time system monitor (CPU, Memory, Redis, PostgreSQL)"
echo "  ---"
echo "  Virtual Queue (Polling - Legacy):"
echo "  10) vq_smoke   - Virtual Queue: 100 users (quick test)"
echo "  11) vq_10k     - Virtual Queue: 10,000 concurrent users"
echo "  12) vq_15k     - Virtual Queue: 15,000 concurrent users (stress)"
echo "  ---"
echo "  Virtual Queue (SSE - Optimized, 50x less overhead):"
echo "  13) sse_smoke  - SSE: 100 users (quick test)"
echo "  14) sse_1k     - SSE: 1,000 users"
echo "  15) sse_3k     - SSE: 3,000 users"
echo "  16) sse_5k     - SSE: 5,000 users"
echo "  17) sse_10k    - SSE: 10,000 concurrent users"
echo "  0) exit"
echo ""
read -p "Enter choice [0-17]: " choice

case $choice in
  1) reset_data && run_test "smoke" ;;
  2) reset_data && run_test "ramp_up" ;;
  3) reset_data && run_test "sustained" ;;
  4) reset_data && run_test "spike" ;;
  5) reset_data && run_test "stress_10k" ;;
  6) reset_data && run_test "all" ;;
  7) reset_data ;;
  8) generate_tokens ;;
  9) monitor_system ;;
  10) reset_data && run_queue_test "virtual_queue_smoke" ;;
  11) reset_data && run_queue_test "virtual_queue_10k" ;;
  12) reset_data && run_queue_test "virtual_queue_15k" ;;
  13) reset_data && run_sse_test "sse_smoke" ;;
  14) reset_data && run_sse_test "sse_1k" ;;
  15) reset_data && run_sse_test "sse_3k" ;;
  16) reset_data && run_sse_test "sse_5k" ;;
  17) reset_data && run_sse_test "sse_10k" ;;
  0) echo "Bye!"; exit 0 ;;
  *) echo "Invalid choice"; exit 1 ;;
esac
