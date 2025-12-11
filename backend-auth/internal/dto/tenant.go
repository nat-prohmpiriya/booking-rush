package dto

import (
	"regexp"
)

// CreateTenantRequest represents request to create a new tenant (organizer)
type CreateTenantRequest struct {
	Name     string                 `json:"name" binding:"required,min=2,max=255"`
	Slug     string                 `json:"slug" binding:"required,min=2,max=100"`
	Domain   string                 `json:"domain" binding:"omitempty,max=255"`
	LogoURL  string                 `json:"logo_url" binding:"omitempty,url"`
	Settings map[string]interface{} `json:"settings" binding:"omitempty"`
}

// ValidateSlug validates slug format (lowercase alphanumeric and hyphens only)
func (r *CreateTenantRequest) ValidateSlug() (bool, string) {
	slugRegex := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !slugRegex.MatchString(r.Slug) {
		return false, "Slug must contain only lowercase letters, numbers, and hyphens"
	}
	if len(r.Slug) < 2 {
		return false, "Slug must be at least 2 characters"
	}
	if len(r.Slug) > 100 {
		return false, "Slug must not exceed 100 characters"
	}
	return true, ""
}

// UpdateTenantRequest represents request to update tenant information
type UpdateTenantRequest struct {
	Name     *string                 `json:"name" binding:"omitempty,min=2,max=255"`
	Domain   *string                 `json:"domain" binding:"omitempty,max=255"`
	LogoURL  *string                 `json:"logo_url" binding:"omitempty,url"`
	Settings *map[string]interface{} `json:"settings" binding:"omitempty"`
	IsActive *bool                   `json:"is_active" binding:"omitempty"`
}

// Validate validates that at least one field is provided for update
func (r *UpdateTenantRequest) Validate() (bool, string) {
	if r.Name == nil && r.Domain == nil && r.LogoURL == nil && r.Settings == nil && r.IsActive == nil {
		return false, "At least one field must be provided for update"
	}
	return true, ""
}

// TenantResponse represents tenant data in response
type TenantResponse struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Slug      string                 `json:"slug"`
	Domain    string                 `json:"domain,omitempty"`
	LogoURL   string                 `json:"logo_url,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
	IsActive  bool                   `json:"is_active"`
	CreatedAt string                 `json:"created_at"`
	UpdatedAt string                 `json:"updated_at"`
}

// ListTenantsQuery represents query parameters for listing tenants
type ListTenantsQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	IsActive *bool  `form:"is_active" binding:"omitempty"`
	Search   string `form:"search" binding:"omitempty,max=255"`
}

// SetDefaults sets default values for query parameters
func (q *ListTenantsQuery) SetDefaults() {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
}

// ListTenantsResponse represents paginated list of tenants
type ListTenantsResponse struct {
	Tenants    []TenantResponse `json:"tenants"`
	TotalCount int              `json:"total_count"`
	Page       int              `json:"page"`
	Limit      int              `json:"limit"`
	TotalPages int              `json:"total_pages"`
}
