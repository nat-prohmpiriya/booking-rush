# Database per Service Design

## Overview

เอกสารนี้อธิบายการออกแบบ Database per Service สำหรับ Booking Rush platform ตามหลัก Microservice Architecture

---

## Current State (Shared Database)

```
┌─────────────────────────────────────────────────────────────┐
│                    PostgreSQL (booking_rush)                 │
├─────────────────────────────────────────────────────────────┤
│  tenants ──┬── users ──┬── events ── shows ── seat_zones    │
│            │           │                                     │
│            │           └── bookings ─── payments             │
│            │                                                 │
│            └── categories                                    │
│                                                              │
│  sessions    outbox    audit_logs                           │
└─────────────────────────────────────────────────────────────┘
         ↑           ↑           ↑           ↑
    auth-svc    ticket-svc  booking-svc  payment-svc

    ⚠️ Problem: All services access the SAME database
    ⚠️ Problem: FK constraints create tight coupling
```

---

## Target State (Database per Service)

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  auth_db     │  │  ticket_db   │  │  booking_db  │  │  payment_db  │
├──────────────┤  ├──────────────┤  ├──────────────┤  ├──────────────┤
│ tenants      │  │ categories   │  │ bookings     │  │ payments     │
│ users        │  │ events       │  │ outbox       │  │              │
│ sessions     │  │ shows        │  │              │  │              │
│              │  │ seat_zones   │  │              │  │              │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │                 │
       ▼                 ▼                 ▼                 ▼
  auth-service     ticket-service    booking-service   payment-service
       │                 │                 │                 │
       └─────────────────┴────────┬────────┴─────────────────┘
                                  │
                         ┌────────▼────────┐
                         │  Kafka/Redpanda │
                         │  (Event Bus)    │
                         └─────────────────┘
```

---

## Service Ownership & Database Schema

### 1. Auth Service Database (`auth_db`)

**Owns:** User identity, authentication, multi-tenancy

```sql
-- auth_db

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    domain VARCHAR(255),
    logo_url TEXT,
    settings JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- No FK, just reference
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(200),
    phone VARCHAR(20),
    role VARCHAR(20) DEFAULT 'customer',
    stripe_customer_id VARCHAR(255),
    email_verified BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT unique_email_per_tenant UNIQUE (tenant_id, email)
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,  -- No FK
    refresh_token_hash VARCHAR(255) NOT NULL,
    user_agent TEXT,
    ip_address INET,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at) WHERE expires_at > NOW();
```

**Events Published:**
- `user.created`
- `user.updated`
- `user.deleted`
- `tenant.created`
- `tenant.updated`

---

### 2. Ticket Service Database (`ticket_db`)

**Owns:** Event catalog, shows, pricing, inventory (source of truth for capacity)

```sql
-- ticket_db

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,  -- Reference only
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,
    icon VARCHAR(50),
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT unique_category_slug_per_tenant UNIQUE (tenant_id, slug)
);

CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,      -- Reference to auth_db.tenants
    organizer_id UUID NOT NULL,   -- Reference to auth_db.users
    category_id UUID,             -- Local FK

    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    short_description VARCHAR(500),

    poster_url TEXT,
    banner_url TEXT,
    gallery JSONB DEFAULT '[]',

    venue_name VARCHAR(255),
    venue_address TEXT,
    city VARCHAR(100),
    country VARCHAR(100),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),

    max_tickets_per_user INT DEFAULT 10,
    booking_start_at TIMESTAMPTZ,
    booking_end_at TIMESTAMPTZ,

    status VARCHAR(20) DEFAULT 'draft',
    is_featured BOOLEAN DEFAULT false,
    is_public BOOLEAN DEFAULT true,

    settings JSONB DEFAULT '{}',
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT unique_event_slug_per_tenant UNIQUE (tenant_id, slug),
    CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
);

