# Booking Rush 10k RPS - Makefile
# ================================

.PHONY: help dev dev-down build test lint migrate-up migrate-down clean \
	load-seed load-smoke load-ramp load-sustained load-spike load-10k load-full load-clean

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

# Database settings (can be overridden)
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/booking_rush?sslmode=disable
MIGRATIONS_PATH ?= scripts/migrations

# Per-service database URLs
AUTH_DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable
TICKET_DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/ticket_db?sslmode=disable

# Default target
help:
	@echo "$(GREEN)Booking Rush 10k RPS - Available Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)Development:$(NC)"
	@echo "  make dev              - Start infrastructure containers (docker-compose up)"
	@echo "  make dev-down         - Stop infrastructure containers"
	@echo "  make dev-logs         - Follow infrastructure logs"
	@echo ""
	@echo "$(YELLOW)Services:$(NC)"
	@echo "  make run-gateway      - Run API Gateway locally"
	@echo "  make run-auth         - Run Auth Service locally"
	@echo "  make run-booking      - Run Booking Service locally"
	@echo "  make run-ticket       - Run Ticket Service locally"
	@echo "  make run-payment      - Run Payment Service locally"
	@echo ""
	@echo "$(YELLOW)Build:$(NC)"
	@echo "  make build            - Build all Go services"
	@echo "  make build-gateway    - Build API Gateway"
	@echo "  make build-auth       - Build Auth Service"
	@echo "  make build-booking    - Build Booking Service"
	@echo ""
	@echo "$(YELLOW)Database:$(NC)"
	@echo "  make migrate-up       - Run all migrations up"
	@echo "  make migrate-down     - Rollback last migration"
	@echo "  make migrate-down-all - Rollback all migrations"
	@echo "  make migrate-create   - Create new migration (NAME=migration_name)"
	@echo "  make migrate-status   - Show migration status"
	@echo ""
	@echo "$(YELLOW)Testing:$(NC)"
	@echo "  make test             - Run all tests"
	@echo "  make test-unit        - Run unit tests only"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-coverage    - Run tests with coverage"
	@echo ""
	@echo "$(YELLOW)Load Testing:$(NC)"
	@echo "  make load-seed        - Seed test data to PostgreSQL and Redis"
	@echo "  make load-smoke       - Run smoke test (1 VU, 30s)"
	@echo "  make load-ramp        - Run ramp-up test (0→1000 VUs)"
	@echo "  make load-sustained   - Run sustained load test (5000 RPS)"
	@echo "  make load-spike       - Run spike test (1000→10000 RPS)"
	@echo "  make load-10k         - Run 10k RPS stress test"
	@echo "  make load-full        - Run full test suite with dashboard"
	@echo "  make load-clean       - Clean up test data"
	@echo ""
	@echo "$(YELLOW)Code Quality:$(NC)"
	@echo "  make lint             - Run linters"
	@echo "  make fmt              - Format code"
	@echo "  make vet              - Run go vet"
	@echo ""
	@echo "$(YELLOW)Cleanup:$(NC)"
	@echo "  make clean            - Remove build artifacts"

# ================================
# Development
# ================================

dev:
	@echo "$(GREEN)Starting infrastructure...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)Infrastructure started!$(NC)"
	@echo "PostgreSQL: localhost:5432"
	@echo "Redis: localhost:6379"
	@echo "Redpanda: localhost:9092"
	@echo "Redpanda Console: http://localhost:8090"

dev-down:
	@echo "$(YELLOW)Stopping infrastructure...$(NC)"
	docker-compose down
	@echo "$(GREEN)Infrastructure stopped$(NC)"

dev-logs:
	docker-compose logs -f

dev-restart: dev-down dev

# ================================
# Run Services Locally
# ================================

# Load .env file if exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

run-gateway:
	@echo "$(GREEN)Starting API Gateway...$(NC)"
	SERVER_PORT=8080 && cd backend-api-gateway && go run main.go

run-auth:
	@echo "$(GREEN)Starting Auth Service...$(NC)"
	SERVER_PORT=8081 && cd backend-auth && go run main.go

run-booking:
	@echo "$(GREEN)Starting Booking Service...$(NC)"
	SERVER_PORT=8083 && cd backend-booking && go run main.go

run-ticket:
	@echo "$(GREEN)Starting Ticket Service...$(NC)"
	SERVER_PORT=8082 && cd backend-ticket && go run main.go

run-payment:
	@echo "$(GREEN)Starting Payment Service...$(NC)"
	SERVER_PORT=8084 && cd backend-payment && go run main.go

