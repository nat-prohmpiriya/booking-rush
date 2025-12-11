package gateway

import (
	"context"
)

// PaymentGateway defines the interface for payment processing
type PaymentGateway interface {
	// Charge processes a payment charge
	Charge(ctx context.Context, req *ChargeRequest) (*ChargeResponse, error)

	// Refund processes a refund
	Refund(ctx context.Context, transactionID string, amount float64) error

	// GetTransaction retrieves transaction details
	GetTransaction(ctx context.Context, transactionID string) (*TransactionInfo, error)

	// CreatePaymentIntent creates a Stripe PaymentIntent and returns client_secret
	CreatePaymentIntent(ctx context.Context, req *PaymentIntentRequest) (*PaymentIntentResponse, error)

	// ConfirmPaymentIntent confirms a PaymentIntent after client-side completion
	ConfirmPaymentIntent(ctx context.Context, paymentIntentID string) (*PaymentIntentResponse, error)

	// CreateCustomer creates a Stripe Customer
	CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*CustomerResponse, error)

	// CreatePortalSession creates a Stripe Customer Portal session
	CreatePortalSession(ctx context.Context, req *PortalSessionRequest) (*PortalSessionResponse, error)

	// ListPaymentMethods lists saved payment methods for a customer
	ListPaymentMethods(ctx context.Context, customerID string) ([]*PaymentMethodInfo, error)

	// Name returns the gateway name
	Name() string
}

// ChargeRequest represents a charge request
type ChargeRequest struct {
	PaymentID   string
	Amount      float64
	Currency    string
	Method      string
	Description string
	Metadata    map[string]string

	// Card details (for direct card payments)
	CardToken string

	// Customer info
	CustomerID    string
	CustomerEmail string
}

// ChargeResponse represents a charge response
type ChargeResponse struct {
	Success       bool
	TransactionID string
	Status        string
	FailureReason string
	FailureCode   string
	Metadata      map[string]string
}

// TransactionInfo represents transaction details
type TransactionInfo struct {
	TransactionID string
	Status        string
	Amount        float64
	Currency      string
	Method        string
	CreatedAt     string
	Metadata      map[string]string
}

// GatewayConfig holds common gateway configuration
type GatewayConfig struct {
	APIKey        string
	SecretKey     string
	WebhookSecret string
	Environment   string // "test" or "live"
}

// PaymentIntentRequest represents a request to create a PaymentIntent
type PaymentIntentRequest struct {
	PaymentID     string
	Amount        float64
	Currency      string
	Description   string
	Metadata      map[string]string
	CustomerEmail string
}

// PaymentIntentResponse represents a PaymentIntent response
type PaymentIntentResponse struct {
	PaymentIntentID string
	ClientSecret    string
	Status          string
	Amount          float64
	Currency        string
}

// CreateCustomerRequest represents a request to create a Stripe Customer
type CreateCustomerRequest struct {
	UserID   string
	Email    string
	Name     string
	Metadata map[string]string
}

// CustomerResponse represents a Stripe Customer response
type CustomerResponse struct {
	CustomerID string
	Email      string
	Name       string
}

// PortalSessionRequest represents a request to create a Customer Portal session
type PortalSessionRequest struct {
	CustomerID string
	ReturnURL  string
}

// PortalSessionResponse represents a Customer Portal session response
type PortalSessionResponse struct {
	URL string
}

// PaymentMethodInfo represents a saved payment method
type PaymentMethodInfo struct {
	ID        string `json:"id"`
	Type      string `json:"type"`       // "card", "promptpay", etc.
	Brand     string `json:"brand"`      // "visa", "mastercard", etc.
	Last4     string `json:"last4"`      // Last 4 digits
	ExpMonth  int64  `json:"exp_month"`  // Expiration month
	ExpYear   int64  `json:"exp_year"`   // Expiration year
	IsDefault bool   `json:"is_default"` // Is default payment method
}
