package saga

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// BookingState represents the state of a booking saga
type BookingState string

const (
	StateCreated   BookingState = "CREATED"
	StateReserved  BookingState = "RESERVED"
	StatePaid      BookingState = "PAID"
	StateConfirmed BookingState = "CONFIRMED"
	StateFailed    BookingState = "FAILED"
	StateCancelled BookingState = "CANCELLED"
)

var (
	// ErrInvalidStateTransition is returned when a state transition is not allowed
	ErrInvalidStateTransition = errors.New("invalid state transition")
	// ErrStateNotFound is returned when a saga state is not found
	ErrStateNotFound = errors.New("saga state not found")
)

// validTransitions defines allowed state transitions
// Key is current state, value is list of allowed next states
var validTransitions = map[BookingState][]BookingState{
	StateCreated:   {StateReserved, StateFailed, StateCancelled},
	StateReserved:  {StatePaid, StateFailed, StateCancelled},
	StatePaid:      {StateConfirmed, StateFailed},
	StateConfirmed: {}, // Terminal state
	StateFailed:    {}, // Terminal state
	StateCancelled: {}, // Terminal state
}

// IsTerminal returns true if the state is a terminal state
func (s BookingState) IsTerminal() bool {
	return s == StateConfirmed || s == StateFailed || s == StateCancelled
}

// IsValid returns true if the state is a valid booking state
func (s BookingState) IsValid() bool {
	_, exists := validTransitions[s]
	return exists
}

// CanTransitionTo returns true if transition to the target state is allowed
func (s BookingState) CanTransitionTo(target BookingState) bool {
	allowedStates, exists := validTransitions[s]
	if !exists {
		return false
	}
	for _, allowed := range allowedStates {
		if allowed == target {
			return true
		}
	}
	return false
}

// BookingSaga represents a booking saga instance with state machine
type BookingSaga struct {
	ID             string                 `json:"id"`
	BookingID      string                 `json:"booking_id"`
	EventID        string                 `json:"event_id"`
	UserID         string                 `json:"user_id"`
	State          BookingState           `json:"state"`
	PreviousState  BookingState           `json:"previous_state,omitempty"`
	Data           map[string]interface{} `json:"data"`
	ReservationID  string                 `json:"reservation_id,omitempty"`
	PaymentID      string                 `json:"payment_id,omitempty"`
	ConfirmationID string                 `json:"confirmation_id,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	RetryCount     int                    `json:"retry_count"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
}