# Stop all backend services
stop-all:
	@echo "$(YELLOW)Stopping all backend services...$(NC)"
	@-pkill -f "backend-api-gateway" 2>/dev/null || true
	@-pkill -f "backend-auth" 2>/dev/null || true
	@-pkill -f "backend-ticket" 2>/dev/null || true
	@-pkill -f "backend-booking" 2>/dev/null || true
	@-pkill -f "backend-payment" 2>/dev/null || true
	@echo "$(GREEN)All services stopped$(NC)"

# Show running services
ps:
	@echo "$(GREEN)Running backend services:$(NC)"
	@lsof -i :8080 -i :8081 -i :8082 -i :8083 -i :8084 2>/dev/null | grep LISTEN || echo "No services running"

# ================================
# Docker Development (with hot reload)
# ================================

docker-dev:
	@echo "$(GREEN)Starting all services (dev mode + hot reload)...$(NC)"
	docker-compose -f docker-compose.services.yml up --build

docker-dev-d:
	@echo "$(GREEN)Starting all services in background (dev mode)...$(NC)"
	docker-compose -f docker-compose.services.yml up --build -d

docker-dev-down:
	@echo "$(YELLOW)Stopping dev services...$(NC)"
	docker-compose -f docker-compose.services.yml down

docker-dev-logs:
	docker-compose -f docker-compose.services.yml logs -f

# ================================
# Docker Production
# ================================

docker-prod:
	@echo "$(GREEN)Starting all services (production mode)...$(NC)"
	docker-compose up --build -d

docker-prod-down:
	@echo "$(YELLOW)Stopping production services...$(NC)"
	docker-compose down

docker-prod-logs:
	docker-compose logs -f

# ================================
# Docker Shortcuts
# ================================

docker-up: docker-dev
docker-down: docker-dev-down
docker-logs: docker-dev-logs
docker-ps:
	@docker-compose -f docker-compose.services.yml ps

# ================================
# Build
# ================================

build: build-gateway build-auth build-booking build-ticket build-payment
	@echo "$(GREEN)All services built successfully!$(NC)"

build-gateway:
	@echo "$(GREEN)Building API Gateway...$(NC)"
	cd backend-api-gateway && go build -o ../../bin/api-gateway .

build-auth:
	@echo "$(GREEN)Building Auth Service...$(NC)"
	cd backend-auth-service && go build -o ../../bin/auth-service . 2>/dev/null || echo "$(YELLOW)Auth Service not ready yet$(NC)"

build-booking:
	@echo "$(GREEN)Building Booking Service...$(NC)"
	cd backend-booking-service && go build -o ../../bin/booking-service . 2>/dev/null || echo "$(YELLOW)Booking Service not ready yet$(NC)"

build-ticket:
	@echo "$(GREEN)Building Ticket Service...$(NC)"
	cd backend-ticket-service && go build -o ../../bin/ticket-service . 2>/dev/null || echo "$(YELLOW)Ticket Service not ready yet$(NC)"

build-payment:
	@echo "$(GREEN)Building Payment Service...$(NC)"
	cd backend-payment-service && go build -o ../../bin/payment-service . 2>/dev/null || echo "$(YELLOW)Payment Service not ready yet$(NC)"

# ================================
# Database Migrations
# ================================

migrate-up:
	@echo "$(GREEN)Running migrations up...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up
	@echo "$(GREEN)Migrations completed$(NC)"

migrate-down:
	@echo "$(YELLOW)Rolling back last migration...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1
	@echo "$(GREEN)Rollback completed$(NC)"

migrate-down-all:
	@echo "$(RED)Rolling back ALL migrations...$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down -all
	@echo "$(GREEN)All migrations rolled back$(NC)"

migrate-status:
	@echo "$(GREEN)Migration status:$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version

migrate-create:
ifndef NAME
	$(error NAME is required. Usage: make migrate-create NAME=create_something)
endif
	@echo "$(GREEN)Creating migration: $(NAME)$(NC)"
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)
	@echo "$(GREEN)Migration files created$(NC)"

migrate-force:
ifndef VERSION
	$(error VERSION is required. Usage: make migrate-force VERSION=1)
endif
	@echo "$(YELLOW)Forcing migration version to: $(VERSION)$(NC)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $(VERSION)

# ================================
# Per-Service Migrations
# ================================

# Auth service migrations
migrate-auth-up:
	@echo "$(GREEN)Running Auth DB migrations up...$(NC)"
	migrate -path scripts/migrations/auth -database "$(AUTH_DATABASE_URL)" up
	@echo "$(GREEN)Auth DB migrations completed$(NC)"

migrate-auth-down:
	@echo "$(YELLOW)Rolling back Auth DB migration...$(NC)"
	migrate -path scripts/migrations/auth -database "$(AUTH_DATABASE_URL)" down 1
	@echo "$(GREEN)Auth DB rollback completed$(NC)"

