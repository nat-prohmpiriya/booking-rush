package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStateStore implements StateStore using PostgreSQL
type PostgresStateStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStateStore creates a new PostgreSQL-based state store
func NewPostgresStateStore(pool *pgxpool.Pool) *PostgresStateStore {
	return &PostgresStateStore{pool: pool}
}

// SaveSaga persists a new saga instance
func (s *PostgresStateStore) SaveSaga(ctx context.Context, saga *BookingSaga) error {
	dataJSON, err := json.Marshal(saga.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal saga data: %w", err)
	}

	query := `
		INSERT INTO saga_instances (
			id, booking_id, event_id, user_id, state, previous_state,
			data, reservation_id, payment_id, confirmation_id,
			error_message, retry_count, created_at, updated_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	var previousState *string
	if saga.PreviousState != "" {
		ps := string(saga.PreviousState)
		previousState = &ps
	}

	var reservationID, paymentID, confirmationID, errorMessage *string
	if saga.ReservationID != "" {
		reservationID = &saga.ReservationID
	}
	if saga.PaymentID != "" {
		paymentID = &saga.PaymentID
	}
	if saga.ConfirmationID != "" {
		confirmationID = &saga.ConfirmationID
	}
	if saga.ErrorMessage != "" {
		errorMessage = &saga.ErrorMessage
	}

	_, err = s.pool.Exec(ctx, query,
		saga.ID,
		saga.BookingID,
		saga.EventID,
		saga.UserID,
		string(saga.State),
		previousState,
		dataJSON,
		reservationID,
		paymentID,
		confirmationID,
		errorMessage,
		saga.RetryCount,
		saga.CreatedAt,
		saga.UpdatedAt,
		saga.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save saga: %w", err)
	}

	return nil
}

// GetSaga retrieves a saga by ID
func (s *PostgresStateStore) GetSaga(ctx context.Context, id string) (*BookingSaga, error) {
	query := `
		SELECT id, booking_id, event_id, user_id, state, previous_state,
			   data, reservation_id, payment_id, confirmation_id,
			   error_message, retry_count, created_at, updated_at, completed_at
		FROM saga_instances
		WHERE id = $1
	`

	return s.scanSaga(ctx, s.pool.QueryRow(ctx, query, id))
}

// GetSagaByBookingID retrieves a saga by booking ID
func (s *PostgresStateStore) GetSagaByBookingID(ctx context.Context, bookingID string) (*BookingSaga, error) {
	query := `
		SELECT id, booking_id, event_id, user_id, state, previous_state,
			   data, reservation_id, payment_id, confirmation_id,
			   error_message, retry_count, created_at, updated_at, completed_at
		FROM saga_instances
		WHERE booking_id = $1
	`

	return s.scanSaga(ctx, s.pool.QueryRow(ctx, query, bookingID))
}

// scanSaga scans a row into a BookingSaga
func (s *PostgresStateStore) scanSaga(ctx context.Context, row pgx.Row) (*BookingSaga, error) {
	var saga BookingSaga
	var state, previousState *string
	var dataJSON []byte
	var reservationID, paymentID, confirmationID, errorMessage *string

	err := row.Scan(
		&saga.ID,
		&saga.BookingID,
		&saga.EventID,
		&saga.UserID,
		&state,
		&previousState,
		&dataJSON,
		&reservationID,
		&paymentID,
		&confirmationID,
		&errorMessage,
		&saga.RetryCount,
		&saga.CreatedAt,
		&saga.UpdatedAt,
		&saga.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrStateNotFound
		}
		return nil, fmt.Errorf("failed to scan saga: %w", err)
	}

	if state != nil {
		saga.State = BookingState(*state)
	}
	if previousState != nil {
		saga.PreviousState = BookingState(*previousState)
	}
	if reservationID != nil {
		saga.ReservationID = *reservationID
	}
	if paymentID != nil {
		saga.PaymentID = *paymentID
	}
	if confirmationID != nil {
		saga.ConfirmationID = *confirmationID
	}
	if errorMessage != nil {
		saga.ErrorMessage = *errorMessage
	}

	if len(dataJSON) > 0 {
		if err := json.Unmarshal(dataJSON, &saga.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal saga data: %w", err)
		}
	} else {
		saga.Data = make(map[string]interface{})
	}

	return &saga, nil
}

// UpdateSaga updates an existing saga instance
func (s *PostgresStateStore) UpdateSaga(ctx context.Context, saga *BookingSaga) error {
	dataJSON, err := json.Marshal(saga.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal saga data: %w", err)
	}

	query := `
		UPDATE saga_instances
		SET state = $2,
			previous_state = $3,
			data = $4,
			reservation_id = $5,
			payment_id = $6,
			confirmation_id = $7,
			error_message = $8,
			retry_count = $9,
			updated_at = $10,
			completed_at = $11
		WHERE id = $1
	`

	var previousState *string
	if saga.PreviousState != "" {
		ps := string(saga.PreviousState)
		previousState = &ps
	}

	var reservationID, paymentID, confirmationID, errorMessage *string
	if saga.ReservationID != "" {
		reservationID = &saga.ReservationID
	}
	if saga.PaymentID != "" {
		paymentID = &saga.PaymentID
	}
	if saga.ConfirmationID != "" {
		confirmationID = &saga.ConfirmationID
	}
	if saga.ErrorMessage != "" {
		errorMessage = &saga.ErrorMessage
	}

	result, err := s.pool.Exec(ctx, query,
		saga.ID,
		string(saga.State),
		previousState,
		dataJSON,
		reservationID,
		paymentID,
		confirmationID,
		errorMessage,
		saga.RetryCount,
		time.Now(),
		saga.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update saga: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrStateNotFound
	}

	return nil
}

// SaveTransition persists a state transition
func (s *PostgresStateStore) SaveTransition(ctx context.Context, transition *StateTransition) error {
	query := `
		INSERT INTO saga_transitions (id, saga_id, from_state, to_state, reason, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	var reason *string
	if transition.Reason != "" {
		reason = &transition.Reason
	}

	_, err := s.pool.Exec(ctx, query,
		transition.ID,
		transition.SagaID,
		string(transition.FromState),
		string(transition.ToState),
		reason,
		transition.Timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to save transition: %w", err)
	}

	return nil
}

// GetTransitions retrieves all transitions for a saga
func (s *PostgresStateStore) GetTransitions(ctx context.Context, sagaID string) ([]StateTransition, error) {
	query := `
		SELECT id, saga_id, from_state, to_state, reason, timestamp
		FROM saga_transitions
		WHERE saga_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := s.pool.Query(ctx, query, sagaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get transitions: %w", err)
	}
	defer rows.Close()

	var transitions []StateTransition
	for rows.Next() {
		var t StateTransition
		var fromState, toState string
		var reason *string

		if err := rows.Scan(&t.ID, &t.SagaID, &fromState, &toState, &reason, &t.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan transition: %w", err)
		}

		t.FromState = BookingState(fromState)
		t.ToState = BookingState(toState)
		if reason != nil {
			t.Reason = *reason
		}

		transitions = append(transitions, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transitions: %w", err)
	}

	return transitions, nil
}

// GetSagasByState retrieves sagas by state
func (s *PostgresStateStore) GetSagasByState(ctx context.Context, state BookingState, limit int) ([]*BookingSaga, error) {
	query := `
		SELECT id, booking_id, event_id, user_id, state, previous_state,
			   data, reservation_id, payment_id, confirmation_id,
			   error_message, retry_count, created_at, updated_at, completed_at
		FROM saga_instances
		WHERE state = $1
		ORDER BY created_at ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.pool.Query(ctx, query, string(state))
	if err != nil {
		return nil, fmt.Errorf("failed to get sagas by state: %w", err)
	}
	defer rows.Close()

	var sagas []*BookingSaga
	for rows.Next() {
		var saga BookingSaga
		var stateStr, previousState *string
		var dataJSON []byte
		var reservationID, paymentID, confirmationID, errorMessage *string

		err := rows.Scan(
			&saga.ID,
			&saga.BookingID,
			&saga.EventID,
			&saga.UserID,
			&stateStr,
			&previousState,
			&dataJSON,
			&reservationID,
			&paymentID,
			&confirmationID,
			&errorMessage,
			&saga.RetryCount,
			&saga.CreatedAt,
			&saga.UpdatedAt,
			&saga.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan saga: %w", err)
		}

		if stateStr != nil {
			saga.State = BookingState(*stateStr)
		}
		if previousState != nil {
			saga.PreviousState = BookingState(*previousState)
		}
		if reservationID != nil {
			saga.ReservationID = *reservationID
		}
		if paymentID != nil {
			saga.PaymentID = *paymentID
		}
		if confirmationID != nil {
			saga.ConfirmationID = *confirmationID
		}
		if errorMessage != nil {
			saga.ErrorMessage = *errorMessage
		}

		if len(dataJSON) > 0 {
			if err := json.Unmarshal(dataJSON, &saga.Data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal saga data: %w", err)
			}
		} else {
			saga.Data = make(map[string]interface{})
		}

		sagas = append(sagas, &saga)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sagas: %w", err)
	}

	return sagas, nil
}
