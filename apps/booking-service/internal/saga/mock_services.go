package saga

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

var (
	// ErrMockServiceFailure is returned when a mock service is configured to fail
	ErrMockServiceFailure = errors.New("mock service failure")
	// ErrInsufficientSeats is returned when there are not enough seats
	ErrInsufficientSeats = errors.New("insufficient seats available")
	// ErrPaymentDeclined is returned when payment is declined
	ErrPaymentDeclined = errors.New("payment declined")
	// ErrReservationNotFound is returned when reservation is not found
	ErrReservationNotFound = errors.New("reservation not found")
	// ErrPaymentNotFound is returned when payment is not found
	ErrPaymentNotFound = errors.New("payment not found")
)

// MockSeatReservationService is a mock implementation of SeatReservationService
type MockSeatReservationService struct {
	mu           sync.RWMutex
	reservations map[string]*MockReservation
	ShouldFail   bool
	FailureError error
}

// MockReservation represents a mock reservation
type MockReservation struct {
	ReservationID string
	BookingID     string
	UserID        string
	EventID       string
	ZoneID        string
	Quantity      int
	Released      bool
}

// NewMockSeatReservationService creates a new mock seat reservation service
func NewMockSeatReservationService() *MockSeatReservationService {
	return &MockSeatReservationService{
		reservations: make(map[string]*MockReservation),
	}
}

