package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-auth/internal/domain"
)

// PostgresUserRepository implements UserRepository using PostgreSQL
type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgresUserRepository
func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

// Create creates a new user
func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, first_name, role, tenant_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	// Convert empty tenant_id to nil for NULL in database
	var tenantID interface{}
	if user.TenantID != "" {
		tenantID = user.TenantID
	}

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		tenantID,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// GetByID retrieves a user by ID
func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, COALESCE(first_name, '') as first_name, role, COALESCE(tenant_id::text, '') as tenant_id, COALESCE(stripe_customer_id, '') as stripe_customer_id, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.TenantID,
		&user.StripeCustomerID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// GetByEmail retrieves a user by email
func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, COALESCE(first_name, '') as first_name, role, COALESCE(tenant_id::text, '') as tenant_id, COALESCE(stripe_customer_id, '') as stripe_customer_id, is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &domain.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.Role,
		&user.TenantID,
		&user.StripeCustomerID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

// Update updates a user
func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET email = $2, password_hash = $3, first_name = $4, role = $5, tenant_id = $6, stripe_customer_id = $7, is_active = $8, updated_at = $9
		WHERE id = $1
	`
	user.UpdatedAt = time.Now()

	// Convert empty stripe_customer_id to nil for NULL in database
	var stripeCustomerID interface{}
	if user.StripeCustomerID != "" {
		stripeCustomerID = user.StripeCustomerID
	}

	_, err := r.pool.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Role,
		user.TenantID,
		stripeCustomerID,
		user.IsActive,
		user.UpdatedAt,
	)
	return err
}

// Delete deletes a user
func (r *PostgresUserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// ExistsByEmail checks if a user exists with the given email
func (r *PostgresUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}

// UpdateStripeCustomerID updates the Stripe Customer ID for a user
func (r *PostgresUserRepository) UpdateStripeCustomerID(ctx context.Context, userID, stripeCustomerID string) error {
	query := `UPDATE users SET stripe_customer_id = $2, updated_at = $3 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, userID, stripeCustomerID, time.Now())
	return err
}
