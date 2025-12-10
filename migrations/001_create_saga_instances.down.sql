-- Migration: Drop saga_instances and saga_transitions tables
-- Version: 001
-- Description: Rollback saga state machine tables

-- Drop trigger first
DROP TRIGGER IF EXISTS trigger_saga_updated_at ON saga_instances;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_saga_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_saga_transitions_timestamp;
DROP INDEX IF EXISTS idx_saga_transitions_saga_id;
DROP INDEX IF EXISTS idx_saga_instances_state_updated;
DROP INDEX IF EXISTS idx_saga_instances_created_at;
DROP INDEX IF EXISTS idx_saga_instances_state;
DROP INDEX IF EXISTS idx_saga_instances_user_id;
DROP INDEX IF EXISTS idx_saga_instances_event_id;
DROP INDEX IF EXISTS idx_saga_instances_booking_id;

-- Drop tables
DROP TABLE IF EXISTS saga_transitions;
DROP TABLE IF EXISTS saga_instances;

-- Drop enum type
DROP TYPE IF EXISTS booking_state;
