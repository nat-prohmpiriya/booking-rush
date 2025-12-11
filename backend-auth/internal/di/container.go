package di

import (
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/handler"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/repository"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
)

// Container holds all dependencies for the auth service
type Container struct {
	// Infrastructure
	DB *database.PostgresDB

	// Repositories
	UserRepo    repository.UserRepository
	SessionRepo repository.SessionRepository
	TenantRepo  repository.TenantRepository

	// Services
	AuthService   service.AuthService
	TenantService service.TenantService

	// Handlers
	HealthHandler *handler.HealthHandler
	AuthHandler   *handler.AuthHandler
	TenantHandler *handler.TenantHandler
}

// ContainerConfig contains configuration for building the container
type ContainerConfig struct {
	DB            *database.PostgresDB
	UserRepo      repository.UserRepository
	SessionRepo   repository.SessionRepository
	TenantRepo    repository.TenantRepository
	ServiceConfig *service.AuthServiceConfig
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *ContainerConfig) *Container {
	c := &Container{
		DB:          cfg.DB,
		UserRepo:    cfg.UserRepo,
		SessionRepo: cfg.SessionRepo,
		TenantRepo:  cfg.TenantRepo,
	}

	// Initialize services
	c.AuthService = service.NewAuthService(
		c.UserRepo,
		c.SessionRepo,
		cfg.ServiceConfig,
	)
	c.TenantService = service.NewTenantService(c.TenantRepo)

	// Initialize handlers
	c.HealthHandler = handler.NewHealthHandler(c.DB)
	c.AuthHandler = handler.NewAuthHandler(c.AuthService)
	c.TenantHandler = handler.NewTenantHandler(c.TenantService)

	return c
}