CREATE TABLE shows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,

    name VARCHAR(255),
    show_date DATE NOT NULL,
    start_time TIMETZ NOT NULL,
    end_time TIMETZ,
    doors_open_at TIMETZ,

    status VARCHAR(20) DEFAULT 'scheduled',
    sale_start_at TIMESTAMPTZ,
    sale_end_at TIMESTAMPTZ,

    -- Denormalized counts (updated via triggers/events)
    total_capacity INT DEFAULT 0,
    reserved_count INT DEFAULT 0,
    sold_count INT DEFAULT 0,

    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE seat_zones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    show_id UUID NOT NULL REFERENCES shows(id) ON DELETE CASCADE,

    name VARCHAR(100) NOT NULL,
    description TEXT,
    color VARCHAR(7),

    price DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'THB',

    -- Inventory (source of truth, synced to Redis)
    total_seats INT NOT NULL,
    available_seats INT NOT NULL,
    reserved_seats INT DEFAULT 0,
    sold_seats INT DEFAULT 0,

    min_per_order INT DEFAULT 1,
    max_per_order INT DEFAULT 10,

    is_active BOOLEAN DEFAULT true,
    sort_order INT DEFAULT 0,
    sale_start_at TIMESTAMPTZ,
    sale_end_at TIMESTAMPTZ,

    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_events_tenant_id ON events(tenant_id);
CREATE INDEX idx_events_organizer_id ON events(organizer_id);
CREATE INDEX idx_events_status ON events(status);
CREATE INDEX idx_events_is_public ON events(is_public) WHERE is_public = true;
CREATE INDEX idx_shows_event_id ON shows(event_id);
CREATE INDEX idx_shows_status ON shows(status);
CREATE INDEX idx_seat_zones_show_id ON seat_zones(show_id);
CREATE INDEX idx_seat_zones_available ON seat_zones(show_id, available_seats)
    WHERE is_active = true AND available_seats > 0;
```

**Events Published:**
- `event.created`
- `event.published`
- `event.cancelled`
- `show.created`
- `zone.inventory.updated`
- `zone.sold_out`

**Events Consumed:**
- `booking.confirmed` → Update sold_seats
- `booking.cancelled` → Release seats
- `booking.expired` → Release seats

---

### 3. Booking Service Database (`booking_db`)

**Owns:** Booking transactions, reservation state

```sql
-- booking_db

CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- External references (no FK, just IDs)
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    event_id UUID NOT NULL,
    show_id UUID NOT NULL,
    zone_id UUID NOT NULL,

    -- Denormalized snapshot at booking time (immutable)
    event_name VARCHAR(255) NOT NULL,
    show_date DATE NOT NULL,
    show_time TIMETZ NOT NULL,
    zone_name VARCHAR(100) NOT NULL,
    venue_name VARCHAR(255),

    -- Booking details
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(12, 2) NOT NULL,
    total_amount DECIMAL(12, 2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'THB',

    -- Status
    status VARCHAR(20) DEFAULT 'reserved',
    status_reason TEXT,

    -- Idempotency & confirmation
    idempotency_key VARCHAR(255) UNIQUE,
    confirmation_code VARCHAR(20),

    -- External payment reference (no FK)
    payment_id UUID,

    -- Timestamps
    reserved_at TIMESTAMPTZ,
    reservation_expires_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancelled_by UUID,

    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Transactional Outbox for event publishing
CREATE TABLE outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(100) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    topic VARCHAR(100) NOT NULL,
    partition_key VARCHAR(255),
    status VARCHAR(20) DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 5,
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    published_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_bookings_tenant_id ON bookings(tenant_id);
CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_bookings_event_id ON bookings(event_id);
CREATE INDEX idx_bookings_show_id ON bookings(show_id);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_idempotency ON bookings(idempotency_key);
CREATE INDEX idx_bookings_user_history ON bookings(user_id, created_at DESC);
CREATE INDEX idx_bookings_pending_expired ON bookings(reservation_expires_at)
    WHERE status = 'reserved';

CREATE INDEX idx_outbox_pending ON outbox(created_at) WHERE status = 'pending';
CREATE INDEX idx_outbox_failed ON outbox(created_at) WHERE status = 'failed' AND retry_count < max_retries;
```

**Events Published:**
- `booking.reserved`
- `booking.confirmed`
- `booking.cancelled`
- `booking.expired`

**Events Consumed:**
- `payment.completed` → Confirm booking
- `payment.failed` → Cancel/Release booking

---

### 4. Payment Service Database (`payment_db`)

**Owns:** Payment transactions, refunds

```sql
-- payment_db

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- External references (no FK)
    booking_id UUID NOT NULL UNIQUE,
    user_id UUID NOT NULL,

    -- Payment details
    amount DECIMAL(12, 2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'THB',
    method VARCHAR(20) NOT NULL,

    -- Status
    status VARCHAR(20) DEFAULT 'pending',

    -- Gateway info
    gateway VARCHAR(50),
    gateway_payment_id VARCHAR(255),
    gateway_charge_id VARCHAR(255),
    gateway_response JSONB,

    -- Idempotency
    idempotency_key VARCHAR(255) UNIQUE,

    -- Card details (masked)
    card_last_four VARCHAR(4),
    card_brand VARCHAR(20),

    -- Timestamps
    initiated_at TIMESTAMPTZ DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    -- Refund
    refund_amount DECIMAL(12, 2),
    refund_reason TEXT,
    refunded_at TIMESTAMPTZ,

    -- Error handling
    error_code VARCHAR(50),
    error_message TEXT,
    retry_count INT DEFAULT 0,

    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Transactional Outbox
CREATE TABLE outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(100) NOT NULL,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    topic VARCHAR(100) NOT NULL,
    partition_key VARCHAR(255),
    status VARCHAR(20) DEFAULT 'pending',
    retry_count INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_payments_booking_id ON payments(booking_id);
CREATE INDEX idx_payments_user_id ON payments(user_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_gateway_id ON payments(gateway_payment_id);
CREATE INDEX idx_payments_pending ON payments(created_at) WHERE status IN ('pending', 'processing');

CREATE INDEX idx_outbox_pending ON outbox(created_at) WHERE status = 'pending';
```

**Events Published:**
- `payment.initiated`
- `payment.processing`
- `payment.completed`
- `payment.failed`
- `payment.refunded`

**Events Consumed:**
- `booking.reserved` → Create pending payment
- `booking.cancelled` → Cancel/Refund payment

---

## Cross-Service Communication

### 1. Synchronous (API Calls)

```
┌─────────────┐         ┌──────────────┐
│   Frontend  │ ──────► │  API Gateway │
└─────────────┘         └──────┬───────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        │                      │                      │
        ▼                      ▼                      ▼
┌───────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Auth Service  │◄───│ Ticket Service  │◄───│ Booking Service │
│               │    │                 │    │                 │
│ GET /users/:id│    │ GET /events/:id │    │ POST /bookings  │
│ (internal)    │    │ GET /zones/:id  │    │                 │
└───────────────┘    └─────────────────┘    └─────────────────┘
```

**When to use:**
- Booking needs event/zone details → Call Ticket Service API
- Payment needs user info → Call Auth Service API (or use JWT claims)

### 2. Asynchronous (Events via Kafka)

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│ Booking Service │     │ Payment Service │     │ Ticket Service  │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         │ booking.reserved      │                       │
         ├──────────────────────►│                       │
         │                       │                       │
         │                       │ payment.completed     │
         │◄──────────────────────┤                       │
         │                       │                       │
         │ booking.confirmed     │                       │
         ├───────────────────────┼──────────────────────►│
         │                       │                       │
         │                       │              (update inventory)
```

---

## Kafka Topics

| Topic | Publisher | Consumers | Purpose |
|-------|-----------|-----------|---------|
| `auth.users` | auth-service | booking, payment, notification | User events |
| `auth.tenants` | auth-service | ticket, booking | Tenant events |
| `ticket.events` | ticket-service | notification, analytics | Event catalog changes |
| `ticket.inventory` | ticket-service | booking | Inventory updates |
| `booking.transactions` | booking-service | payment, ticket, notification | Booking lifecycle |
| `payment.transactions` | payment-service | booking, notification | Payment lifecycle |

---

## Data Denormalization Strategy

### Booking Service - Snapshot Pattern

เมื่อสร้าง booking ให้เก็บ snapshot ของข้อมูลที่จำเป็น:

```go
// When creating a booking, fetch and store snapshot
type BookingSnapshot struct {
    // From Ticket Service (API call at booking time)
    EventName   string
    ShowDate    time.Time
    ShowTime    time.Time
    ZoneName    string
    VenueName   string
    UnitPrice   float64
}

func (s *BookingService) CreateReservation(ctx context.Context, req ReserveRequest) (*Booking, error) {
    // 1. Call Ticket Service to get zone details
    zone, err := s.ticketClient.GetZone(ctx, req.ZoneID)
    if err != nil {
        return nil, err
    }

    show, err := s.ticketClient.GetShow(ctx, req.ShowID)
    if err != nil {
        return nil, err
    }

    event, err := s.ticketClient.GetEvent(ctx, req.EventID)
    if err != nil {
        return nil, err
    }

    // 2. Create booking with snapshot
    booking := &Booking{
        ID:        uuid.New().String(),
        EventID:   req.EventID,
        ShowID:    req.ShowID,
        ZoneID:    req.ZoneID,

        // Snapshot (immutable)
        EventName: event.Name,
        ShowDate:  show.ShowDate,
        ShowTime:  show.StartTime,
        ZoneName:  zone.Name,
        VenueName: event.VenueName,
        UnitPrice: zone.Price,

        Quantity:   req.Quantity,
        TotalAmount: zone.Price * float64(req.Quantity),
        Status:    BookingStatusReserved,
    }

    return booking, nil
}
```

---

## Migration Strategy

### Phase 1: Prepare (No Downtime)

1. สร้าง databases ใหม่แยกตาม service
2. Setup Kafka topics
3. Implement dual-write ใน services

### Phase 2: Migrate Data

```bash
# 1. Create new databases
createdb auth_db
createdb ticket_db
createdb booking_db
createdb payment_db

# 2. Migrate data (use pg_dump/restore or custom scripts)
# Auth data
pg_dump -t tenants -t users -t sessions booking_rush | psql auth_db

# Ticket data
pg_dump -t categories -t events -t shows -t seat_zones booking_rush | psql ticket_db

# Booking data (add denormalized columns first)
# Custom migration script to populate snapshots

# Payment data
pg_dump -t payments booking_rush | psql payment_db
```

### Phase 3: Switch (Short Downtime)

1. Stop all services
2. Run final data sync
3. Update service configurations to new databases
4. Remove FK constraints from old DB
5. Start services with new config
6. Verify data consistency

### Phase 4: Cleanup

1. Monitor for issues
2. Remove dual-write code
3. Drop old shared database

---

## Infrastructure Changes

### Before (Current)

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: booking_rush
```

### After (Per-Service)

```yaml
# docker-compose.yml
services:
  postgres-auth:
    image: postgres:16
    environment:
      POSTGRES_DB: auth_db
    volumes:
      - auth_data:/var/lib/postgresql/data

  postgres-ticket:
    image: postgres:16
    environment:
      POSTGRES_DB: ticket_db
    volumes:
      - ticket_data:/var/lib/postgresql/data

  postgres-booking:
    image: postgres:16
    environment:
      POSTGRES_DB: booking_db
    volumes:
      - booking_data:/var/lib/postgresql/data

  postgres-payment:
    image: postgres:16
    environment:
      POSTGRES_DB: payment_db
    volumes:
      - payment_data:/var/lib/postgresql/data
```

---

## Trade-offs

### Pros
- Independent scaling per service
- Independent deployments
- Technology freedom per service
- Fault isolation (one DB down doesn't affect others)
- Clear data ownership

### Cons
- Eventual consistency (not immediate)
- More complex queries (no joins across services)
- Data duplication (snapshots)
- More infrastructure to manage
- Distributed transactions require Saga pattern

---

## Recommendations

1. **Start with Ticket + Auth split** - ง่ายที่สุด เพราะเป็น read-heavy
2. **Keep Booking + Payment together initially** - จนกว่า event system จะ stable
3. **Implement Outbox Pattern** - สำหรับ reliable event publishing
4. **Add Circuit Breakers** - สำหรับ inter-service calls
5. **Monitor Event Lag** - เพื่อ detect eventual consistency issues
