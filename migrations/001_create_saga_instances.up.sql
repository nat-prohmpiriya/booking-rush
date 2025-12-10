-- Migration: Create saga_instances and saga_transitions tables
-- Version: 001
-- Description: Tables for Saga state machine persistence

-- Create enum type for booking states
CREATE TYPE booking_state AS ENUM (
    'CREATED',
    'RESERVED',
    'PAID',
    'CONFIRMED',
    'FAILED',
    'CANCELLED'
);

-- Create saga_instances table
CREATE TABLE IF NOT EXISTS saga_instances (
    id VARCHAR(64) PRIMARY KEY,
    booking_id VARCHAR(64) NOT NULL UNIQUE,
    event_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    state booking_state NOT NULL DEFAULT 'CREATED',
    previous_state booking_state,
    data JSONB DEFAULT '{}',
    reservation_id VARCHAR(64),
    payment_id VARCHAR(64),
    confirmation_id VARCHAR(64),
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for common queries
CREATE INDEX idx_saga_instances_booking_id ON saga_instances(booking_id);
CREATE INDEX idx_saga_instances_event_id ON saga_instances(event_id);
CREATE INDEX idx_saga_instances_user_id ON saga_instances(user_id);
CREATE INDEX idx_saga_instances_state ON saga_instances(state);
CREATE INDEX idx_saga_instances_created_at ON saga_instances(created_at);
CREATE INDEX idx_saga_instances_state_updated ON saga_instances(state, updated_at)
    WHERE state NOT IN ('CONFIRMED', 'FAILED', 'CANCELLED');

-- Create saga_transitions table for audit trail
CREATE TABLE IF NOT EXISTS saga_transitions (
    id VARCHAR(64) PRIMARY KEY,
    saga_id VARCHAR(64) NOT NULL REFERENCES saga_instances(id) ON DELETE CASCADE,
    from_state booking_state NOT NULL,
    to_state booking_state NOT NULL,
    reason TEXT,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for transitions
CREATE INDEX idx_saga_transitions_saga_id ON saga_transitions(saga_id);
CREATE INDEX idx_saga_transitions_timestamp ON saga_transitions(timestamp);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_saga_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_saga_updated_at
    BEFORE UPDATE ON saga_instances
    FOR EACH ROW
    EXECUTE FUNCTION update_saga_updated_at();

-- Add comments for documentation
COMMENT ON TABLE saga_instances IS 'Stores booking saga instances with state machine';
COMMENT ON TABLE saga_transitions IS 'Audit trail for saga state transitions';
COMMENT ON COLUMN saga_instances.state IS 'Current state: CREATED -> RESERVED -> PAID -> CONFIRMED or FAILED/CANCELLED';
COMMENT ON COLUMN saga_instances.data IS 'Additional saga data as JSON';