migrate-auth-status:
	@echo "$(GREEN)Auth DB migration status:$(NC)"
	migrate -path scripts/migrations/auth -database "$(AUTH_DATABASE_URL)" version

migrate-auth-create:
ifndef NAME
	$(error NAME is required. Usage: make migrate-auth-create NAME=create_something)
endif
	@echo "$(GREEN)Creating Auth DB migration: $(NAME)$(NC)"
	migrate create -ext sql -dir scripts/migrations/auth -seq $(NAME)
	@echo "$(GREEN)Auth DB migration files created$(NC)"

# Ticket service migrations
migrate-ticket-up:
	@echo "$(GREEN)Running Ticket DB migrations up...$(NC)"
	migrate -path scripts/migrations/ticket -database "$(TICKET_DATABASE_URL)" up
	@echo "$(GREEN)Ticket DB migrations completed$(NC)"

migrate-ticket-down:
	@echo "$(YELLOW)Rolling back Ticket DB migration...$(NC)"
	migrate -path scripts/migrations/ticket -database "$(TICKET_DATABASE_URL)" down 1
	@echo "$(GREEN)Ticket DB rollback completed$(NC)"

migrate-ticket-status:
	@echo "$(GREEN)Ticket DB migration status:$(NC)"
	migrate -path scripts/migrations/ticket -database "$(TICKET_DATABASE_URL)" version

migrate-ticket-create:
ifndef NAME
	$(error NAME is required. Usage: make migrate-ticket-create NAME=create_something)
endif
	@echo "$(GREEN)Creating Ticket DB migration: $(NAME)$(NC)"
	migrate create -ext sql -dir scripts/migrations/ticket -seq $(NAME)
	@echo "$(GREEN)Ticket DB migration files created$(NC)"

# Run all service migrations
migrate-all-up: migrate-auth-up migrate-ticket-up
	@echo "$(GREEN)All service migrations completed$(NC)"

migrate-all-down:
	@echo "$(YELLOW)Rolling back all service migrations...$(NC)"
	-migrate -path scripts/migrations/ticket -database "$(TICKET_DATABASE_URL)" down 1
	-migrate -path scripts/migrations/auth -database "$(AUTH_DATABASE_URL)" down 1
	@echo "$(GREEN)All service rollbacks completed$(NC)"

migrate-all-status:
	@echo "$(GREEN)Auth DB:$(NC)"
	-migrate -path scripts/migrations/auth -database "$(AUTH_DATABASE_URL)" version
	@echo "$(GREEN)Ticket DB:$(NC)"
	-migrate -path scripts/migrations/ticket -database "$(TICKET_DATABASE_URL)" version

# ================================
# Testing
# ================================

test:
	@echo "$(GREEN)Running all tests...$(NC)"
	go test ./pkg/... ./backend-... -v -race -count=1
	@echo "$(GREEN)All tests passed$(NC)"

test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	go test ./pkg/... ./backend-... -v -short -race
	@echo "$(GREEN)Unit tests passed$(NC)"

test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	INTEGRATION_TEST=true go test ./pkg/... ./backend-... -v -race -run Integration
	@echo "$(GREEN)Integration tests passed$(NC)"

test-coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	go test ./pkg/... ./backend-... -v -race -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

test-bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	go test ./pkg/... ./backend-... -bench=. -benchmem

# ================================
# Load Testing (k6)
# ================================

# Load test settings
LOAD_TEST_DIR := tests/load
LOAD_TEST_SCRIPT := $(LOAD_TEST_DIR)/booking_reserve.js
BASE_URL ?= http://localhost:8083

# Check if k6 is installed
check-k6:
	@which k6 > /dev/null || (echo "$(RED)k6 not installed. Run: brew install k6$(NC)" && exit 1)

# Seed test data to PostgreSQL and Redis
load-seed:
	@echo "$(GREEN)Seeding load test data...$(NC)"
	@chmod +x $(LOAD_TEST_DIR)/seed_all.sh
	@$(LOAD_TEST_DIR)/seed_all.sh
	@echo "$(GREEN)Test data seeded successfully!$(NC)"

# Smoke test - quick validation (1 VU, 30s)
load-smoke: check-k6
	@echo "$(GREEN)Running smoke test...$(NC)"
	k6 run --env BASE_URL=$(BASE_URL) \
		--config /dev/stdin <<< '{"scenarios":{"smoke":{"executor":"constant-vus","vus":1,"duration":"30s"}}}' \
		$(LOAD_TEST_SCRIPT)

# Ramp-up test (0→1000 VUs over 9 minutes)
load-ramp: check-k6
	@echo "$(GREEN)Running ramp-up test...$(NC)"
	K6_WEB_DASHBOARD=true k6 run --env BASE_URL=$(BASE_URL) \
		--tag testid=ramp-$(shell date +%Y%m%d-%H%M%S) \
		-e SCENARIO=ramp_up \
		$(LOAD_TEST_SCRIPT)

