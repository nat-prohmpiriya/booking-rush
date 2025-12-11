package domain

import (
	"time"
)

// Tenant represents an event organizer entity in the multi-tenant system
type Tenant struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Slug      string                 `json:"slug"`
	Domain    string                 `json:"domain,omitempty"`
	LogoURL   string                 `json:"logo_url,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	IsActive  bool                   `json:"is_active"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	DeletedAt *time.Time             `json:"deleted_at,omitempty"` // Soft delete support
}
