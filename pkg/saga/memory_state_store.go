package saga

import (
	"context"
	"sync"
)

// MemoryStateStore is an in-memory implementation of StateStore for testing
type MemoryStateStore struct {
	mu          sync.RWMutex
	sagas       map[string]*BookingSaga
	transitions map[string][]StateTransition
	byBookingID map[string]string // bookingID -> sagaID
}

// NewMemoryStateStore creates a new in-memory state store
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		sagas:       make(map[string]*BookingSaga),
		transitions: make(map[string][]StateTransition),
		byBookingID: make(map[string]string),
	}
}

// SaveSaga persists a new saga instance
func (s *MemoryStateStore) SaveSaga(ctx context.Context, saga *BookingSaga) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate
	if _, exists := s.sagas[saga.ID]; exists {
		return ErrStateNotFound // Already exists
	}

	// Deep copy
	copied := s.copySaga(saga)
	s.sagas[saga.ID] = copied
	s.byBookingID[saga.BookingID] = saga.ID

	return nil
}

// GetSaga retrieves a saga by ID
func (s *MemoryStateStore) GetSaga(ctx context.Context, id string) (*BookingSaga, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	saga, exists := s.sagas[id]
	if !exists {
		return nil, ErrStateNotFound
	}

	return s.copySaga(saga), nil
}

// GetSagaByBookingID retrieves a saga by booking ID
func (s *MemoryStateStore) GetSagaByBookingID(ctx context.Context, bookingID string) (*BookingSaga, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sagaID, exists := s.byBookingID[bookingID]
	if !exists {
		return nil, ErrStateNotFound
	}

	saga, exists := s.sagas[sagaID]
	if !exists {
		return nil, ErrStateNotFound
	}

	return s.copySaga(saga), nil
}

// UpdateSaga updates an existing saga instance
func (s *MemoryStateStore) UpdateSaga(ctx context.Context, saga *BookingSaga) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sagas[saga.ID]; !exists {
		return ErrStateNotFound
	}

	s.sagas[saga.ID] = s.copySaga(saga)
	return nil
}

// SaveTransition persists a state transition
func (s *MemoryStateStore) SaveTransition(ctx context.Context, transition *StateTransition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.transitions[transition.SagaID] = append(s.transitions[transition.SagaID], *transition)
	return nil
}

// GetTransitions retrieves all transitions for a saga
func (s *MemoryStateStore) GetTransitions(ctx context.Context, sagaID string) ([]StateTransition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	transitions := s.transitions[sagaID]
	if transitions == nil {
		return []StateTransition{}, nil
	}

	// Return a copy
	result := make([]StateTransition, len(transitions))
	copy(result, transitions)
	return result, nil
}

// GetSagasByState retrieves sagas by state
func (s *MemoryStateStore) GetSagasByState(ctx context.Context, state BookingState, limit int) ([]*BookingSaga, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*BookingSaga
	for _, saga := range s.sagas {
		if saga.State == state {
			result = append(result, s.copySaga(saga))
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// copySaga creates a deep copy of a saga
func (s *MemoryStateStore) copySaga(saga *BookingSaga) *BookingSaga {
	if saga == nil {
		return nil
	}

	copied := &BookingSaga{
		ID:             saga.ID,
		BookingID:      saga.BookingID,
		EventID:        saga.EventID,
		UserID:         saga.UserID,
		State:          saga.State,
		PreviousState:  saga.PreviousState,
		ReservationID:  saga.ReservationID,
		PaymentID:      saga.PaymentID,
		ConfirmationID: saga.ConfirmationID,
		ErrorMessage:   saga.ErrorMessage,
		RetryCount:     saga.RetryCount,
		CreatedAt:      saga.CreatedAt,
		UpdatedAt:      saga.UpdatedAt,
		CompletedAt:    saga.CompletedAt,
	}

	// Copy data map
	if saga.Data != nil {
		copied.Data = make(map[string]interface{})
		for k, v := range saga.Data {
			copied.Data[k] = v
		}
	}

	return copied
}

// Clear removes all data (for testing)
func (s *MemoryStateStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sagas = make(map[string]*BookingSaga)
	s.transitions = make(map[string][]StateTransition)
	s.byBookingID = make(map[string]string)
}

// Count returns the number of stored sagas (for testing)
func (s *MemoryStateStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sagas)
}
