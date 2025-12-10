package saga

import (
	"context"
	"errors"
	"testing"
	"time"

	pkgsaga "github.com/prohmpiriya/booking-rush-10k-rps/pkg/saga"
)

func TestBookingSagaBuilder_Build(t *testing.T) {
	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  NewMockSeatReservationService(),
		PaymentService:      NewMockPaymentService(),
		ConfirmationService: NewMockBookingConfirmationService(),
		NotificationService: NewMockNotificationService(),
	})

	def := builder.Build()

	if def.Name != BookingSagaName {
		t.Errorf("expected saga name %s, got %s", BookingSagaName, def.Name)
	}

	if len(def.Steps) != 4 {
		t.Errorf("expected 4 steps, got %d", len(def.Steps))
	}

	expectedSteps := []string{
		StepReserveSeats,
		StepProcessPayment,
		StepConfirmBooking,
		StepSendNotification,
	}

	for i, step := range def.Steps {
		if step.Name != expectedSteps[i] {
			t.Errorf("step %d: expected name %s, got %s", i, expectedSteps[i], step.Name)
		}
	}
}

func TestBookingSaga_SuccessfulExecution(t *testing.T) {
	// Setup mock services
	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()
	notificationSvc := NewMockNotificationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: notificationSvc,
		StepTimeout:         5 * time.Second,
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-123",
		"user_id":        "user-456",
		"event_id":       "event-789",
		"zone_id":        "zone-A",
		"quantity":       2,
		"total_price":    200.00,
		"currency":       "THB",
		"payment_method": "credit_card",
	}

	instance, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err != nil {
		t.Fatalf("saga execution failed: %v", err)
	}

	// Verify saga completed successfully
	if instance.Status != pkgsaga.StatusCompleted {
		t.Errorf("expected status %s, got %s", pkgsaga.StatusCompleted, instance.Status)
	}

	// Verify all steps completed
	if len(instance.StepResults) != 4 {
		t.Errorf("expected 4 step results, got %d", len(instance.StepResults))
	}

	for _, result := range instance.StepResults {
		if result.Status != pkgsaga.StepStatusCompleted {
			t.Errorf("step %s: expected status %s, got %s", result.StepName, pkgsaga.StepStatusCompleted, result.Status)
		}
	}

	// Verify reservation was created
	reservation, exists := reservationSvc.GetReservation("booking-123")
	if !exists {
		t.Error("expected reservation to exist")
	}
	if reservation.Released {
		t.Error("expected reservation not to be released")
	}

	// Verify payment was processed
	payment, exists := paymentSvc.GetPaymentByBookingID("booking-123")
	if !exists {
		t.Error("expected payment to exist")
	}
	if payment.Refunded {
		t.Error("expected payment not to be refunded")
	}

	// Verify booking was confirmed
	confirmation, exists := confirmationSvc.GetConfirmation("booking-123")
	if !exists {
		t.Error("expected confirmation to exist")
	}
	if confirmation.ConfirmationCode == "" {
		t.Error("expected confirmation code to be set")
	}

	// Verify notification was sent
	notification, exists := notificationSvc.GetNotificationByBookingID("booking-123")
	if !exists {
		t.Error("expected notification to exist")
	}
	if notification.NotificationID == "" {
		t.Error("expected notification ID to be set")
	}
}

func TestBookingSaga_ReservationFailure_NoCompensation(t *testing.T) {
	// Setup mock services with reservation failure
	reservationSvc := NewMockSeatReservationService()
	reservationSvc.ShouldFail = true
	reservationSvc.FailureError = ErrInsufficientSeats

	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()
	notificationSvc := NewMockNotificationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: notificationSvc,
		StepTimeout:         5 * time.Second,
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-123",
		"user_id":        "user-456",
		"event_id":       "event-789",
		"zone_id":        "zone-A",
		"quantity":       2,
		"total_price":    200.00,
		"currency":       "THB",
		"payment_method": "credit_card",
	}

	_, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err == nil {
		t.Fatal("expected saga execution to fail")
	}

	// Verify no payment was processed (first step failed, no compensation needed)
	_, exists := paymentSvc.GetPaymentByBookingID("booking-123")
	if exists {
		t.Error("expected no payment to exist since reservation failed")
	}
}

