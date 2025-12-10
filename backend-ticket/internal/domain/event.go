package domain

import "time"

// Event represents an event in the system
type Event struct {
	ID               string     `json:"id"`
	TenantID         string     `json:"tenant_id"`
	OrganizerID      string     `json:"organizer_id"`
	CategoryID       *string    `json:"category_id,omitempty"`
	Name             string     `json:"name"`
	Slug             string     `json:"slug"`
	Description      string     `json:"description"`
	ShortDescription string     `json:"short_description"`
	PosterURL        string     `json:"poster_url"`
	BannerURL        string     `json:"banner_url"`
	Gallery          []string   `json:"gallery"`
	VenueName        string     `json:"venue_name"`
	VenueAddress     string     `json:"venue_address"`
	City             string     `json:"city"`
	Country          string     `json:"country"`
	Latitude         *float64   `json:"latitude,omitempty"`
	Longitude        *float64   `json:"longitude,omitempty"`
	MaxTicketsPerUser int       `json:"max_tickets_per_user"`
	BookingStartAt   *time.Time `json:"booking_start_at,omitempty"`
	BookingEndAt     *time.Time `json:"booking_end_at,omitempty"`
	Status           string     `json:"status"` // draft, published, cancelled, completed
	IsFeatured       bool       `json:"is_featured"`
	IsPublic         bool       `json:"is_public"`
	MetaTitle        string     `json:"meta_title"`
	MetaDescription  string     `json:"meta_description"`
	Settings         string     `json:"settings"` // JSON string
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
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

// Show represents a specific showing/performance of an event
type Show struct {
	ID            string     `json:"id"`
	EventID       string     `json:"event_id"`
	Name          string     `json:"name"`           // e.g. "Evening Show", "Matinee"
	ShowDate      time.Time  `json:"show_date"`      // Date of the show
	StartTime     time.Time  `json:"start_time"`     // Start time (stored as timetz in DB)
	EndTime       time.Time  `json:"end_time"`       // End time (stored as timetz in DB)
	DoorsOpenAt   *time.Time `json:"doors_open_at"`  // When doors open
	Status        string     `json:"status"`         // scheduled, on_sale, sold_out, cancelled, completed
	SaleStartAt   *time.Time `json:"sale_start_at"`  // When ticket sale starts
	SaleEndAt     *time.Time `json:"sale_end_at"`    // When ticket sale ends
	TotalCapacity int        `json:"total_capacity"` // Total capacity
	ReservedCount int        `json:"reserved_count"` // Reserved seats count
	SoldCount     int        `json:"sold_count"`     // Sold seats count
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// ShowStatus constants (matches show_status enum in DB)
const (
	ShowStatusScheduled = "scheduled"
	ShowStatusOnSale    = "on_sale"
	ShowStatusSoldOut   = "sold_out"
	ShowStatusCancelled = "cancelled"
	ShowStatusCompleted = "completed"
)

// ShowZone represents a zone/section for a specific show (maps to seat_zones table)
// This allows different pricing and availability per show
type ShowZone struct {
	ID             string     `json:"id"`
	ShowID         string     `json:"show_id"`
	Name           string     `json:"name"`            // e.g. "VIP", "Standard", "Standing"
	Description    string     `json:"description"`
	Color          string     `json:"color"`           // Color code for UI (e.g. "#FFD700")
	Price          float64    `json:"price"`           // Price per ticket in this zone
	Currency       string     `json:"currency"`        // Currency code (default: THB)
	TotalSeats     int        `json:"total_seats"`     // Total seats in this zone
	AvailableSeats int        `json:"available_seats"` // Seats still available
	ReservedSeats  int        `json:"reserved_seats"`  // Currently reserved seats
	SoldSeats      int        `json:"sold_seats"`      // Already sold seats
	MinPerOrder    int        `json:"min_per_order"`   // Min tickets per order
	MaxPerOrder    int        `json:"max_per_order"`   // Max tickets per order
	IsActive       bool       `json:"is_active"`       // Whether zone is active for sale
	SortOrder      int        `json:"sort_order"`      // Display order
	SaleStartAt    *time.Time `json:"sale_start_at"`   // Zone-specific sale start
	SaleEndAt      *time.Time `json:"sale_end_at"`     // Zone-specific sale end
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
