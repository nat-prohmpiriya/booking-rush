#!/bin/bash
# =============================================================================
# Data Migration Script: Shared DB -> Separate Auth & Ticket Databases
# =============================================================================
# This script migrates data from the shared booking_rush database to
# separate auth_db and ticket_db databases.
#
# Usage:
#   ./scripts/data-migration/migrate_to_separate_dbs.sh
#
# Environment Variables:
#   SOURCE_DB_URL      - Source database URL (default: booking_rush on localhost)
#   AUTH_DB_URL        - Auth database URL (default: auth_db on localhost)
#   TICKET_DB_URL      - Ticket database URL (default: ticket_db on localhost)
#   PG_HOST            - PostgreSQL host (default: localhost)
#   PG_PORT            - PostgreSQL port (default: 5432)
#   PG_USER            - PostgreSQL user (default: postgres)
#   PG_PASSWORD        - PostgreSQL password (default: postgres)
# =============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration (can be overridden by environment variables)
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_USER="${PG_USER:-postgres}"
PG_PASSWORD="${PG_PASSWORD:-postgres}"

SOURCE_DB="${SOURCE_DB:-booking_rush}"
AUTH_DB="${AUTH_DB:-auth_db}"
TICKET_DB="${TICKET_DB:-ticket_db}"

SOURCE_DB_URL="${SOURCE_DB_URL:-postgres://${PG_USER}:${PG_PASSWORD}@${PG_HOST}:${PG_PORT}/${SOURCE_DB}?sslmode=disable}"
AUTH_DB_URL="${AUTH_DB_URL:-postgres://${PG_USER}:${PG_PASSWORD}@${PG_HOST}:${PG_PORT}/${AUTH_DB}?sslmode=disable}"
TICKET_DB_URL="${TICKET_DB_URL:-postgres://${PG_USER}:${PG_PASSWORD}@${PG_HOST}:${PG_PORT}/${TICKET_DB}?sslmode=disable}"

# Temp directory for CSV exports
TMP_DIR="/tmp/booking-rush-migration"
mkdir -p "$TMP_DIR"

echo -e "${GREEN}=== Data Migration: Shared DB -> Separate Databases ===${NC}"
echo ""
echo "Source DB: ${SOURCE_DB} @ ${PG_HOST}:${PG_PORT}"
echo "Auth DB:   ${AUTH_DB} @ ${PG_HOST}:${PG_PORT}"
echo "Ticket DB: ${TICKET_DB} @ ${PG_HOST}:${PG_PORT}"
echo ""

# Helper function to run psql
run_psql() {
    local db=$1
    shift
    PGPASSWORD="${PG_PASSWORD}" psql -h "${PG_HOST}" -p "${PG_PORT}" -U "${PG_USER}" -d "$db" "$@"
}

# Check if source database exists
echo -e "${YELLOW}Checking source database...${NC}"
if ! run_psql "postgres" -c "SELECT 1 FROM pg_database WHERE datname='${SOURCE_DB}'" | grep -q 1; then
    echo -e "${RED}Error: Source database '${SOURCE_DB}' does not exist${NC}"
    exit 1
fi
echo -e "${GREEN}Source database exists${NC}"

# =============================================================================
# Step 1: Create new databases
# =============================================================================
echo ""
echo -e "${GREEN}=== Step 1: Creating new databases ===${NC}"

# Create auth_db if not exists
if run_psql "postgres" -c "SELECT 1 FROM pg_database WHERE datname='${AUTH_DB}'" | grep -q 1; then
    echo -e "${YELLOW}Database '${AUTH_DB}' already exists${NC}"
else
    echo "Creating database '${AUTH_DB}'..."
    run_psql "postgres" -c "CREATE DATABASE ${AUTH_DB};"
    echo -e "${GREEN}Created ${AUTH_DB}${NC}"
fi

# Create ticket_db if not exists
if run_psql "postgres" -c "SELECT 1 FROM pg_database WHERE datname='${TICKET_DB}'" | grep -q 1; then
    echo -e "${YELLOW}Database '${TICKET_DB}' already exists${NC}"
else
    echo "Creating database '${TICKET_DB}'..."
    run_psql "postgres" -c "CREATE DATABASE ${TICKET_DB};"
    echo -e "${GREEN}Created ${TICKET_DB}${NC}"
fi

# =============================================================================
# Step 2: Run migrations on new databases
# =============================================================================
echo ""
echo -e "${GREEN}=== Step 2: Running migrations ===${NC}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "Running Auth DB migrations..."
migrate -path "${PROJECT_ROOT}/scripts/migrations/auth" -database "${AUTH_DB_URL}" up
echo -e "${GREEN}Auth DB migrations completed${NC}"

echo "Running Ticket DB migrations..."
migrate -path "${PROJECT_ROOT}/scripts/migrations/ticket" -database "${TICKET_DB_URL}" up
echo -e "${GREEN}Ticket DB migrations completed${NC}"

# =============================================================================
# Step 3: Export data from source database
# =============================================================================
echo ""
echo -e "${GREEN}=== Step 3: Exporting data from source database ===${NC}"

echo "Exporting tenants..."
run_psql "${SOURCE_DB}" -c "\COPY (SELECT * FROM tenants ORDER BY created_at) TO '${TMP_DIR}/tenants.csv' CSV HEADER"

echo "Exporting users..."
run_psql "${SOURCE_DB}" -c "\COPY (SELECT * FROM users ORDER BY created_at) TO '${TMP_DIR}/users.csv' CSV HEADER"

