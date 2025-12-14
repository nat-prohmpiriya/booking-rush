package config

import (
	"os"
	"testing"
)

func TestLoad_WithDefaults(t *testing.T) {
	// Clear any existing env vars that might interfere
	envVars := []string{
		"APP_NAME", "APP_ENVIRONMENT", "APP_DEBUG",
		"SERVER_HOST", "SERVER_PORT",
		"AUTH_DATABASE_HOST", "AUTH_DATABASE_PORT", "AUTH_DATABASE_USER", "AUTH_DATABASE_PASSWORD", "AUTH_DATABASE_DBNAME",
		"TICKET_DATABASE_HOST", "TICKET_DATABASE_PORT",
		"BOOKING_DATABASE_HOST", "BOOKING_DATABASE_PORT",
		"PAYMENT_DATABASE_HOST", "PAYMENT_DATABASE_PORT",
		"REDIS_HOST", "REDIS_PORT",
		"JWT_SECRET",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Check defaults
	if cfg.App.Name != "booking-rush" {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, "booking-rush")
	}

	if cfg.App.Environment != "development" {
		t.Errorf("App.Environment = %q, want %q", cfg.App.Environment, "development")
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}

	if cfg.AuthDatabase.Port != 5432 {
		t.Errorf("AuthDatabase.Port = %d, want %d", cfg.AuthDatabase.Port, 5432)
	}

	if cfg.TicketDatabase.Port != 5432 {
		t.Errorf("TicketDatabase.Port = %d, want %d", cfg.TicketDatabase.Port, 5432)
	}

	if cfg.BookingDatabase.Port != 5432 {
		t.Errorf("BookingDatabase.Port = %d, want %d", cfg.BookingDatabase.Port, 5432)
	}

	if cfg.PaymentDatabase.Port != 5432 {
		t.Errorf("PaymentDatabase.Port = %d, want %d", cfg.PaymentDatabase.Port, 5432)
	}

	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port = %d, want %d", cfg.Redis.Port, 6379)
	}
}

func TestLoad_WithEnvOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("APP_NAME", "test-app")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("AUTH_DATABASE_HOST", "auth-db.example.com")
	os.Setenv("TICKET_DATABASE_HOST", "ticket-db.example.com")
	defer func() {
		os.Unsetenv("APP_NAME")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("AUTH_DATABASE_HOST")
		os.Unsetenv("TICKET_DATABASE_HOST")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.App.Name != "test-app" {
		t.Errorf("App.Name = %q, want %q", cfg.App.Name, "test-app")
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 9090)
	}

	if cfg.AuthDatabase.Host != "auth-db.example.com" {
		t.Errorf("AuthDatabase.Host = %q, want %q", cfg.AuthDatabase.Host, "auth-db.example.com")
	}

	if cfg.TicketDatabase.Host != "ticket-db.example.com" {
		t.Errorf("TicketDatabase.Host = %q, want %q", cfg.TicketDatabase.Host, "ticket-db.example.com")
	}
}

func TestDatabaseConfig_DSN(t *testing.T) {
	cfg := DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	expected := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
	if dsn := cfg.DSN(); dsn != expected {
		t.Errorf("DSN() = %q, want %q", dsn, expected)
	}
}

func TestRedisConfig_Addr(t *testing.T) {
	cfg := RedisConfig{
		Host: "redis.example.com",
		Port: 6380,
	}

	expected := "redis.example.com:6380"
	if addr := cfg.Addr(); addr != expected {
		t.Errorf("Addr() = %q, want %q", addr, expected)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				App:    AppConfig{Name: "test", Environment: "development"},
				Server: ServerConfig{Port: 8080},
				JWT:    JWTConfig{Secret: "secret"},
			},
			wantErr: false,
		},
		{
			name: "missing app name",
			cfg: Config{
				App:    AppConfig{Name: "", Environment: "development"},
				Server: ServerConfig{Port: 8080},
				JWT:    JWTConfig{Secret: "secret"},
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			cfg: Config{
				App:    AppConfig{Name: "test", Environment: "development"},
				Server: ServerConfig{Port: -1},
				JWT:    JWTConfig{Secret: "secret"},
			},
			wantErr: true,
		},
		{
			name: "port too high",
			cfg: Config{
				App:    AppConfig{Name: "test", Environment: "development"},
				Server: ServerConfig{Port: 70000},
				JWT:    JWTConfig{Secret: "secret"},
			},
			wantErr: true,
		},
		{
			name: "missing JWT secret",
			cfg: Config{
				App:    AppConfig{Name: "test", Environment: "development"},
				Server: ServerConfig{Port: 8080},
				JWT:    JWTConfig{Secret: ""},
			},
			wantErr: true,
		},
		{
			name: "default JWT secret in production",
			cfg: Config{
				App:    AppConfig{Name: "test", Environment: "production"},
				Server: ServerConfig{Port: 8080},
				JWT:    JWTConfig{Secret: "your-secret-key-change-in-production"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ValidateAuthDatabase(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid auth database",
			cfg: Config{
				AuthDatabase: DatabaseConfig{Host: "localhost", DBName: "auth_db"},
			},
			wantErr: false,
		},
		{
			name: "missing auth database host",
			cfg: Config{
				AuthDatabase: DatabaseConfig{Host: "", DBName: "auth_db"},
			},
			wantErr: true,
		},
		{
			name: "missing auth database name",
			cfg: Config{
				AuthDatabase: DatabaseConfig{Host: "localhost", DBName: ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateAuthDatabase()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAuthDatabase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ValidateTicketDatabase(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid ticket database",
			cfg: Config{
				TicketDatabase: DatabaseConfig{Host: "localhost", DBName: "ticket_db"},
			},
			wantErr: false,
		},
		{
			name: "missing ticket database host",
			cfg: Config{
				TicketDatabase: DatabaseConfig{Host: "", DBName: "ticket_db"},
			},
			wantErr: true,
		},
		{
			name: "missing ticket database name",
			cfg: Config{
				TicketDatabase: DatabaseConfig{Host: "localhost", DBName: ""},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateTicketDatabase()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTicketDatabase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ValidateBookingDatabase(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid booking database",
			cfg: Config{
				BookingDatabase: DatabaseConfig{Host: "localhost", DBName: "booking_db"},
			},
			wantErr: false,
		},
		{
			name: "missing booking database host",
			cfg: Config{
				BookingDatabase: DatabaseConfig{Host: "", DBName: "booking_db"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidateBookingDatabase()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBookingDatabase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_ValidatePaymentDatabase(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid payment database",
			cfg: Config{
				PaymentDatabase: DatabaseConfig{Host: "localhost", DBName: "payment_db"},
			},
			wantErr: false,
		},
		{
			name: "missing payment database host",
			cfg: Config{
				PaymentDatabase: DatabaseConfig{Host: "", DBName: "payment_db"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.ValidatePaymentDatabase()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePaymentDatabase() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	cfg := &Config{
		App: AppConfig{Environment: "production"},
	}
	if !cfg.IsProduction() {
		t.Error("IsProduction() = false, want true")
	}

	cfg.App.Environment = "development"
	if cfg.IsProduction() {
		t.Error("IsProduction() = true, want false")
	}
}

func TestConfig_IsDevelopment(t *testing.T) {
	cfg := &Config{
		App: AppConfig{Environment: "development"},
	}
	if !cfg.IsDevelopment() {
		t.Error("IsDevelopment() = false, want true")
	}

	cfg.App.Environment = "production"
	if cfg.IsDevelopment() {
		t.Error("IsDevelopment() = true, want false")
	}
}
