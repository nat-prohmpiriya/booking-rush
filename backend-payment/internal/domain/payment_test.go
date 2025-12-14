package domain

import (
	"testing"
)

func TestNewPayment(t *testing.T) {
	tests := []struct {
		name      string
		tenantID  string
		bookingID string
		userID    string
		amount    float64
		currency  string
		method    PaymentMethod
		wantErr   bool
	}{
		{
			name:      "valid payment",
			tenantID:  "tenant-123",
			bookingID: "booking-123",
			userID:    "user-123",
			amount:    100.00,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   false,
		},
		{
			name:      "missing tenant_id",
			tenantID:  "",
			bookingID: "booking-123",
			userID:    "user-123",
			amount:    100.00,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
		{
			name:      "missing booking_id",
			tenantID:  "tenant-123",
			bookingID: "",
			userID:    "user-123",
			amount:    100.00,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
		{
			name:      "missing user_id",
			tenantID:  "tenant-123",
			bookingID: "booking-123",
			userID:    "",
			amount:    100.00,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
		{
			name:      "zero amount",
			tenantID:  "tenant-123",
			bookingID: "booking-123",
			userID:    "user-123",
			amount:    0,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
		{
			name:      "negative amount",
			tenantID:  "tenant-123",
			bookingID: "booking-123",
			userID:    "user-123",
			amount:    -50.00,
			currency:  "THB",
			method:    PaymentMethodCreditCard,
			wantErr:   true,
		},
		{
			name:      "empty currency defaults to THB",
			tenantID:  "tenant-123",
			bookingID: "booking-123",
			userID:    "user-123",
			amount:    100.00,
			currency:  "",
			method:    PaymentMethodCreditCard,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, err := NewPayment(tt.tenantID, tt.bookingID, tt.userID, tt.amount, tt.currency, tt.method)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if payment.ID == "" {
				t.Error("Expected payment ID to be set")
			}
			if payment.TenantID != tt.tenantID {
				t.Errorf("Expected tenant_id %s, got %s", tt.tenantID, payment.TenantID)
			}
			if payment.BookingID != tt.bookingID {
				t.Errorf("Expected booking_id %s, got %s", tt.bookingID, payment.BookingID)
			}
			if payment.UserID != tt.userID {
				t.Errorf("Expected user_id %s, got %s", tt.userID, payment.UserID)
			}
			if payment.Amount != tt.amount {
				t.Errorf("Expected amount %f, got %f", tt.amount, payment.Amount)
			}
			if payment.Status != PaymentStatusPending {
				t.Errorf("Expected status pending, got %s", payment.Status)
			}
		})
	}
}

func TestPayment_MarkProcessing(t *testing.T) {
	payment, _ := NewPayment("tenant-123", "booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	err := payment.MarkProcessing()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if payment.Status != PaymentStatusProcessing {
		t.Errorf("Expected status processing, got %s", payment.Status)
	}

	// Should fail if called again
	err = payment.MarkProcessing()
	if err == nil {
		t.Error("Expected error when marking processing again")
	}
}

func TestPayment_Complete(t *testing.T) {
	payment, _ := NewPayment("tenant-123", "booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	// Can complete from pending (fast path)
	err := payment.Complete("pi_123")
	if err != nil {
		t.Errorf("Unexpected error completing from pending: %v", err)
	}

	if payment.Status != PaymentStatusSucceeded {
		t.Errorf("Expected status succeeded, got %s", payment.Status)
	}
	if payment.GatewayPaymentID != "pi_123" {
		t.Errorf("Expected gateway_payment_id pi_123, got %s", payment.GatewayPaymentID)
	}
	if payment.ProcessedAt == nil {
		t.Error("Expected processed_at to be set")
	}

	// Test completing from processing
	payment2, _ := NewPayment("tenant-123", "booking-456", "user-123", 100.00, "THB", PaymentMethodCreditCard)
	payment2.MarkProcessing()
	err = payment2.Complete("pi_456")
	if err != nil {
		t.Errorf("Unexpected error completing from processing: %v", err)
	}
}

func TestPayment_Fail(t *testing.T) {
	payment, _ := NewPayment("tenant-123", "booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	err := payment.Fail("card_declined", "insufficient funds")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if payment.Status != PaymentStatusFailed {
		t.Errorf("Expected status failed, got %s", payment.Status)
	}
	if payment.ErrorCode != "card_declined" {
		t.Errorf("Expected error_code 'card_declined', got '%s'", payment.ErrorCode)
	}
	if payment.ErrorMessage != "insufficient funds" {
		t.Errorf("Expected error_message 'insufficient funds', got '%s'", payment.ErrorMessage)
	}
}

func TestPayment_Refund(t *testing.T) {
	payment, _ := NewPayment("tenant-123", "booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	// Should fail from pending status
	err := payment.Refund(100.00, "customer request")
	if err == nil {
		t.Error("Expected error when refunding from pending status")
	}

	// Complete the payment first
	payment.Complete("pi_123")

	// Should succeed
	err = payment.Refund(100.00, "customer request")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if payment.Status != PaymentStatusRefunded {
		t.Errorf("Expected status refunded, got %s", payment.Status)
	}
	if payment.RefundAmount == nil || *payment.RefundAmount != 100.00 {
		t.Error("Expected refund_amount to be 100.00")
	}
	if payment.RefundReason != "customer request" {
		t.Errorf("Expected refund_reason 'customer request', got '%s'", payment.RefundReason)
	}
}

func TestPayment_Cancel(t *testing.T) {
	payment, _ := NewPayment("tenant-123", "booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	err := payment.Cancel()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if payment.Status != PaymentStatusCancelled {
		t.Errorf("Expected status cancelled, got %s", payment.Status)
	}

	// Should fail if called again
	payment2, _ := NewPayment("tenant-123", "booking-456", "user-123", 100.00, "THB", PaymentMethodCreditCard)
	payment2.MarkProcessing()

	err = payment2.Cancel()
	if err == nil {
		t.Error("Expected error when cancelling processing payment")
	}
}

func TestPayment_IsFinal(t *testing.T) {
	payment, _ := NewPayment("tenant-123", "booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	if payment.IsFinal() {
		t.Error("Pending payment should not be final")
	}

	payment.MarkProcessing()
	if payment.IsFinal() {
		t.Error("Processing payment should not be final")
	}

	payment.Complete("pi_123")
	if !payment.IsFinal() {
		t.Error("Succeeded payment should be final")
	}
}

func TestPayment_IsSuccessful(t *testing.T) {
	payment, _ := NewPayment("tenant-123", "booking-123", "user-123", 100.00, "THB", PaymentMethodCreditCard)

	if payment.IsSuccessful() {
		t.Error("Pending payment should not be successful")
	}

	payment.Complete("pi_123")

	if !payment.IsSuccessful() {
		t.Error("Succeeded payment should be successful")
	}
}
