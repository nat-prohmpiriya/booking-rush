package repository

import (
	"context"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/domain"
)

// TenantRepository defines the interface for tenant data access
type TenantRepository interface {
	// Create creates a new tenant
	Create(ctx context.Context, tenant *domain.Tenant) error
	// GetByID retrieves a tenant by ID
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)
	// GetBySlug retrieves a tenant by slug
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	// List retrieves tenants with pagination and filters
	List(ctx context.Context, page, limit int, isActive *bool, search string) ([]*domain.Tenant, int, error)
	// Update updates a tenant
	Update(ctx context.Context, tenant *domain.Tenant) error
	// SoftDelete soft deletes a tenant
	SoftDelete(ctx context.Context, id string) error
	// ExistsBySlug checks if a tenant exists with the given slug
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}