// ReserveSeats reserves seats in inventory
func (s *MockSeatReservationService) ReserveSeats(ctx context.Context, bookingID, userID, eventID, zoneID string, quantity int) (string, error) {
	if s.ShouldFail {
		if s.FailureError != nil {
			return "", s.FailureError
		}
		return "", ErrMockServiceFailure
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	reservationID := uuid.New().String()
	s.reservations[bookingID] = &MockReservation{
		ReservationID: reservationID,
		BookingID:     bookingID,
		UserID:        userID,
		EventID:       eventID,
		ZoneID:        zoneID,
		Quantity:      quantity,
		Released:      false,
	}

	return reservationID, nil
}

// ReleaseSeats releases reserved seats back to inventory
func (s *MockSeatReservationService) ReleaseSeats(ctx context.Context, bookingID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reservation, exists := s.reservations[bookingID]
	if !exists {
		return ErrReservationNotFound
	}

	reservation.Released = true
	return nil
}

// GetReservation returns a reservation by booking ID (for testing)
func (s *MockSeatReservationService) GetReservation(bookingID string) (*MockReservation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, exists := s.reservations[bookingID]
	return r, exists
}

// Clear removes all reservations (for testing)
func (s *MockSeatReservationService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reservations = make(map[string]*MockReservation)
}

// MockPaymentService is a mock implementation of PaymentService
type MockPaymentService struct {
	mu           sync.RWMutex
	payments     map[string]*MockPayment
	ShouldFail   bool
	FailureError error
}

// MockPayment represents a mock payment
type MockPayment struct {
	PaymentID   string
	BookingID   string
	UserID      string
	Amount      float64
	Currency    string
	Method      string
	Refunded    bool
	RefundReason string
}

// NewMockPaymentService creates a new mock payment service
func NewMockPaymentService() *MockPaymentService {
	return &MockPaymentService{
		payments: make(map[string]*MockPayment),
	}
}

// ProcessPayment processes a payment for booking
func (s *MockPaymentService) ProcessPayment(ctx context.Context, bookingID, userID string, amount float64, currency, method string) (string, error) {
	if s.ShouldFail {
		if s.FailureError != nil {
			return "", s.FailureError
		}
		return "", ErrMockServiceFailure
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	paymentID := uuid.New().String()
	s.payments[paymentID] = &MockPayment{
		PaymentID: paymentID,
		BookingID: bookingID,
		UserID:    userID,
		Amount:    amount,
		Currency:  currency,
		Method:    method,
		Refunded:  false,
	}

	return paymentID, nil
}

// RefundPayment refunds a payment
func (s *MockPaymentService) RefundPayment(ctx context.Context, paymentID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	payment, exists := s.payments[paymentID]
	if !exists {
		return ErrPaymentNotFound
	}

	payment.Refunded = true
	payment.RefundReason = reason
	return nil
}

// GetPayment returns a payment by ID (for testing)
func (s *MockPaymentService) GetPayment(paymentID string) (*MockPayment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, exists := s.payments[paymentID]
	return p, exists
}

// GetPaymentByBookingID returns a payment by booking ID (for testing)
func (s *MockPaymentService) GetPaymentByBookingID(bookingID string) (*MockPayment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.payments {
		if p.BookingID == bookingID {
			return p, true
		}
	}
	return nil, false
}

// Clear removes all payments (for testing)
func (s *MockPaymentService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payments = make(map[string]*MockPayment)
}

// MockBookingConfirmationService is a mock implementation of BookingConfirmationService
type MockBookingConfirmationService struct {
	mu            sync.RWMutex
	confirmations map[string]*MockConfirmation
	ShouldFail    bool
	FailureError  error
}

// MockConfirmation represents a mock booking confirmation
type MockConfirmation struct {
	BookingID        string
	UserID           string
	PaymentID        string
	ConfirmationCode string
}

// NewMockBookingConfirmationService creates a new mock booking confirmation service
func NewMockBookingConfirmationService() *MockBookingConfirmationService {
	return &MockBookingConfirmationService{
		confirmations: make(map[string]*MockConfirmation),
	}
}

// ConfirmBooking confirms a booking after payment
func (s *MockBookingConfirmationService) ConfirmBooking(ctx context.Context, bookingID, userID, paymentID string) (string, error) {
	if s.ShouldFail {
		if s.FailureError != nil {
			return "", s.FailureError
		}
		return "", ErrMockServiceFailure
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	confirmationCode := generateConfirmationCode()
	s.confirmations[bookingID] = &MockConfirmation{
		BookingID:        bookingID,
		UserID:           userID,
		PaymentID:        paymentID,
		ConfirmationCode: confirmationCode,
	}

	return confirmationCode, nil
}

// GetConfirmation returns a confirmation by booking ID (for testing)
func (s *MockBookingConfirmationService) GetConfirmation(bookingID string) (*MockConfirmation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, exists := s.confirmations[bookingID]
	return c, exists
}

// Clear removes all confirmations (for testing)
func (s *MockBookingConfirmationService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.confirmations = make(map[string]*MockConfirmation)
}

// MockNotificationService is a mock implementation of NotificationService
type MockNotificationService struct {
	mu            sync.RWMutex
	notifications map[string]*MockNotification
	ShouldFail    bool
	FailureError  error
}

// MockNotification represents a mock notification
type MockNotification struct {
	NotificationID   string
	UserID           string
	BookingID        string
	ConfirmationCode string
}

// NewMockNotificationService creates a new mock notification service
func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{
		notifications: make(map[string]*MockNotification),
	}
}

// SendBookingConfirmation sends a booking confirmation notification
func (s *MockNotificationService) SendBookingConfirmation(ctx context.Context, userID, bookingID, confirmationCode string) (string, error) {
	if s.ShouldFail {
		if s.FailureError != nil {
			return "", s.FailureError
		}
		return "", ErrMockServiceFailure
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	notificationID := uuid.New().String()
	s.notifications[notificationID] = &MockNotification{
		NotificationID:   notificationID,
		UserID:           userID,
		BookingID:        bookingID,
		ConfirmationCode: confirmationCode,
	}

	return notificationID, nil
}

// GetNotification returns a notification by ID (for testing)
func (s *MockNotificationService) GetNotification(notificationID string) (*MockNotification, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, exists := s.notifications[notificationID]
	return n, exists
}

// GetNotificationByBookingID returns a notification by booking ID (for testing)
func (s *MockNotificationService) GetNotificationByBookingID(bookingID string) (*MockNotification, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, n := range s.notifications {
		if n.BookingID == bookingID {
			return n, true
		}
	}
	return nil, false
}

// Clear removes all notifications (for testing)
func (s *MockNotificationService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications = make(map[string]*MockNotification)
}

// generateConfirmationCode generates a random confirmation code
func generateConfirmationCode() string {
	return uuid.New().String()[:8]
}
