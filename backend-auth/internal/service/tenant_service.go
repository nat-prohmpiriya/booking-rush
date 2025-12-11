package service

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/repository"
)

var (
	ErrTenantAlreadyExists = errors.New("tenant with this slug already exists")
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrInvalidSlug         = errors.New("invalid slug format")
)

// TenantService defines the interface for tenant management operations
type TenantService interface {
	// Create creates a new tenant (organizer)
	Create(ctx context.Context, req *dto.CreateTenantRequest) (*dto.TenantResponse, error)
	// GetByID retrieves a tenant by ID
	GetByID(ctx context.Context, id string) (*dto.TenantResponse, error)
	// GetBySlug retrieves a tenant by slug
	GetBySlug(ctx context.Context, slug string) (*dto.TenantResponse, error)
	// List retrieves tenants with pagination and filters
	List(ctx context.Context, query *dto.ListTenantsQuery) (*dto.ListTenantsResponse, error)
	// Update updates a tenant
	Update(ctx context.Context, id string, req *dto.UpdateTenantRequest) (*dto.TenantResponse, error)
	// Delete soft deletes a tenant
	Delete(ctx context.Context, id string) error
}

// tenantService implements TenantService
type tenantService struct {
	tenantRepo repository.TenantRepository
}

// NewTenantService creates a new TenantService
func NewTenantService(tenantRepo repository.TenantRepository) TenantService {
	return &tenantService{
		tenantRepo: tenantRepo,
	}
}

// Create creates a new tenant (organizer)
func (s *tenantService) Create(ctx context.Context, req *dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	// Validate slug format
	if valid, errMsg := req.ValidateSlug(); !valid {
		return nil, errors.New(errMsg)
	}

	// Check if tenant with this slug already exists
	exists, err := s.tenantRepo.ExistsBySlug(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrTenantAlreadyExists
	}

	// Create tenant
	now := time.Now()
	tenant := &domain.Tenant{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Slug:      req.Slug,
		Domain:    req.Domain,
		LogoURL:   req.LogoURL,
		Settings:  req.Settings,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Initialize settings if nil
	if tenant.Settings == nil {
		tenant.Settings = make(map[string]interface{})
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return s.toTenantResponse(tenant), nil
}

// GetByID retrieves a tenant by ID
func (s *tenantService) GetByID(ctx context.Context, id string) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	return s.toTenantResponse(tenant), nil
}

// GetBySlug retrieves a tenant by slug
func (s *tenantService) GetBySlug(ctx context.Context, slug string) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}
	return s.toTenantResponse(tenant), nil
}

// List retrieves tenants with pagination and filters
func (s *tenantService) List(ctx context.Context, query *dto.ListTenantsQuery) (*dto.ListTenantsResponse, error) {
	// Set defaults
	query.SetDefaults()

	// Get tenants from repository
	tenants, totalCount, err := s.tenantRepo.List(ctx, query.Page, query.Limit, query.IsActive, query.Search)
	if err != nil {
		return nil, err
	}

	// Convert to response DTOs
	tenantResponses := make([]dto.TenantResponse, 0, len(tenants))
	for _, tenant := range tenants {
		tenantResponses = append(tenantResponses, *s.toTenantResponse(tenant))
	}

	// Calculate total pages
	totalPages := int(math.Ceil(float64(totalCount) / float64(query.Limit)))

	return &dto.ListTenantsResponse{
		Tenants:    tenantResponses,
		TotalCount: totalCount,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages,
	}, nil
}

// Update updates a tenant
func (s *tenantService) Update(ctx context.Context, id string, req *dto.UpdateTenantRequest) (*dto.TenantResponse, error) {
	// Validate that at least one field is provided
	if valid, errMsg := req.Validate(); !valid {
		return nil, errors.New(errMsg)
	}

	// Get existing tenant
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	// Update fields if provided
	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.Domain != nil {
		tenant.Domain = *req.Domain
	}
	if req.LogoURL != nil {
		tenant.LogoURL = *req.LogoURL
	}
	if req.Settings != nil {
		tenant.Settings = *req.Settings
	}
	if req.IsActive != nil {
		tenant.IsActive = *req.IsActive
	}

	// Update in repository
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	return s.toTenantResponse(tenant), nil
}

// Delete soft deletes a tenant
func (s *tenantService) Delete(ctx context.Context, id string) error {
	// Check if tenant exists
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if tenant == nil {
		return ErrTenantNotFound
	}

	// Soft delete tenant
	return s.tenantRepo.SoftDelete(ctx, id)
}

// toTenantResponse converts domain.Tenant to dto.TenantResponse
func (s *tenantService) toTenantResponse(tenant *domain.Tenant) *dto.TenantResponse {
	return &dto.TenantResponse{
		ID:        tenant.ID,
		Name:      tenant.Name,
		Slug:      tenant.Slug,
		Domain:    tenant.Domain,
		LogoURL:   tenant.LogoURL,
		Settings:  tenant.Settings,
		IsActive:  tenant.IsActive,
		CreatedAt: tenant.CreatedAt.Format(time.RFC3339),
		UpdatedAt: tenant.UpdatedAt.Format(time.RFC3339),
	}
}
