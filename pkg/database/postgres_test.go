package database

import (
	"context"
	"os"
	"testing"
	"time"
)

// getTestConfig returns config for testing
// Uses environment variables or defaults
func getTestConfig() *PostgresConfig {
	cfg := DefaultPostgresConfig()

	if host := os.Getenv("TEST_POSTGRES_HOST"); host != "" {
		cfg.Host = host
	}
	if port := os.Getenv("TEST_POSTGRES_PORT"); port != "" {
		// For simplicity, use default port if parsing fails
	}
	if user := os.Getenv("TEST_POSTGRES_USER"); user != "" {
		cfg.User = user
	}
	if password := os.Getenv("TEST_POSTGRES_PASSWORD"); password != "" {
		cfg.Password = password
	}
	if dbname := os.Getenv("TEST_POSTGRES_DATABASE"); dbname != "" {
		cfg.Database = dbname
	}

	return cfg
}

// skipIfNoDatabase skips test if database is not available
func skipIfNoDatabase(t *testing.T, db *PostgresDB) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
}

func TestDefaultPostgresConfig(t *testing.T) {
	cfg := DefaultPostgresConfig()

	if cfg.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", cfg.Port)
	}
	if cfg.MaxConns != 25 {
		t.Errorf("Expected max conns 25, got %d", cfg.MaxConns)
	}
	if cfg.MinConns != 5 {
		t.Errorf("Expected min conns 5, got %d", cfg.MinConns)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", cfg.MaxRetries)
	}
}

func TestPostgresConfig_DSN(t *testing.T) {
	cfg := &PostgresConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
	}

	dsn := cfg.DSN()
	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"

	if dsn != expected {
		t.Errorf("DSN mismatch:\nExpected: %s\nGot: %s", expected, dsn)
	}
}

func TestNewPostgres_InvalidConfig(t *testing.T) {
	cfg := &PostgresConfig{
		Host:           "invalid-host-that-does-not-exist",
		Port:           9999,
		User:           "invalid",
		Password:       "invalid",
		Database:       "invalid",
		SSLMode:        "disable",
		MaxRetries:     0,
		RetryInterval:  100 * time.Millisecond,
		ConnectTimeout: 1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := NewPostgres(ctx, cfg)
	if err == nil {
		t.Error("Expected error for invalid config, got nil")
	}
}

// Integration tests - run only when database is available

func TestNewPostgres_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	db, err := NewPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer db.Close()

	// Test Ping
	if err := db.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	// Test IsConnected
	if !db.IsConnected(ctx) {
		t.Error("Expected IsConnected to return true")
	}

	// Test Pool not nil
	if db.Pool() == nil {
		t.Error("Expected Pool() to return non-nil")
	}

	// Test Stats
	stats := db.Stats()
	if stats == nil {
		t.Error("Expected Stats() to return non-nil")
	}
}

func TestPostgresDB_HealthCheck_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	db, err := NewPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer db.Close()

	if err := db.HealthCheck(ctx); err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestPostgresDB_Exec_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	db, err := NewPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer db.Close()

	// Create a temporary table
	err = db.Exec(ctx, "CREATE TEMP TABLE test_table (id SERIAL PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Errorf("Exec failed: %v", err)
	}

	// Insert a row
	err = db.Exec(ctx, "INSERT INTO test_table (name) VALUES ($1)", "test")
	if err != nil {
		t.Errorf("Insert failed: %v", err)
	}

	// Query the row
	var name string
	err = db.QueryRow(ctx, "SELECT name FROM test_table WHERE name = $1", "test").Scan(&name)
	if err != nil {
		t.Errorf("QueryRow failed: %v", err)
	}
	if name != "test" {
		t.Errorf("Expected name 'test', got '%s'", name)
	}
}

func TestPostgresDB_Transaction_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	db, err := NewPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer db.Close()

	// Create temp table
	err = db.Exec(ctx, "CREATE TEMP TABLE tx_test (id SERIAL PRIMARY KEY, value INT)")
	if err != nil {
		t.Fatalf("Failed to create temp table: %v", err)
	}

	// Start transaction
	tx, err := db.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx failed: %v", err)
	}

	// Insert in transaction
	_, err = tx.Exec(ctx, "INSERT INTO tx_test (value) VALUES ($1)", 100)
	if err != nil {
		tx.Rollback(ctx)
		t.Fatalf("Insert in tx failed: %v", err)
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		t.Errorf("Commit failed: %v", err)
	}

	// Verify
	var value int
	err = db.QueryRow(ctx, "SELECT value FROM tx_test WHERE value = $1", 100).Scan(&value)
	if err != nil {
		t.Errorf("Query after commit failed: %v", err)
	}
	if value != 100 {
		t.Errorf("Expected value 100, got %d", value)
	}
}

func TestPostgresDB_Close(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	cfg := getTestConfig()
	ctx := context.Background()

	db, err := NewPostgres(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to postgres: %v", err)
	}

	// Close should not panic
	db.Close()

	// After close, ping should fail
	err = db.Ping(ctx)
	if err == nil {
		t.Error("Expected Ping to fail after Close")
	}
}