func TestBookingSaga_PaymentFailure_ReleasesSeats(t *testing.T) {
	// Setup mock services with payment failure
	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	paymentSvc.ShouldFail = true
	paymentSvc.FailureError = ErrPaymentDeclined

	confirmationSvc := NewMockBookingConfirmationService()
	notificationSvc := NewMockNotificationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: notificationSvc,
		StepTimeout:         5 * time.Second,
		MaxRetries:          0, // No retries for faster test
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-456",
		"user_id":        "user-789",
		"event_id":       "event-123",
		"zone_id":        "zone-B",
		"quantity":       3,
		"total_price":    300.00,
		"currency":       "THB",
		"payment_method": "credit_card",
	}

	_, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err == nil {
		t.Fatal("expected saga execution to fail")
	}

	// Verify reservation was created and then released (compensated)
	reservation, exists := reservationSvc.GetReservation("booking-456")
	if !exists {
		t.Error("expected reservation to exist")
	}
	if !reservation.Released {
		t.Error("expected reservation to be released (compensated)")
	}
}

func TestBookingSaga_ConfirmationFailure_RefundsPayment(t *testing.T) {
	// Setup mock services with confirmation failure
	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()
	confirmationSvc.ShouldFail = true
	confirmationSvc.FailureError = errors.New("confirmation service unavailable")

	notificationSvc := NewMockNotificationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: notificationSvc,
		StepTimeout:         5 * time.Second,
		MaxRetries:          0,
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-789",
		"user_id":        "user-123",
		"event_id":       "event-456",
		"zone_id":        "zone-C",
		"quantity":       1,
		"total_price":    100.00,
		"currency":       "THB",
		"payment_method": "debit_card",
	}

	_, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err == nil {
		t.Fatal("expected saga execution to fail")
	}

	// Verify reservation was released
	reservation, exists := reservationSvc.GetReservation("booking-789")
	if !exists {
		t.Error("expected reservation to exist")
	}
	if !reservation.Released {
		t.Error("expected reservation to be released (compensated)")
	}

	// Verify payment was refunded
	payment, exists := paymentSvc.GetPaymentByBookingID("booking-789")
	if !exists {
		t.Error("expected payment to exist")
	}
	if !payment.Refunded {
		t.Error("expected payment to be refunded (compensated)")
	}
}

func TestBookingSaga_NotificationFailure_StillCompletes(t *testing.T) {
	// Setup mock services with notification failure
	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()
	notificationSvc := NewMockNotificationService()
	notificationSvc.ShouldFail = true
	notificationSvc.FailureError = errors.New("notification service unavailable")

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: notificationSvc,
		StepTimeout:         5 * time.Second,
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-notify-fail",
		"user_id":        "user-notify",
		"event_id":       "event-notify",
		"zone_id":        "zone-D",
		"quantity":       2,
		"total_price":    250.00,
		"currency":       "THB",
		"payment_method": "bank_transfer",
	}

	instance, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err != nil {
		t.Fatalf("saga execution should succeed even with notification failure: %v", err)
	}

	// Verify saga completed successfully (notification failure is not critical)
	if instance.Status != pkgsaga.StatusCompleted {
		t.Errorf("expected status %s, got %s", pkgsaga.StatusCompleted, instance.Status)
	}

	// Verify reservation was not released
	reservation, exists := reservationSvc.GetReservation("booking-notify-fail")
	if !exists {
		t.Error("expected reservation to exist")
	}
	if reservation.Released {
		t.Error("expected reservation not to be released")
	}

	// Verify payment was not refunded
	payment, exists := paymentSvc.GetPaymentByBookingID("booking-notify-fail")
	if !exists {
		t.Error("expected payment to exist")
	}
	if payment.Refunded {
		t.Error("expected payment not to be refunded")
	}

	// Verify booking was confirmed
	confirmation, exists := confirmationSvc.GetConfirmation("booking-notify-fail")
	if !exists {
		t.Error("expected confirmation to exist")
	}
	if confirmation.ConfirmationCode == "" {
		t.Error("expected confirmation code to be set")
	}
}