echo "Exporting sessions..."
run_psql "${SOURCE_DB}" -c "\COPY (SELECT * FROM sessions ORDER BY created_at) TO '${TMP_DIR}/sessions.csv' CSV HEADER"

echo "Exporting categories..."
run_psql "${SOURCE_DB}" -c "\COPY (SELECT * FROM categories ORDER BY created_at) TO '${TMP_DIR}/categories.csv' CSV HEADER"

echo "Exporting events..."
run_psql "${SOURCE_DB}" -c "\COPY (SELECT * FROM events ORDER BY created_at) TO '${TMP_DIR}/events.csv' CSV HEADER"

echo "Exporting shows..."
run_psql "${SOURCE_DB}" -c "\COPY (SELECT * FROM shows ORDER BY created_at) TO '${TMP_DIR}/shows.csv' CSV HEADER"

echo "Exporting seat_zones..."
run_psql "${SOURCE_DB}" -c "\COPY (SELECT * FROM seat_zones ORDER BY created_at) TO '${TMP_DIR}/seat_zones.csv' CSV HEADER"

echo -e "${GREEN}Data export completed${NC}"

# =============================================================================
# Step 4: Import data to Auth DB
# =============================================================================
echo ""
echo -e "${GREEN}=== Step 4: Importing data to Auth DB ===${NC}"

# Disable FK constraints temporarily
run_psql "${AUTH_DB}" -c "SET session_replication_role = replica;"

echo "Importing tenants..."
run_psql "${AUTH_DB}" -c "\COPY tenants FROM '${TMP_DIR}/tenants.csv' CSV HEADER"

echo "Importing users..."
run_psql "${AUTH_DB}" -c "\COPY users FROM '${TMP_DIR}/users.csv' CSV HEADER"

echo "Importing sessions..."
run_psql "${AUTH_DB}" -c "\COPY sessions FROM '${TMP_DIR}/sessions.csv' CSV HEADER"

# Re-enable FK constraints
run_psql "${AUTH_DB}" -c "SET session_replication_role = DEFAULT;"

echo -e "${GREEN}Auth DB import completed${NC}"

# =============================================================================
# Step 5: Import data to Ticket DB
# =============================================================================
echo ""
echo -e "${GREEN}=== Step 5: Importing data to Ticket DB ===${NC}"

# Disable FK constraints temporarily
run_psql "${TICKET_DB}" -c "SET session_replication_role = replica;"

echo "Importing categories..."
run_psql "${TICKET_DB}" -c "\COPY categories FROM '${TMP_DIR}/categories.csv' CSV HEADER"

echo "Importing events..."
run_psql "${TICKET_DB}" -c "\COPY events FROM '${TMP_DIR}/events.csv' CSV HEADER"

echo "Importing shows..."
run_psql "${TICKET_DB}" -c "\COPY shows FROM '${TMP_DIR}/shows.csv' CSV HEADER"

echo "Importing seat_zones..."
run_psql "${TICKET_DB}" -c "\COPY seat_zones FROM '${TMP_DIR}/seat_zones.csv' CSV HEADER"

# Re-enable FK constraints
run_psql "${TICKET_DB}" -c "SET session_replication_role = DEFAULT;"

echo -e "${GREEN}Ticket DB import completed${NC}"

# =============================================================================
# Step 6: Verify data
# =============================================================================
echo ""
echo -e "${GREEN}=== Step 6: Verifying data ===${NC}"

echo ""
echo "Source DB (${SOURCE_DB}) counts:"
run_psql "${SOURCE_DB}" -t -c "
SELECT 'tenants' as table_name, COUNT(*)::text FROM tenants
UNION ALL SELECT 'users', COUNT(*)::text FROM users
UNION ALL SELECT 'sessions', COUNT(*)::text FROM sessions
UNION ALL SELECT 'categories', COUNT(*)::text FROM categories
UNION ALL SELECT 'events', COUNT(*)::text FROM events
UNION ALL SELECT 'shows', COUNT(*)::text FROM shows
UNION ALL SELECT 'seat_zones', COUNT(*)::text FROM seat_zones;
"

echo ""
echo "Auth DB (${AUTH_DB}) counts:"
run_psql "${AUTH_DB}" -t -c "
SELECT 'tenants' as table_name, COUNT(*)::text FROM tenants
UNION ALL SELECT 'users', COUNT(*)::text FROM users
UNION ALL SELECT 'sessions', COUNT(*)::text FROM sessions;
"

echo ""
echo "Ticket DB (${TICKET_DB}) counts:"
run_psql "${TICKET_DB}" -t -c "
SELECT 'categories' as table_name, COUNT(*)::text FROM categories
UNION ALL SELECT 'events', COUNT(*)::text FROM events
UNION ALL SELECT 'shows', COUNT(*)::text FROM shows
UNION ALL SELECT 'seat_zones', COUNT(*)::text FROM seat_zones;
"

# =============================================================================
# Cleanup
# =============================================================================
echo ""
echo -e "${GREEN}=== Cleanup ===${NC}"
rm -rf "${TMP_DIR}"
echo "Temporary files cleaned up"

# =============================================================================
# Done
# =============================================================================
echo ""
echo -e "${GREEN}=== Data Migration Complete ===${NC}"
echo ""
echo "Next steps:"
echo "1. Update .env with AUTH_DATABASE_* and TICKET_DATABASE_* variables"
echo "2. Update services to use new database configs"
echo "3. Test services with new databases"
echo "4. (Optional) Clean up old tables from shared database after verification"
echo ""
