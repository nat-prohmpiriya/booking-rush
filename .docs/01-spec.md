# Project Specification: Booking Rush (10k RPS)

## 1. Project Overview
ระบบจองตั๋วประสิทธิภาพสูง (High-Concurrency Ticket Booking System) ออกแบบมาเพื่อรองรับ Traffic มหาศาลในระยะเวลาสั้นๆ (Flash Sale)
**Target Goal:** รองรับ 10,000 Requests Per Second (RPS) โดยไม่มีการขายตั๋วเกิน (No Overselling)

## 2. Business Requirements
- **Users:** สามารถดูคอนเสิร์ต, เลือกรอบ, และกดจองบัตรได้
- **Concurrency:** ระบบต้องจัดการ Race Condition เมื่อคนหลายพันคนแย่งกดที่นั่งเดียวกันในระดับ Millisecond
- **Reliability:** ถ้า Payment Service ล่ม การจองต้องไม่หาย (Eventual Consistency)
- **Scale:** ออกแบบเป็น Microservices เพื่อแยก Scale เฉพาะจุด (เช่น Booking Service ต้องรับโหลดหนักสุด)

## 3. Technical Stack
- **Architecture:** Microservices (Monorepo)
- **Backend Language:** Go (Golang) with Gin
- **Frontend:** Next.js 15 (App Router), TailwindCSS, Shadcn UI
- **Database:** PostgreSQL (Partitioned for scale)
- **Caching/Locking:** Redis Cluster + Lua Scripts (Atomic Operations)
- **Message Broker:** Kafka (Buffering High Load)
- **Infrastructure:** Docker, Kubernetes (K8s), Coolify
- **Observability:** Prometheus, Grafana, Jaeger

## 4. Service Boundaries
1. **API Gateway:** Entry point, Rate Limiting, Request Routing.
2. **Auth Service:** User management, JWT issuance.
3. **Ticket Service:** Event catalog, Seat mapping (Read-heavy, Cached).
4. **Booking Service:** **(Core)** Handle reservation, Deduct stock (Redis Lua), Produce Kafka events.
5. **Payment Service:** Consume Kafka, Process payment (Mock), Update DB.