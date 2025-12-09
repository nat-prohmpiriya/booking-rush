package domain

import "time"

// Event represents an event in the system
type Event struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description"`
	VenueID     string     `json:"venue_id"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     time.Time  `json:"end_time"`
	Status      string     `json:"status"` // draft, published, cancelled, completed
	TenantID    string     `json:"tenant_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// EventStatus constants
const (
	EventStatusDraft     = "draft"
	EventStatusPublished = "published"
	EventStatusCancelled = "cancelled"
	EventStatusCompleted = "completed"
)

// Venue represents a venue where events are held
type Venue struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Capacity  int       `json:"capacity"`
	TenantID  string    `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Zone represents a zone/section within a venue
type Zone struct {
	ID        string    `json:"id"`
	VenueID   string    `json:"venue_id"`
	Name      string    `json:"name"`
	Capacity  int       `json:"capacity"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Seat represents an individual seat in a zone
type Seat struct {
	ID        string    `json:"id"`
	ZoneID    string    `json:"zone_id"`
	Row       string    `json:"row"`
	Number    string    `json:"number"`
	Status    string    `json:"status"` // available, reserved, sold, blocked
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SeatStatus constants
const (
	SeatStatusAvailable = "available"
	SeatStatusReserved  = "reserved"
	SeatStatusSold      = "sold"
	SeatStatusBlocked   = "blocked"
)

// TicketType represents a type of ticket for an event
type TicketType struct {
	ID             string    `json:"id"`
	EventID        string    `json:"event_id"`
	ZoneID         string    `json:"zone_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Price          float64   `json:"price"`
	TotalQuantity  int       `json:"total_quantity"`
	SoldQuantity   int       `json:"sold_quantity"`
	MaxPerBooking  int       `json:"max_per_booking"`
	SaleStartTime  time.Time `json:"sale_start_time"`
	SaleEndTime    time.Time `json:"sale_end_time"`
	Status         string    `json:"status"` // active, sold_out, inactive
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TicketTypeStatus constants
const (
	TicketTypeStatusActive   = "active"
	TicketTypeStatusSoldOut  = "sold_out"
	TicketTypeStatusInactive = "inactive"
)

// AvailableQuantity returns the number of tickets still available
func (t *TicketType) AvailableQuantity() int {
	return t.TotalQuantity - t.SoldQuantity
}

// IsAvailable checks if tickets are available for purchase
func (t *TicketType) IsAvailable() bool {
	now := time.Now()
	return t.Status == TicketTypeStatusActive &&
		t.AvailableQuantity() > 0 &&
		now.After(t.SaleStartTime) &&
		now.Before(t.SaleEndTime)
}
