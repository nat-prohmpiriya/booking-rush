package dto

import "time"

// CreateEventRequest represents the request to create a new event
type CreateEventRequest struct {
	Name        string    `json:"name" binding:"required,min=1,max=200"`
	Description string    `json:"description" binding:"max=2000"`
	VenueID     string    `json:"venue_id" binding:"required"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required"`
	TenantID    string    `json:"-"` // Set from context
}

// Validate validates the CreateEventRequest
func (r *CreateEventRequest) Validate() (bool, string) {
	if r.Name == "" {
		return false, "Event name is required"
	}
	if r.VenueID == "" {
		return false, "Venue ID is required"
	}
	if r.StartTime.IsZero() {
		return false, "Start time is required"
	}
	if r.EndTime.IsZero() {
		return false, "End time is required"
	}
	if r.EndTime.Before(r.StartTime) {
		return false, "End time must be after start time"
	}
	if r.StartTime.Before(time.Now()) {
		return false, "Start time must be in the future"
	}
	return true, ""
}

// UpdateEventRequest represents the request to update an event
type UpdateEventRequest struct {
	Name        string    `json:"name" binding:"omitempty,min=1,max=200"`
	Description string    `json:"description" binding:"max=2000"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

// Validate validates the UpdateEventRequest
func (r *UpdateEventRequest) Validate() (bool, string) {
	if r.Name == "" && r.Description == "" && r.StartTime.IsZero() && r.EndTime.IsZero() {
		return false, "At least one field must be provided for update"
	}
	if !r.StartTime.IsZero() && !r.EndTime.IsZero() && r.EndTime.Before(r.StartTime) {
		return false, "End time must be after start time"
	}
	return true, ""
}

// EventResponse represents the response for an event
type EventResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	VenueID     string `json:"venue_id"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Status      string `json:"status"`
	TenantID    string `json:"tenant_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// EventListResponse represents a list of events
type EventListResponse struct {
	Events []*EventResponse `json:"events"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

// EventListFilter represents filters for listing events
type EventListFilter struct {
	Status   string `form:"status"`
	TenantID string `form:"-"`
	VenueID  string `form:"venue_id"`
	Search   string `form:"search"`
	Limit    int    `form:"limit"`
	Offset   int    `form:"offset"`
}

// SetDefaults sets default values for pagination
func (f *EventListFilter) SetDefaults() {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}