func TestBookingSaga_WithoutNotificationService(t *testing.T) {
	// Setup mock services without notification service
	reservationSvc := NewMockSeatReservationService()
	paymentSvc := NewMockPaymentService()
	confirmationSvc := NewMockBookingConfirmationService()

	builder := NewBookingSagaBuilder(&BookingSagaConfig{
		ReservationService:  reservationSvc,
		PaymentService:      paymentSvc,
		ConfirmationService: confirmationSvc,
		NotificationService: nil, // No notification service
		StepTimeout:         5 * time.Second,
	})

	// Create orchestrator
	orchestrator := pkgsaga.NewOrchestrator(&pkgsaga.OrchestratorConfig{
		Store: pkgsaga.NewMemoryStore(),
	})

	// Register saga definition
	def := builder.Build()
	if err := orchestrator.RegisterDefinition(def); err != nil {
		t.Fatalf("failed to register saga definition: %v", err)
	}

	// Execute saga
	ctx := context.Background()
	initialData := map[string]interface{}{
		"booking_id":     "booking-no-notify",
		"user_id":        "user-no-notify",
		"event_id":       "event-no-notify",
		"zone_id":        "zone-E",
		"quantity":       1,
		"total_price":    150.00,
		"currency":       "THB",
		"payment_method": "credit_card",
	}

	instance, err := orchestrator.Execute(ctx, BookingSagaName, initialData)
	if err != nil {
		t.Fatalf("saga execution failed: %v", err)
	}

	// Verify saga completed successfully
	if instance.Status != pkgsaga.StatusCompleted {
		t.Errorf("expected status %s, got %s", pkgsaga.StatusCompleted, instance.Status)
	}
}

func TestBookingSagaData_ToMapAndFromMap(t *testing.T) {
	original := &BookingSagaData{
		BookingID:        "booking-123",
		UserID:           "user-456",
		EventID:          "event-789",
		ZoneID:           "zone-A",
		Quantity:         5,
		TotalPrice:       500.00,
		Currency:         "THB",
		PaymentMethod:    "credit_card",
		IdempotencyKey:   "idem-key-123",
		ReservationID:    "res-123",
		PaymentID:        "pay-456",
		ConfirmationCode: "CONF-789",
		NotificationID:   "notif-012",
	}

	// Convert to map
	m := original.ToMap()

	// Convert back from map
	restored := &BookingSagaData{}
	restored.FromMap(m)

	// Verify all fields
	if restored.BookingID != original.BookingID {
		t.Errorf("BookingID: expected %s, got %s", original.BookingID, restored.BookingID)
	}
	if restored.UserID != original.UserID {
		t.Errorf("UserID: expected %s, got %s", original.UserID, restored.UserID)
	}
	if restored.EventID != original.EventID {
		t.Errorf("EventID: expected %s, got %s", original.EventID, restored.EventID)
	}
	if restored.ZoneID != original.ZoneID {
		t.Errorf("ZoneID: expected %s, got %s", original.ZoneID, restored.ZoneID)
	}
	if restored.Quantity != original.Quantity {
		t.Errorf("Quantity: expected %d, got %d", original.Quantity, restored.Quantity)
	}
	if restored.TotalPrice != original.TotalPrice {
		t.Errorf("TotalPrice: expected %f, got %f", original.TotalPrice, restored.TotalPrice)
	}
	if restored.Currency != original.Currency {
		t.Errorf("Currency: expected %s, got %s", original.Currency, restored.Currency)
	}
	if restored.PaymentMethod != original.PaymentMethod {
		t.Errorf("PaymentMethod: expected %s, got %s", original.PaymentMethod, restored.PaymentMethod)
	}
	if restored.IdempotencyKey != original.IdempotencyKey {
		t.Errorf("IdempotencyKey: expected %s, got %s", original.IdempotencyKey, restored.IdempotencyKey)
	}
	if restored.ReservationID != original.ReservationID {
		t.Errorf("ReservationID: expected %s, got %s", original.ReservationID, restored.ReservationID)
	}
	if restored.PaymentID != original.PaymentID {
		t.Errorf("PaymentID: expected %s, got %s", original.PaymentID, restored.PaymentID)
	}
	if restored.ConfirmationCode != original.ConfirmationCode {
		t.Errorf("ConfirmationCode: expected %s, got %s", original.ConfirmationCode, restored.ConfirmationCode)
	}
	if restored.NotificationID != original.NotificationID {
		t.Errorf("NotificationID: expected %s, got %s", original.NotificationID, restored.NotificationID)
	}
}

