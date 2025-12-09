package di

import (
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/ticket-service/internal/handler"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/ticket-service/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/apps/ticket-service/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
)

// Container holds all dependencies for the ticket service
type Container struct {
	// Infrastructure
	DB *database.PostgresDB

	// Repositories
	EventRepo repository.EventRepository
	VenueRepo repository.VenueRepository
	// ZoneRepo       repository.ZoneRepository
	// SeatRepo       repository.SeatRepository
	// TicketTypeRepo repository.TicketTypeRepository

	// Services
	EventService service.EventService
	// TicketService service.TicketService
	// VenueService  service.VenueService

	// Handlers
	HealthHandler *handler.HealthHandler
	EventHandler  *handler.EventHandler
	// TicketHandler *handler.TicketHandler
	// VenueHandler  *handler.VenueHandler
}

// ContainerConfig contains configuration for building the container
type ContainerConfig struct {
	DB *database.PostgresDB
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *ContainerConfig) *Container {
	c := &Container{
		DB: cfg.DB,
	}

	// Initialize repositories
	c.EventRepo = repository.NewPostgresEventRepository(c.DB.Pool())
	c.VenueRepo = repository.NewPostgresVenueRepository(c.DB.Pool())
	// c.ZoneRepo = repository.NewPostgresZoneRepository(c.DB.Pool())
	// c.SeatRepo = repository.NewPostgresSeatRepository(c.DB.Pool())
	// c.TicketTypeRepo = repository.NewPostgresTicketTypeRepository(c.DB.Pool())

	// Initialize services
	c.EventService = service.NewEventService(c.EventRepo, c.VenueRepo)
	// c.TicketService = service.NewTicketService(c.TicketTypeRepo, c.EventRepo)
	// c.VenueService = service.NewVenueService(c.VenueRepo, c.ZoneRepo, c.SeatRepo)

	// Initialize handlers
	c.HealthHandler = handler.NewHealthHandler(c.DB)
	c.EventHandler = handler.NewEventHandler(c.EventService)
	// c.TicketHandler = handler.NewTicketHandler(c.TicketService)
	// c.VenueHandler = handler.NewVenueHandler(c.VenueService)

	return c
}