// StateTransition represents a state transition record
type StateTransition struct {
	ID        string       `json:"id"`
	SagaID    string       `json:"saga_id"`
	FromState BookingState `json:"from_state"`
	ToState   BookingState `json:"to_state"`
	Reason    string       `json:"reason,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
}

// StateMachine manages state transitions for booking sagas
type StateMachine struct {
	store       StateStore
	transitions []StateTransition
}

// StateStore interface for persisting saga states
type StateStore interface {
	// SaveSaga persists a new saga
	SaveSaga(ctx context.Context, saga *BookingSaga) error
	// GetSaga retrieves a saga by ID
	GetSaga(ctx context.Context, id string) (*BookingSaga, error)
	// GetSagaByBookingID retrieves a saga by booking ID
	GetSagaByBookingID(ctx context.Context, bookingID string) (*BookingSaga, error)
	// UpdateSaga updates an existing saga
	UpdateSaga(ctx context.Context, saga *BookingSaga) error
	// SaveTransition persists a state transition
	SaveTransition(ctx context.Context, transition *StateTransition) error
	// GetTransitions retrieves all transitions for a saga
	GetTransitions(ctx context.Context, sagaID string) ([]StateTransition, error)
	// GetSagasByState retrieves sagas by state
	GetSagasByState(ctx context.Context, state BookingState, limit int) ([]*BookingSaga, error)
}

// NewStateMachine creates a new state machine
func NewStateMachine(store StateStore) *StateMachine {
	return &StateMachine{
		store:       store,
		transitions: make([]StateTransition, 0),
	}
}

// CreateSaga creates a new booking saga in CREATED state
func (sm *StateMachine) CreateSaga(ctx context.Context, bookingID, eventID, userID string, data map[string]interface{}) (*BookingSaga, error) {
	now := time.Now()
	if data == nil {
		data = make(map[string]interface{})
	}

	saga := &BookingSaga{
		ID:        generateID(),
		BookingID: bookingID,
		EventID:   eventID,
		UserID:    userID,
		State:     StateCreated,
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := sm.store.SaveSaga(ctx, saga); err != nil {
		return nil, fmt.Errorf("failed to save saga: %w", err)
	}

	return saga, nil
}

// TransitionTo transitions the saga to a new state
func (sm *StateMachine) TransitionTo(ctx context.Context, sagaID string, newState BookingState, reason string) (*BookingSaga, error) {
	saga, err := sm.store.GetSaga(ctx, sagaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get saga: %w", err)
	}

	// Validate transition
	if !saga.State.CanTransitionTo(newState) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, saga.State, newState)
	}

	// Record transition
	transition := &StateTransition{
		ID:        generateID(),
		SagaID:    sagaID,
		FromState: saga.State,
		ToState:   newState,
		Reason:    reason,
		Timestamp: time.Now(),
	}

	if err := sm.store.SaveTransition(ctx, transition); err != nil {
		return nil, fmt.Errorf("failed to save transition: %w", err)
	}

	// Update saga state
	saga.PreviousState = saga.State
	saga.State = newState
	saga.UpdatedAt = time.Now()

	// Mark completion if terminal state
	if newState.IsTerminal() {
		now := time.Now()
		saga.CompletedAt = &now
	}

	if err := sm.store.UpdateSaga(ctx, saga); err != nil {
		return nil, fmt.Errorf("failed to update saga: %w", err)
	}

	return saga, nil
}

// MarkReserved transitions saga to RESERVED state with reservation details
func (sm *StateMachine) MarkReserved(ctx context.Context, sagaID, reservationID string) (*BookingSaga, error) {
	saga, err := sm.TransitionTo(ctx, sagaID, StateReserved, "Seats reserved successfully")
	if err != nil {
		return nil, err
	}

	saga.ReservationID = reservationID
	if err := sm.store.UpdateSaga(ctx, saga); err != nil {
		return nil, fmt.Errorf("failed to update reservation ID: %w", err)
	}

	return saga, nil
}

// MarkPaid transitions saga to PAID state with payment details
func (sm *StateMachine) MarkPaid(ctx context.Context, sagaID, paymentID string) (*BookingSaga, error) {
	saga, err := sm.TransitionTo(ctx, sagaID, StatePaid, "Payment processed successfully")
	if err != nil {
		return nil, err
	}

	saga.PaymentID = paymentID
	if err := sm.store.UpdateSaga(ctx, saga); err != nil {
		return nil, fmt.Errorf("failed to update payment ID: %w", err)
	}

	return saga, nil
}

// MarkConfirmed transitions saga to CONFIRMED state
func (sm *StateMachine) MarkConfirmed(ctx context.Context, sagaID, confirmationID string) (*BookingSaga, error) {
	saga, err := sm.TransitionTo(ctx, sagaID, StateConfirmed, "Booking confirmed")
	if err != nil {
		return nil, err
	}

	saga.ConfirmationID = confirmationID
	if err := sm.store.UpdateSaga(ctx, saga); err != nil {
		return nil, fmt.Errorf("failed to update confirmation ID: %w", err)
	}

	return saga, nil
}

// MarkFailed transitions saga to FAILED state with error message
func (sm *StateMachine) MarkFailed(ctx context.Context, sagaID, errorMessage string) (*BookingSaga, error) {
	saga, err := sm.store.GetSaga(ctx, sagaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get saga: %w", err)
	}

	// FAILED transition is special - can happen from any non-terminal state
	if saga.State.IsTerminal() {
		return nil, fmt.Errorf("%w: cannot transition from terminal state %s", ErrInvalidStateTransition, saga.State)
	}

	saga, err = sm.TransitionTo(ctx, sagaID, StateFailed, errorMessage)
	if err != nil {
		return nil, err
	}

	saga.ErrorMessage = errorMessage
	saga.RetryCount++
	if err := sm.store.UpdateSaga(ctx, saga); err != nil {
		return nil, fmt.Errorf("failed to update error message: %w", err)
	}

	return saga, nil
}

// MarkCancelled transitions saga to CANCELLED state
func (sm *StateMachine) MarkCancelled(ctx context.Context, sagaID, reason string) (*BookingSaga, error) {
	saga, err := sm.store.GetSaga(ctx, sagaID)
	if err != nil {
		return nil, fmt.Errorf("failed to get saga: %w", err)
	}

	// Can only cancel from CREATED or RESERVED
	if saga.State != StateCreated && saga.State != StateReserved {
		return nil, fmt.Errorf("%w: can only cancel from CREATED or RESERVED state", ErrInvalidStateTransition)
	}

	return sm.TransitionTo(ctx, sagaID, StateCancelled, reason)
}

// GetSaga retrieves a saga by ID
func (sm *StateMachine) GetSaga(ctx context.Context, sagaID string) (*BookingSaga, error) {
	return sm.store.GetSaga(ctx, sagaID)
}

// GetSagaByBookingID retrieves a saga by booking ID
func (sm *StateMachine) GetSagaByBookingID(ctx context.Context, bookingID string) (*BookingSaga, error) {
	return sm.store.GetSagaByBookingID(ctx, bookingID)
}

// GetTransitionHistory retrieves all transitions for a saga
func (sm *StateMachine) GetTransitionHistory(ctx context.Context, sagaID string) ([]StateTransition, error) {
	return sm.store.GetTransitions(ctx, sagaID)
}

// GetPendingSagas retrieves sagas that are not in terminal state
func (sm *StateMachine) GetPendingSagas(ctx context.Context, limit int) ([]*BookingSaga, error) {
	// Get sagas in non-terminal states
	var result []*BookingSaga

	for _, state := range []BookingState{StateCreated, StateReserved, StatePaid} {
		sagas, err := sm.store.GetSagasByState(ctx, state, limit)
		if err != nil {
			return nil, err
		}
		result = append(result, sagas...)
		if limit > 0 && len(result) >= limit {
			return result[:limit], nil
		}
	}

	return result, nil
}

// generateID generates a unique ID using UUID
func generateID() string {
	return uuid.New().String()
}
