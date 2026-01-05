package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BookingDetails contains enriched booking information for notifications
type BookingDetails struct {
	BookingID        string  `json:"booking_id"`
	TenantID         string  `json:"tenant_id"`
	UserID           string  `json:"user_id"`
	UserEmail        string  `json:"user_email"`
	EventID          string  `json:"event_id"`
	EventName        string  `json:"event_name"`
	ShowID           string  `json:"show_id"`
	ShowDate         string  `json:"show_date"`
	ZoneID           string  `json:"zone_id"`
	ZoneName         string  `json:"zone_name"`
	Quantity         int     `json:"quantity"`
	UnitPrice        float64 `json:"unit_price"`
	TotalPrice       float64 `json:"total_price"`
	Currency         string  `json:"currency"`
	Status           string  `json:"status"`
	ConfirmationCode string  `json:"confirmation_code,omitempty"`
	VenueName        string  `json:"venue_name,omitempty"`
	VenueAddress     string  `json:"venue_address,omitempty"`
}

// BookingClient is a client for the booking service
type BookingClient interface {
	// GetBookingDetails fetches enriched booking details by ID
	GetBookingDetails(ctx context.Context, bookingID string, authToken string) (*BookingDetails, error)
}

// HTTPBookingClient implements BookingClient using HTTP
type HTTPBookingClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPBookingClient creates a new HTTP booking client
func NewHTTPBookingClient(baseURL string) *HTTPBookingClient {
	return &HTTPBookingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetBookingDetails fetches enriched booking details from booking service
func (c *HTTPBookingClient) GetBookingDetails(ctx context.Context, bookingID string, authToken string) (*BookingDetails, error) {
	url := fmt.Sprintf("%s/api/v1/bookings/%s/details", c.baseURL, bookingID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch booking details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("booking service returned status %d", resp.StatusCode)
	}

	var apiResponse struct {
		Success bool            `json:"success"`
		Data    *BookingDetails `json:"data"`
		Error   string          `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResponse.Success {
		return nil, fmt.Errorf("booking service error: %s", apiResponse.Error)
	}

	return apiResponse.Data, nil
}

// NoOpBookingClient is a no-op implementation for testing or when booking service is unavailable
type NoOpBookingClient struct{}

// NewNoOpBookingClient creates a new no-op booking client
func NewNoOpBookingClient() *NoOpBookingClient {
	return &NoOpBookingClient{}
}

// GetBookingDetails returns nil (no enrichment)
func (c *NoOpBookingClient) GetBookingDetails(ctx context.Context, bookingID string, authToken string) (*BookingDetails, error) {
	return nil, nil
}