func TestMockSeatReservationService(t *testing.T) {
	svc := NewMockSeatReservationService()
	ctx := context.Background()

	// Test successful reservation
	reservationID, err := svc.ReserveSeats(ctx, "booking-1", "user-1", "event-1", "zone-1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reservationID == "" {
		t.Error("expected reservation ID to be set")
	}

	// Verify reservation exists
	reservation, exists := svc.GetReservation("booking-1")
	if !exists {
		t.Error("expected reservation to exist")
	}
	if reservation.Quantity != 2 {
		t.Errorf("expected quantity 2, got %d", reservation.Quantity)
	}

	// Test release
	err = svc.ReleaseSeats(ctx, "booking-1", "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reservation, _ = svc.GetReservation("booking-1")
	if !reservation.Released {
		t.Error("expected reservation to be released")
	}

	// Test release non-existent
	err = svc.ReleaseSeats(ctx, "non-existent", "user-1")
	if !errors.Is(err, ErrReservationNotFound) {
		t.Errorf("expected ErrReservationNotFound, got %v", err)
	}
}

func TestMockPaymentService(t *testing.T) {
	svc := NewMockPaymentService()
	ctx := context.Background()

	// Test successful payment
	paymentID, err := svc.ProcessPayment(ctx, "booking-1", "user-1", 100.00, "THB", "credit_card")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paymentID == "" {
		t.Error("expected payment ID to be set")
	}

	// Verify payment exists
	payment, exists := svc.GetPayment(paymentID)
	if !exists {
		t.Error("expected payment to exist")
	}
	if payment.Amount != 100.00 {
		t.Errorf("expected amount 100.00, got %f", payment.Amount)
	}

	// Test refund
	err = svc.RefundPayment(ctx, paymentID, "test refund")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payment, _ = svc.GetPayment(paymentID)
	if !payment.Refunded {
		t.Error("expected payment to be refunded")
	}
	if payment.RefundReason != "test refund" {
		t.Errorf("expected refund reason 'test refund', got '%s'", payment.RefundReason)
	}

	// Test refund non-existent
	err = svc.RefundPayment(ctx, "non-existent", "reason")
	if !errors.Is(err, ErrPaymentNotFound) {
		t.Errorf("expected ErrPaymentNotFound, got %v", err)
	}
}

func TestMockBookingConfirmationService(t *testing.T) {
	svc := NewMockBookingConfirmationService()
	ctx := context.Background()

	// Test successful confirmation
	confirmationCode, err := svc.ConfirmBooking(ctx, "booking-1", "user-1", "payment-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if confirmationCode == "" {
		t.Error("expected confirmation code to be set")
	}

	// Verify confirmation exists
	confirmation, exists := svc.GetConfirmation("booking-1")
	if !exists {
		t.Error("expected confirmation to exist")
	}
	if confirmation.PaymentID != "payment-1" {
		t.Errorf("expected payment ID 'payment-1', got '%s'", confirmation.PaymentID)
	}
}

func TestMockNotificationService(t *testing.T) {
	svc := NewMockNotificationService()
	ctx := context.Background()

	// Test successful notification
	notificationID, err := svc.SendBookingConfirmation(ctx, "user-1", "booking-1", "CONF-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notificationID == "" {
		t.Error("expected notification ID to be set")
	}

	// Verify notification exists
	notification, exists := svc.GetNotification(notificationID)
	if !exists {
		t.Error("expected notification to exist")
	}
	if notification.ConfirmationCode != "CONF-123" {
		t.Errorf("expected confirmation code 'CONF-123', got '%s'", notification.ConfirmationCode)
	}
}
