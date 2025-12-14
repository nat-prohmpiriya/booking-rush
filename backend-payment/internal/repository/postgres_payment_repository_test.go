package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/prohmpiriya/booking-rush-10k-rps/backend-payment/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/database"
)

func skipIfNoIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}
}

func setupTestDB(t *testing.T) *database.PostgresDB {
	ctx := context.Background()

	cfg := &database.PostgresConfig{
		Host:            getEnv("POSTGRES_HOST", "100.104.0.42"),
		Port:            5432,
		User:            getEnv("POSTGRES_USER", "postgres"),
		Password:        getEnv("POSTGRES_PASSWORD", ""),
		Database:        getEnv("POSTGRES_DB", "booking_rush"),
		SSLMode:         "disable",
		MaxConns:        5,
		MinConns:        1,
		MaxConnLifetime: 5 * time.Minute,
		MaxConnIdleTime: 1 * time.Minute,
		ConnectTimeout:  5 * time.Second,
		MaxRetries:      3,
		RetryInterval:   1 * time.Second,
	}

	db, err := database.NewPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	return db
}

func cleanupTestData(t *testing.T, db *database.PostgresDB) {
	ctx := context.Background()
	_, err := db.Pool().Exec(ctx, "DELETE FROM payments WHERE booking_id LIKE 'test-booking-%'")
	if err != nil {
		t.Logf("Warning: failed to cleanup test data: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func TestPostgresPaymentRepository_Create(t *testing.T) {
	skipIfNoIntegration(t)

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresPaymentRepository(db)
	ctx := context.Background()

	payment, err := domain.NewPayment("tenant-123", "test-booking-create", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)
	if err != nil {
		t.Fatalf("Failed to create payment: %v", err)
	}

	err = repo.Create(ctx, payment)
	if err != nil {
		t.Fatalf("Failed to create payment in DB: %v", err)
	}

	// Verify it was created
	found, err := repo.GetByID(ctx, payment.ID)
	if err != nil {
		t.Fatalf("Failed to get payment: %v", err)
	}

	if found.ID != payment.ID {
		t.Errorf("Expected ID %s, got %s", payment.ID, found.ID)
	}

	if found.BookingID != payment.BookingID {
		t.Errorf("Expected BookingID %s, got %s", payment.BookingID, found.BookingID)
	}

	if found.Amount != payment.Amount {
		t.Errorf("Expected Amount %f, got %f", payment.Amount, found.Amount)
	}
}

func TestPostgresPaymentRepository_Create_Duplicate(t *testing.T) {
	skipIfNoIntegration(t)

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresPaymentRepository(db)
	ctx := context.Background()

	payment1, _ := domain.NewPayment("tenant-123", "test-booking-dup", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)
	payment2, _ := domain.NewPayment("tenant-123", "test-booking-dup", "user-789", 500.00, "THB", domain.PaymentMethodDebitCard)

	err := repo.Create(ctx, payment1)
	if err != nil {
		t.Fatalf("Failed to create first payment: %v", err)
	}

	err = repo.Create(ctx, payment2)
	if err != domain.ErrPaymentAlreadyExists {
		t.Errorf("Expected ErrPaymentAlreadyExists, got %v", err)
	}
}

func TestPostgresPaymentRepository_GetByBookingID(t *testing.T) {
	skipIfNoIntegration(t)

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresPaymentRepository(db)
	ctx := context.Background()

	payment, _ := domain.NewPayment("tenant-123", "test-booking-get", "user-456", 1500.00, "THB", domain.PaymentMethodCreditCard)
	repo.Create(ctx, payment)

	found, err := repo.GetByBookingID(ctx, "test-booking-get")
	if err != nil {
		t.Fatalf("Failed to get payment by booking ID: %v", err)
	}

	if found.BookingID != "test-booking-get" {
		t.Errorf("Expected BookingID 'test-booking-get', got '%s'", found.BookingID)
	}
}

func TestPostgresPaymentRepository_GetByUserID(t *testing.T) {
	skipIfNoIntegration(t)

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresPaymentRepository(db)
	ctx := context.Background()

	// Create multiple payments for the same user
	testUserID := "test-user-list"
	for i := 0; i < 3; i++ {
		payment, _ := domain.NewPayment(
			"tenant-123",
			"test-booking-user-"+string(rune('A'+i)),
			testUserID,
			float64(100*(i+1)),
			"THB",
			domain.PaymentMethodCreditCard,
		)
		repo.Create(ctx, payment)
	}

	payments, err := repo.GetByUserID(ctx, testUserID, 10, 0)
	if err != nil {
		t.Fatalf("Failed to get payments by user ID: %v", err)
	}

	if len(payments) != 3 {
		t.Errorf("Expected 3 payments, got %d", len(payments))
	}
}

func TestPostgresPaymentRepository_Update(t *testing.T) {
	skipIfNoIntegration(t)

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresPaymentRepository(db)
	ctx := context.Background()

	payment, _ := domain.NewPayment("tenant-123", "test-booking-update", "user-456", 2000.00, "THB", domain.PaymentMethodCreditCard)
	repo.Create(ctx, payment)

	// Update payment status
	payment.Complete("pi_test_123")

	err := repo.Update(ctx, payment)
	if err != nil {
		t.Fatalf("Failed to update payment: %v", err)
	}

	// Verify update
	found, _ := repo.GetByID(ctx, payment.ID)
	if found.Status != domain.PaymentStatusSucceeded {
		t.Errorf("Expected status 'succeeded', got '%s'", found.Status)
	}

	if found.GatewayPaymentID != "pi_test_123" {
		t.Errorf("Expected GatewayPaymentID 'pi_test_123', got '%s'", found.GatewayPaymentID)
	}
}

func TestPostgresPaymentRepository_GetByGatewayPaymentID(t *testing.T) {
	skipIfNoIntegration(t)

	db := setupTestDB(t)
	defer db.Close()
	defer cleanupTestData(t, db)

	repo := NewPostgresPaymentRepository(db)
	ctx := context.Background()

	payment, _ := domain.NewPayment("tenant-123", "test-booking-txn", "user-456", 3000.00, "THB", domain.PaymentMethodCreditCard)
	repo.Create(ctx, payment)

	payment.Complete("pi_find_me_123")
	repo.Update(ctx, payment)

	found, err := repo.GetByGatewayPaymentID(ctx, "pi_find_me_123")
	if err != nil {
		t.Fatalf("Failed to get payment by gateway payment ID: %v", err)
	}

	if found.GatewayPaymentID != "pi_find_me_123" {
		t.Errorf("Expected GatewayPaymentID 'pi_find_me_123', got '%s'", found.GatewayPaymentID)
	}
}

func TestPostgresPaymentRepository_NotFound(t *testing.T) {
	skipIfNoIntegration(t)

	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresPaymentRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "non-existent-id")
	if err != domain.ErrPaymentNotFound {
		t.Errorf("Expected ErrPaymentNotFound, got %v", err)
	}

	_, err = repo.GetByBookingID(ctx, "non-existent-booking")
	if err != domain.ErrPaymentNotFound {
		t.Errorf("Expected ErrPaymentNotFound, got %v", err)
	}

	_, err = repo.GetByGatewayPaymentID(ctx, "non-existent-pi")
	if err != domain.ErrPaymentNotFound {
		t.Errorf("Expected ErrPaymentNotFound, got %v", err)
	}
}

func TestPostgresPaymentRepository_Update_NotFound(t *testing.T) {
	skipIfNoIntegration(t)

	db := setupTestDB(t)
	defer db.Close()

	repo := NewPostgresPaymentRepository(db)
	ctx := context.Background()

	payment, _ := domain.NewPayment("tenant-123", "test-booking-not-exist", "user-456", 1000.00, "THB", domain.PaymentMethodCreditCard)

	err := repo.Update(ctx, payment)
	if err != domain.ErrPaymentNotFound {
		t.Errorf("Expected ErrPaymentNotFound, got %v", err)
	}
}