# Sustained load test (5000 RPS for 5 minutes)
load-sustained: check-k6
	@echo "$(GREEN)Running sustained load test (5000 RPS)...$(NC)"
	K6_WEB_DASHBOARD=true k6 run --env BASE_URL=$(BASE_URL) \
		--tag testid=sustained-$(shell date +%Y%m%d-%H%M%S) \
		-e SCENARIO=sustained \
		$(LOAD_TEST_SCRIPT)

# Spike test (1000→10000 RPS)
load-spike: check-k6
	@echo "$(GREEN)Running spike test...$(NC)"
	K6_WEB_DASHBOARD=true k6 run --env BASE_URL=$(BASE_URL) \
		--tag testid=spike-$(shell date +%Y%m%d-%H%M%S) \
		-e SCENARIO=spike \
		$(LOAD_TEST_SCRIPT)

# 10k RPS stress test
load-10k: check-k6
	@echo "$(GREEN)Running 10k RPS stress test...$(NC)"
	K6_WEB_DASHBOARD=true k6 run --env BASE_URL=$(BASE_URL) \
		--tag testid=stress10k-$(shell date +%Y%m%d-%H%M%S) \
		-e SCENARIO=stress_10k \
		$(LOAD_TEST_SCRIPT)

# Full test suite with web dashboard
load-full: check-k6
	@echo "$(GREEN)Running full load test suite with dashboard...$(NC)"
	@echo "$(YELLOW)Dashboard available at: http://localhost:5665$(NC)"
	K6_WEB_DASHBOARD=true k6 run --env BASE_URL=$(BASE_URL) \
		--tag testid=full-$(shell date +%Y%m%d-%H%M%S) \
		$(LOAD_TEST_SCRIPT)

# Quick load test (smoke only, no dashboard)
load-quick: check-k6
	@echo "$(GREEN)Running quick load test...$(NC)"
	k6 run --env BASE_URL=$(BASE_URL) --duration 30s --vus 10 $(LOAD_TEST_SCRIPT)

# Clean up test data
load-clean:
	@echo "$(YELLOW)Cleaning up load test data...$(NC)"
	@echo "Cleaning PostgreSQL..."
	@docker run --rm \
		-e PGPASSWORD="$${POSTGRES_PASSWORD}" \
		postgres:15-alpine \
		psql -h $${POSTGRES_HOST:-100.104.0.42} -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-booking_rush} \
		-c "DELETE FROM bookings WHERE user_id LIKE 'load-test-%'; \
		    DELETE FROM seat_zones WHERE id LIKE 'load-test-%'; \
		    DELETE FROM shows WHERE id LIKE 'load-test-%'; \
		    DELETE FROM events WHERE id LIKE 'load-test-%'; \
		    DELETE FROM users WHERE id LIKE 'load-test-%'; \
		    DELETE FROM tenants WHERE id = 'load-test-tenant';"
	@echo "Cleaning Redis..."
	@docker run --rm redis:7-alpine redis-cli -h $${REDIS_HOST:-100.104.0.42} -a "$${REDIS_PASSWORD}" --no-auth-warning \
		KEYS "zone:availability:load-test-*" | xargs -r docker run --rm redis:7-alpine redis-cli -h $${REDIS_HOST:-100.104.0.42} -a "$${REDIS_PASSWORD}" --no-auth-warning DEL || true
	@echo "$(GREEN)Cleanup complete!$(NC)"

# ================================
# Code Quality
# ================================

lint:
	@echo "$(GREEN)Running linters...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(RED)golangci-lint not installed. Run: brew install golangci-lint$(NC)" && exit 1)
	golangci-lint run ./...
	@echo "$(GREEN)Linting passed$(NC)"

fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	gofmt -s -w .
	@echo "$(GREEN)Code formatted$(NC)"

vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	go vet ./...
	@echo "$(GREEN)Vet passed$(NC)"

# ================================
# Go Workspace
# ================================

tidy:
	@echo "$(GREEN)Tidying Go modules...$(NC)"
	cd pkg && go mod tidy
	cd backend-api-gateway && go mod tidy
	cd backend-auth-service && go mod tidy
	cd backend-booking-service && go mod tidy
	cd backend-ticket-service && go mod tidy
	cd backend-payment-service && go mod tidy
	go work sync
	@echo "$(GREEN)Modules tidied$(NC)"

# ================================
# Cleanup
# ================================

clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "$(GREEN)Cleaned$(NC)"

# ================================
# Quick Start
# ================================

setup: dev migrate-up
	@echo "$(GREEN)Setup complete! Run 'make run-gateway' to start the API Gateway$(NC)"

.DEFAULT_GOAL := help
