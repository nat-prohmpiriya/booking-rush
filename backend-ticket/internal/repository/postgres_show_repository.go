package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
)

// showColumns defines the columns to select for shows
// Note: start_time/end_time/doors_open_at are combined with show_date to create full timestamps
const showColumns = `id, event_id, COALESCE(name, '') as name, show_date,
	(show_date + start_time) as start_time,
	(show_date + COALESCE(end_time, start_time)) as end_time,
	CASE WHEN doors_open_at IS NOT NULL THEN (show_date + doors_open_at) ELSE NULL END as doors_open_at,
	status, sale_start_at, sale_end_at,
	COALESCE(total_capacity, 0) as total_capacity,
	COALESCE(reserved_count, 0) as reserved_count,
	COALESCE(sold_count, 0) as sold_count,
	created_at, updated_at, deleted_at`

// PostgresShowRepository implements ShowRepository using PostgreSQL
type PostgresShowRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresShowRepository creates a new PostgresShowRepository
func NewPostgresShowRepository(pool *pgxpool.Pool) *PostgresShowRepository {
	return &PostgresShowRepository{pool: pool}
}

// scanShow scans a row into a Show struct
func (r *PostgresShowRepository) scanShow(row pgx.Row) (*domain.Show, error) {
	show := &domain.Show{}
	err := row.Scan(
		&show.ID,
		&show.EventID,
		&show.Name,
		&show.ShowDate,
		&show.StartTime,
		&show.EndTime,
		&show.DoorsOpenAt,
		&show.Status,
		&show.SaleStartAt,
		&show.SaleEndAt,
		&show.TotalCapacity,
		&show.ReservedCount,
		&show.SoldCount,
		&show.CreatedAt,
		&show.UpdatedAt,
		&show.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return show, nil
}

// Create creates a new show
func (r *PostgresShowRepository) Create(ctx context.Context, show *domain.Show) error {
	query := `
		INSERT INTO shows (id, event_id, name, show_date, start_time, end_time, doors_open_at,
			status, sale_start_at, sale_end_at, total_capacity, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.pool.Exec(ctx, query,
		show.ID,
		show.EventID,
		show.Name,
		show.ShowDate,
		show.StartTime,
		show.EndTime,
		show.DoorsOpenAt,
		show.Status,
		show.SaleStartAt,
		show.SaleEndAt,
		show.TotalCapacity,
		show.CreatedAt,
		show.UpdatedAt,
	)
	return err
}

// GetByID retrieves a show by ID
func (r *PostgresShowRepository) GetByID(ctx context.Context, id string) (*domain.Show, error) {
	query := `SELECT ` + showColumns + ` FROM shows WHERE id = $1 AND deleted_at IS NULL`
	return r.scanShow(r.pool.QueryRow(ctx, query, id))
}

// GetByEventID retrieves shows by event ID with pagination
func (r *PostgresShowRepository) GetByEventID(ctx context.Context, eventID string, limit, offset int) ([]*domain.Show, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM shows WHERE event_id = $1 AND deleted_at IS NULL`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, eventID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get shows
	query := `SELECT ` + showColumns + ` FROM shows
		WHERE event_id = $1 AND deleted_at IS NULL
		ORDER BY show_date ASC, start_time ASC
		LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, eventID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shows []*domain.Show
	for rows.Next() {
		show := &domain.Show{}
		err := rows.Scan(
			&show.ID,
			&show.EventID,
			&show.Name,
			&show.ShowDate,
			&show.StartTime,
			&show.EndTime,
			&show.DoorsOpenAt,
			&show.Status,
			&show.SaleStartAt,
			&show.SaleEndAt,
			&show.TotalCapacity,
			&show.ReservedCount,
			&show.SoldCount,
			&show.CreatedAt,
			&show.UpdatedAt,
			&show.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		shows = append(shows, show)
	}
	return shows, total, nil
}

// Update updates a show
func (r *PostgresShowRepository) Update(ctx context.Context, show *domain.Show) error {
	query := `
		UPDATE shows
		SET name = $2, show_date = $3, start_time = $4, end_time = $5, doors_open_at = $6,
			status = $7, sale_start_at = $8, sale_end_at = $9, updated_at = $10
		WHERE id = $1 AND deleted_at IS NULL
	`
	show.UpdatedAt = time.Now()
	result, err := r.pool.Exec(ctx, query,
		show.ID,
		show.Name,
		show.ShowDate,
		show.StartTime,
		show.EndTime,
		show.DoorsOpenAt,
		show.Status,
		show.SaleStartAt,
		show.SaleEndAt,
		show.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("show not found")
	}
	return nil
}

// Delete soft deletes a show by ID
func (r *PostgresShowRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE shows
		SET deleted_at = $2, updated_at = $2
		WHERE id = $1 AND deleted_at IS NULL
	`
	now := time.Now()
	result, err := r.pool.Exec(ctx, query, id, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("show not found")
	}
	return nil
}
