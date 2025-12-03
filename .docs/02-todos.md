# Development Roadmap

## Phase 1: Foundation & Infrastructure (The Skeleton)
- [ ] Setup Monorepo structure (Go Workspaces)
- [ ] Setup Docker Compose (Redis, Kafka, Postgres, Zookeeper, Kafka UI)
- [ ] Implement API Gateway (Basic Proxy)
- [ ] Setup Shared Packages (Logger, Response Wrapper, Error Handling)

## Phase 2: The Core (10k RPS Engine)
- [ ] **Booking Service:** Implement Redis Lua Script for Atomic Stock Deduction
- [ ] **Booking Service:** Implement Kafka Producer (Order Created Event)
- [ ] **Load Test 1:** Write k6 script to test "Stock Deduction" endpoint (Target: 10k RPS)

## Phase 3: The Ecosystem (Services Integration)
- [ ] **Auth Service:** Register/Login & JWT Middleware
- [ ] **Ticket Service:** CRUD Events & Redis Caching
- [ ] **Payment Service:** Kafka Consumer -> PostgreSQL Transaction
- [ ] **Data Consistency:** Implement Idempotency (prevent duplicate orders)

## Phase 4: Frontend & Visualization
- [ ] **Frontend:** Landing Page & Event List
- [ ] **Frontend:** Virtual Queue / Waiting Room UI
- [ ] **Dashboard:** Real-time Sales Monitor (WebSocket/SSE)

## Phase 5: Production Ready
- [ ] Deploy to Coolify/VPS
- [ ] Final Load Test (End-to-End)