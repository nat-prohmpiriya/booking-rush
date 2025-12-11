package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/domain"
)

// PostgresTenantRepository implements TenantRepository using PostgreSQL
type PostgresTenantRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresTenantRepository creates a new PostgresTenantRepository
func NewPostgresTenantRepository(pool *pgxpool.Pool) *PostgresTenantRepository {
	return &PostgresTenantRepository{pool: pool}
}

// Create creates a new tenant
func (r *PostgresTenantRepository) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, slug, domain, logo_url, settings, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		nullStringOrValue(tenant.Domain),
		nullStringOrValue(tenant.LogoURL),
		tenant.Settings,
		tenant.IsActive,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)
	return err
}

// GetByID retrieves a tenant by ID
func (r *PostgresTenantRepository) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	query := `
		SELECT id, name, slug, COALESCE(domain, '') as domain, COALESCE(logo_url, '') as logo_url,
		       COALESCE(settings, '{}'::jsonb) as settings, is_active, created_at, updated_at, deleted_at
		FROM tenants
		WHERE id = $1 AND deleted_at IS NULL
	`
	tenant := &domain.Tenant{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.Domain,
		&tenant.LogoURL,
		&tenant.Settings,
		&tenant.IsActive,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return tenant, nil
}

// GetBySlug retrieves a tenant by slug
func (r *PostgresTenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	query := `
		SELECT id, name, slug, COALESCE(domain, '') as domain, COALESCE(logo_url, '') as logo_url,
		       COALESCE(settings, '{}'::jsonb) as settings, is_active, created_at, updated_at, deleted_at
		FROM tenants
		WHERE slug = $1 AND deleted_at IS NULL
	`
	tenant := &domain.Tenant{}
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.Domain,
		&tenant.LogoURL,
		&tenant.Settings,
		&tenant.IsActive,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return tenant, nil
}

// List retrieves tenants with pagination and filters
func (r *PostgresTenantRepository) List(ctx context.Context, page, limit int, isActive *bool, search string) ([]*domain.Tenant, int, error) {
	// Build WHERE clause
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argIndex := 1

	if isActive != nil {
		whereClause += fmt.Sprintf(" AND is_active = $%d", argIndex)
		args = append(args, *isActive)
		argIndex++
	}

	if search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR slug ILIKE $%d)", argIndex, argIndex)
		args = append(args, "%"+search+"%")
		argIndex++
	}

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenants %s", whereClause)
	var totalCount int
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Get paginated records
	offset := (page - 1) * limit
	query := fmt.Sprintf(`
		SELECT id, name, slug, COALESCE(domain, '') as domain, COALESCE(logo_url, '') as logo_url,
		       COALESCE(settings, '{}'::jsonb) as settings, is_active, created_at, updated_at, deleted_at
		FROM tenants
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tenants := make([]*domain.Tenant, 0)
	for rows.Next() {
		tenant := &domain.Tenant{}
		err := rows.Scan(
			&tenant.ID,
			&tenant.Name,
			&tenant.Slug,
			&tenant.Domain,
			&tenant.LogoURL,
			&tenant.Settings,
			&tenant.IsActive,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
			&tenant.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, tenant)
	}

	return tenants, totalCount, nil
}

// Update updates a tenant
func (r *PostgresTenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, domain = $3, logo_url = $4, settings = $5, is_active = $6, updated_at = $7
		WHERE id = $1 AND deleted_at IS NULL
	`
	tenant.UpdatedAt = time.Now()
	result, err := r.pool.Exec(ctx, query,
		tenant.ID,
		tenant.Name,
		nullStringOrValue(tenant.Domain),
		nullStringOrValue(tenant.LogoURL),
		tenant.Settings,
		tenant.IsActive,
		tenant.UpdatedAt,
	)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found or already deleted")
	}

	return nil
}

// SoftDelete soft deletes a tenant by setting deleted_at timestamp
func (r *PostgresTenantRepository) SoftDelete(ctx context.Context, id string) error {
	query := `
		UPDATE tenants
		SET deleted_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	result, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found or already deleted")
	}

	return nil
}

// ExistsBySlug checks if a tenant exists with the given slug
func (r *PostgresTenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, slug).Scan(&exists)
	return exists, err
}

// nullStringOrValue returns nil for empty strings, otherwise returns the value
func nullStringOrValue(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
