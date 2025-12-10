package saga

import (
	"context"
	"testing"
)

func TestBookingStateIsTerminal(t *testing.T) {
	tests := []struct {
		state    BookingState
		expected bool
	}{
		{StateCreated, false},
		{StateReserved, false},
		{StatePaid, false},
		{StateConfirmed, true},
		{StateFailed, true},
		{StateCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.expected {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBookingStateIsValid(t *testing.T) {
	tests := []struct {
		state    BookingState
		expected bool
	}{
		{StateCreated, true},
		{StateReserved, true},
		{StatePaid, true},
		{StateConfirmed, true},
		{StateFailed, true},
		{StateCancelled, true},
		{BookingState("INVALID"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBookingStateCanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     BookingState
		to       BookingState
		expected bool
	}{
		// From CREATED
		{"CREATED -> RESERVED", StateCreated, StateReserved, true},
		{"CREATED -> FAILED", StateCreated, StateFailed, true},
		{"CREATED -> CANCELLED", StateCreated, StateCancelled, true},
		{"CREATED -> PAID", StateCreated, StatePaid, false},
		{"CREATED -> CONFIRMED", StateCreated, StateConfirmed, false},

		// From RESERVED
		{"RESERVED -> PAID", StateReserved, StatePaid, true},
		{"RESERVED -> FAILED", StateReserved, StateFailed, true},
		{"RESERVED -> CANCELLED", StateReserved, StateCancelled, true},
		{"RESERVED -> CONFIRMED", StateReserved, StateConfirmed, false},
		{"RESERVED -> CREATED", StateReserved, StateCreated, false},

		// From PAID
		{"PAID -> CONFIRMED", StatePaid, StateConfirmed, true},
		{"PAID -> FAILED", StatePaid, StateFailed, true},
		{"PAID -> CANCELLED", StatePaid, StateCancelled, false},
		{"PAID -> RESERVED", StatePaid, StateReserved, false},

		// Terminal states
		{"CONFIRMED -> any", StateConfirmed, StateReserved, false},
		{"FAILED -> any", StateFailed, StateCreated, false},
		{"CANCELLED -> any", StateCancelled, StateReserved, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.expected {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestStateMachineCreateSaga(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, err := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", map[string]interface{}{
		"seats": 2,
	})

	if err != nil {
		t.Fatalf("CreateSaga failed: %v", err)
	}

	if saga.ID == "" {
		t.Error("expected non-empty ID")
	}
	if saga.BookingID != "booking-123" {
		t.Errorf("expected booking_id 'booking-123', got '%s'", saga.BookingID)
	}
	if saga.EventID != "event-456" {
		t.Errorf("expected event_id 'event-456', got '%s'", saga.EventID)
	}
	if saga.UserID != "user-789" {
		t.Errorf("expected user_id 'user-789', got '%s'", saga.UserID)
	}
	if saga.State != StateCreated {
		t.Errorf("expected state 'CREATED', got '%s'", saga.State)
	}
	if saga.Data["seats"] != 2 {
		t.Errorf("expected seats 2, got %v", saga.Data["seats"])
	}
}

func TestStateMachineTransitionTo(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	// Create saga
	saga, _ := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", nil)

	// Valid transition: CREATED -> RESERVED
	updated, err := sm.TransitionTo(ctx, saga.ID, StateReserved, "Seats reserved")
	if err != nil {
		t.Fatalf("TransitionTo failed: %v", err)
	}
	if updated.State != StateReserved {
		t.Errorf("expected state 'RESERVED', got '%s'", updated.State)
	}
	if updated.PreviousState != StateCreated {
		t.Errorf("expected previous state 'CREATED', got '%s'", updated.PreviousState)
	}

	// Invalid transition: RESERVED -> CREATED
	_, err = sm.TransitionTo(ctx, saga.ID, StateCreated, "Invalid transition")
	if err == nil {
		t.Error("expected error for invalid transition")
	}
}

func TestStateMachineMarkReserved(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, _ := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", nil)

	updated, err := sm.MarkReserved(ctx, saga.ID, "res-abc123")
	if err != nil {
		t.Fatalf("MarkReserved failed: %v", err)
	}

	if updated.State != StateReserved {
		t.Errorf("expected state 'RESERVED', got '%s'", updated.State)
	}
	if updated.ReservationID != "res-abc123" {
		t.Errorf("expected reservation_id 'res-abc123', got '%s'", updated.ReservationID)
	}
}

func TestStateMachineMarkPaid(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, _ := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", nil)
	sm.MarkReserved(ctx, saga.ID, "res-abc123")

	updated, err := sm.MarkPaid(ctx, saga.ID, "pay-xyz789")
	if err != nil {
		t.Fatalf("MarkPaid failed: %v", err)
	}

	if updated.State != StatePaid {
		t.Errorf("expected state 'PAID', got '%s'", updated.State)
	}
	if updated.PaymentID != "pay-xyz789" {
		t.Errorf("expected payment_id 'pay-xyz789', got '%s'", updated.PaymentID)
	}
}

func TestStateMachineMarkConfirmed(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, _ := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", nil)
	sm.MarkReserved(ctx, saga.ID, "res-abc123")
	sm.MarkPaid(ctx, saga.ID, "pay-xyz789")

	updated, err := sm.MarkConfirmed(ctx, saga.ID, "conf-final")
	if err != nil {
		t.Fatalf("MarkConfirmed failed: %v", err)
	}

	if updated.State != StateConfirmed {
		t.Errorf("expected state 'CONFIRMED', got '%s'", updated.State)
	}
	if updated.ConfirmationID != "conf-final" {
		t.Errorf("expected confirmation_id 'conf-final', got '%s'", updated.ConfirmationID)
	}
	if updated.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}
}

func TestStateMachineMarkFailed(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, _ := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", nil)
	sm.MarkReserved(ctx, saga.ID, "res-abc123")

	updated, err := sm.MarkFailed(ctx, saga.ID, "Payment declined")
	if err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}

	if updated.State != StateFailed {
		t.Errorf("expected state 'FAILED', got '%s'", updated.State)
	}
	if updated.ErrorMessage != "Payment declined" {
		t.Errorf("expected error message 'Payment declined', got '%s'", updated.ErrorMessage)
	}
	if updated.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", updated.RetryCount)
	}
}

func TestStateMachineMarkCancelled(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, _ := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", nil)
	sm.MarkReserved(ctx, saga.ID, "res-abc123")

	updated, err := sm.MarkCancelled(ctx, saga.ID, "User requested cancellation")
	if err != nil {
		t.Fatalf("MarkCancelled failed: %v", err)
	}

	if updated.State != StateCancelled {
		t.Errorf("expected state 'CANCELLED', got '%s'", updated.State)
	}
}

func TestStateMachineCannotCancelAfterPaid(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, _ := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", nil)
	sm.MarkReserved(ctx, saga.ID, "res-abc123")
	sm.MarkPaid(ctx, saga.ID, "pay-xyz789")

	_, err := sm.MarkCancelled(ctx, saga.ID, "User requested cancellation")
	if err == nil {
		t.Error("expected error when cancelling after payment")
	}
}

func TestStateMachineCannotFailFromTerminalState(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, _ := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", nil)
	sm.MarkReserved(ctx, saga.ID, "res-abc123")
	sm.MarkPaid(ctx, saga.ID, "pay-xyz789")
	sm.MarkConfirmed(ctx, saga.ID, "conf-final")

	_, err := sm.MarkFailed(ctx, saga.ID, "Some error")
	if err == nil {
		t.Error("expected error when failing from terminal state")
	}
}

func TestStateMachineGetTransitionHistory(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, _ := sm.CreateSaga(ctx, "booking-123", "event-456", "user-789", nil)
	sm.MarkReserved(ctx, saga.ID, "res-abc123")
	sm.MarkPaid(ctx, saga.ID, "pay-xyz789")
	sm.MarkConfirmed(ctx, saga.ID, "conf-final")

	history, err := sm.GetTransitionHistory(ctx, saga.ID)
	if err != nil {
		t.Fatalf("GetTransitionHistory failed: %v", err)
	}

	if len(history) != 3 {
		t.Fatalf("expected 3 transitions, got %d", len(history))
	}

	// Verify transition order
	expected := []struct {
		from BookingState
		to   BookingState
	}{
		{StateCreated, StateReserved},
		{StateReserved, StatePaid},
		{StatePaid, StateConfirmed},
	}

	for i, e := range expected {
		if history[i].FromState != e.from {
			t.Errorf("transition %d: expected from state '%s', got '%s'", i, e.from, history[i].FromState)
		}
		if history[i].ToState != e.to {
			t.Errorf("transition %d: expected to state '%s', got '%s'", i, e.to, history[i].ToState)
		}
	}
}

func TestStateMachineGetPendingSagas(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	// Create sagas in different states
	saga1, _ := sm.CreateSaga(ctx, "booking-1", "event-1", "user-1", nil)
	saga2, _ := sm.CreateSaga(ctx, "booking-2", "event-1", "user-2", nil)
	saga3, _ := sm.CreateSaga(ctx, "booking-3", "event-1", "user-3", nil)

	// Move saga2 to RESERVED
	sm.MarkReserved(ctx, saga2.ID, "res-2")

	// Move saga3 to CONFIRMED (terminal)
	sm.MarkReserved(ctx, saga3.ID, "res-3")
	sm.MarkPaid(ctx, saga3.ID, "pay-3")
	sm.MarkConfirmed(ctx, saga3.ID, "conf-3")

	pending, err := sm.GetPendingSagas(ctx, 10)
	if err != nil {
		t.Fatalf("GetPendingSagas failed: %v", err)
	}

	// Should only get saga1 (CREATED) and saga2 (RESERVED)
	if len(pending) != 2 {
		t.Errorf("expected 2 pending sagas, got %d", len(pending))
	}

	// Verify that saga3 is not in pending list
	for _, s := range pending {
		if s.ID == saga3.ID {
			t.Error("saga3 should not be in pending list")
		}
	}

	// Verify saga1 is in list
	found := false
	for _, s := range pending {
		if s.ID == saga1.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("saga1 should be in pending list")
	}
}

func TestStateMachineGetSagaByBookingID(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	saga, _ := sm.CreateSaga(ctx, "booking-unique", "event-456", "user-789", nil)

	retrieved, err := sm.GetSagaByBookingID(ctx, "booking-unique")
	if err != nil {
		t.Fatalf("GetSagaByBookingID failed: %v", err)
	}

	if retrieved.ID != saga.ID {
		t.Errorf("expected saga ID '%s', got '%s'", saga.ID, retrieved.ID)
	}
}

func TestStateMachineFullBookingFlow(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStateStore()
	sm := NewStateMachine(store)

	// 1. Create saga
	saga, err := sm.CreateSaga(ctx, "booking-flow", "event-concert", "user-fan", map[string]interface{}{
		"seats":      4,
		"seat_type":  "VIP",
		"total_cost": 400.00,
	})
	if err != nil {
		t.Fatalf("CreateSaga failed: %v", err)
	}
	if saga.State != StateCreated {
		t.Errorf("step 1: expected CREATED, got %s", saga.State)
	}

	// 2. Reserve seats
	saga, err = sm.MarkReserved(ctx, saga.ID, "reservation-12345")
	if err != nil {
		t.Fatalf("MarkReserved failed: %v", err)
	}
	if saga.State != StateReserved {
		t.Errorf("step 2: expected RESERVED, got %s", saga.State)
	}

	// 3. Process payment
	saga, err = sm.MarkPaid(ctx, saga.ID, "payment-67890")
	if err != nil {
		t.Fatalf("MarkPaid failed: %v", err)
	}
	if saga.State != StatePaid {
		t.Errorf("step 3: expected PAID, got %s", saga.State)
	}

	// 4. Confirm booking
	saga, err = sm.MarkConfirmed(ctx, saga.ID, "confirmation-ABCDE")
	if err != nil {
		t.Fatalf("MarkConfirmed failed: %v", err)
	}
	if saga.State != StateConfirmed {
		t.Errorf("step 4: expected CONFIRMED, got %s", saga.State)
	}

	// Verify final state
	if saga.ReservationID != "reservation-12345" {
		t.Errorf("expected reservation_id 'reservation-12345', got '%s'", saga.ReservationID)
	}
	if saga.PaymentID != "payment-67890" {
		t.Errorf("expected payment_id 'payment-67890', got '%s'", saga.PaymentID)
	}
	if saga.ConfirmationID != "confirmation-ABCDE" {
		t.Errorf("expected confirmation_id 'confirmation-ABCDE', got '%s'", saga.ConfirmationID)
	}
	if saga.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}

	// Verify transition history
	history, _ := sm.GetTransitionHistory(ctx, saga.ID)
	if len(history) != 3 {
		t.Errorf("expected 3 transitions, got %d", len(history))
	}
}
