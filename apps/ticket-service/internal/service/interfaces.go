package service

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/apps/ticket-service/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/ticket-service/internal/dto"
)

// EventService defines the interface for event business logic
type EventService interface {
	// CreateEvent creates a new event
	CreateEvent(ctx context.Context, req *dto.CreateEventRequest) (*domain.Event, error)
	// GetEventByID retrieves an event by ID
	GetEventByID(ctx context.Context, id string) (*domain.Event, error)
	// GetEventBySlug retrieves an event by slug
	GetEventBySlug(ctx context.Context, slug string) (*domain.Event, error)
	// ListEvents lists events with filters and pagination
	ListEvents(ctx context.Context, filter *dto.EventListFilter) ([]*domain.Event, int, error)
	// UpdateEvent updates an event
	UpdateEvent(ctx context.Context, id string, req *dto.UpdateEventRequest) (*domain.Event, error)
	// DeleteEvent soft deletes an event
	DeleteEvent(ctx context.Context, id string) error
	// PublishEvent publishes an event
	PublishEvent(ctx context.Context, id string) (*domain.Event, error)
}

// TicketService defines the interface for ticket business logic
type TicketService interface {
	// CreateTicketType creates a new ticket type for an event
	CreateTicketType(ctx context.Context, req *dto.CreateTicketTypeRequest) (*domain.TicketType, error)
	// GetTicketType retrieves a ticket type by ID
	GetTicketType(ctx context.Context, id string) (*domain.TicketType, error)
	// GetTicketTypesByEvent retrieves ticket types by event ID
	GetTicketTypesByEvent(ctx context.Context, eventID string) ([]*domain.TicketType, error)
	// GetAvailableTicketTypes retrieves available ticket types by event ID
	GetAvailableTicketTypes(ctx context.Context, eventID string) ([]*domain.TicketType, error)
	// UpdateTicketType updates a ticket type
	UpdateTicketType(ctx context.Context, id string, req *dto.UpdateTicketTypeRequest) (*domain.TicketType, error)
	// DeleteTicketType deletes a ticket type
	DeleteTicketType(ctx context.Context, id string) error
	// CheckAvailability checks ticket availability for an event
	CheckAvailability(ctx context.Context, eventID string, ticketTypeID string, quantity int) (*dto.AvailabilityResponse, error)
}

// VenueService defines the interface for venue business logic
type VenueService interface {
	// CreateVenue creates a new venue
	CreateVenue(ctx context.Context, req *dto.CreateVenueRequest) (*domain.Venue, error)
	// GetVenue retrieves a venue by ID
	GetVenue(ctx context.Context, id string) (*domain.Venue, error)
	// GetVenuesByTenant retrieves venues by tenant ID
	GetVenuesByTenant(ctx context.Context, tenantID string) ([]*domain.Venue, error)
	// UpdateVenue updates a venue
	UpdateVenue(ctx context.Context, id string, req *dto.UpdateVenueRequest) (*domain.Venue, error)
	// DeleteVenue deletes a venue
	DeleteVenue(ctx context.Context, id string) error
}
