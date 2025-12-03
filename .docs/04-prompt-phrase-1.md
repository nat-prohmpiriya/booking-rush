# Prompt Phase 1: Project Initialization

**Role:** You are an Expert Go Software Architect and DevOps Engineer specializing in High-Concurrency Systems.

**Context:**
I am building a high-performance ticket booking microservices system named `booking-rush-10k-rps`.
The goal is to handle **10,000 Requests Per Second (RPS)** without overselling.
We will use a Monorepo structure with Go Workspaces.

**Tech Stack:**
- **Language:** Go (Golang)
- **Framework:** Fiber (for performance)
- **Databases:** PostgreSQL, Redis (Alpine)
- **Messaging:** Kafka (Bitnami)
- **Infra:** Docker Compose

**Architecture:**
1. `apps/api-gateway`: Entry point.
2. `apps/auth-service`: JWT handling.
3. `apps/ticket-service`: Manage events/seats.
4. `apps/booking-service`: **Core Service** (Redis Lua + Kafka Producer).
5. `apps/payment-service`: Kafka Consumer.

**Task:**
Please help me set up the **Monorepo Foundation**.

1. **Directory Structure:** Generate the command line instructions (mkdir/touch) to create the folder structure described above (including `pkg/` for shared code).
2. **Go Workspace:** Create the `go.work` file content to manage these services.
3. **Docker Compose:** Write a production-grade `docker-compose.yml` that includes:
   - PostgreSQL (User & DB setup)
   - Redis (Optimized config if possible)
   - Kafka + Zookeeper
   - Kafka UI (for monitoring)
   - Redis Commander (for monitoring)
   - Expose necessary ports.

**Constraints:**
- Use clean architecture.
- Ensure Docker Compose services are networked correctly.
- Do not write the Go application code yet, just the infrastructure and project skeleton.