-- 000003_create_shows.up.sql
-- Ticket DB: Shows table (specific date/time instances of an event)

CREATE TYPE show_status AS ENUM ('scheduled', 'on_sale', 'sold_out', 'cancelled', 'completed');

CREATE TABLE IF NOT EXISTS shows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,

    -- Timing
    name VARCHAR(255),
    show_date DATE NOT NULL,
    start_time TIME WITH TIME ZONE NOT NULL,
    end_time TIME WITH TIME ZONE,
    doors_open_at TIME WITH TIME ZONE,

    -- Status
    status show_status DEFAULT 'scheduled',

    -- Sales period
    sale_start_at TIMESTAMP WITH TIME ZONE,
    sale_end_at TIMESTAMP WITH TIME ZONE,

    -- Capacity tracking (cached for performance)
    total_capacity INT DEFAULT 0,
    reserved_count INT DEFAULT 0,
    sold_count INT DEFAULT 0,

    -- Settings
    settings JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Indexes
CREATE INDEX idx_shows_event_id ON shows(event_id);
CREATE INDEX idx_shows_show_date ON shows(show_date);
CREATE INDEX idx_shows_status ON shows(status);
CREATE INDEX idx_shows_sale_dates ON shows(sale_start_at, sale_end_at);
CREATE INDEX idx_shows_deleted_at ON shows(deleted_at) WHERE deleted_at IS NULL;

-- Composite index for finding available shows
CREATE INDEX idx_shows_event_available ON shows(event_id, status, show_date)
    WHERE status IN ('scheduled', 'on_sale') AND deleted_at IS NULL;

CREATE TRIGGER update_shows_updated_at
    BEFORE UPDATE ON shows
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
