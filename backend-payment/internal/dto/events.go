package dto

import (
	"time"
)

// Topic names for payment events
const (
	TopicSeatRelease     = "payment.seat-release"
	TopicPaymentSuccess  = "payment.success"
)

// SeatReleaseReason represents the reason for releasing seats
type SeatReleaseReason string

const (
	SeatReleaseReasonPaymentFailed   SeatReleaseReason = "payment_failed"
	SeatReleaseReasonPaymentCanceled SeatReleaseReason = "payment_canceled"
	SeatReleaseReasonPaymentRefunded SeatReleaseReason = "payment_refunded"
)

// SeatReleaseEvent is published when seats need to be released due to payment failure
type SeatReleaseEvent struct {
	EventType   string            `json:"event_type"`
	BookingID   string            `json:"booking_id"`
	PaymentID   string            `json:"payment_id"`
	UserID      string            `json:"user_id,omitempty"`
	Reason      SeatReleaseReason `json:"reason"`
	FailureCode string            `json:"failure_code,omitempty"`
	Message     string            `json:"message,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Key returns the Kafka message key for partitioning
func (e *SeatReleaseEvent) Key() string {
	return e.BookingID
}

// PaymentSuccessEvent is published when payment succeeds to trigger post-payment saga
// This event contains enriched booking data for notification service
type PaymentSuccessEvent struct {
	EventType             string    `json:"event_type"`
	BookingID             string    `json:"booking_id"`
	PaymentID             string    `json:"payment_id"`
	StripePaymentIntentID string    `json:"stripe_payment_intent_id"`
	UserID                string    `json:"user_id,omitempty"`
	Amount                int64     `json:"amount"`
	Currency              string    `json:"currency"`
	Timestamp             time.Time `json:"timestamp"`

	// Enriched booking data for notification service
	UserEmail        string  `json:"user_email,omitempty"`
	EventID          string  `json:"event_id,omitempty"`
	EventName        string  `json:"event_name,omitempty"`
	ShowID           string  `json:"show_id,omitempty"`
	ShowDate         string  `json:"show_date,omitempty"`
	ZoneID           string  `json:"zone_id,omitempty"`
	ZoneName         string  `json:"zone_name,omitempty"`
	Quantity         int     `json:"quantity,omitempty"`
	UnitPrice        float64 `json:"unit_price,omitempty"`
	TotalPrice       float64 `json:"total_price,omitempty"`
	ConfirmationCode string  `json:"confirmation_code,omitempty"`
	VenueName        string  `json:"venue_name,omitempty"`
	VenueAddress     string  `json:"venue_address,omitempty"`
}

// Key returns the Kafka message key for partitioning
func (e *PaymentSuccessEvent) Key() string {
	return e.BookingID
}
