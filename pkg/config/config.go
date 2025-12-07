package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	MongoDB  MongoDBConfig  `mapstructure:"mongodb"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	OTel     OTelConfig     `mapstructure:"otel"`
}

// AppConfig holds application-level settings
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"` // development, staging, production
	Debug       bool   `mapstructure:"debug"`
	Version     string `mapstructure:"version"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// DatabaseConfig holds PostgreSQL connection settings
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

// DSN returns the PostgreSQL connection string
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// Addr returns the Redis address
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// KafkaConfig holds Kafka/Redpanda connection settings
type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	ConsumerGroup string   `mapstructure:"consumer_group"`
	ClientID      string   `mapstructure:"client_id"`
}

// MongoDBConfig holds MongoDB connection settings
type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

// JWTConfig holds JWT settings
type JWTConfig struct {
	Secret           string        `mapstructure:"secret"`
	AccessTokenTTL   time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL  time.Duration `mapstructure:"refresh_token_ttl"`
	Issuer           string        `mapstructure:"issuer"`
}

// OTelConfig holds OpenTelemetry settings
type OTelConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	ServiceName   string `mapstructure:"service_name"`
	CollectorAddr string `mapstructure:"collector_addr"`
}

// Load loads configuration from environment variables and .env file
func Load() (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigFile(".env")
	v.SetConfigType("env")

	// Read from .env file (optional)
	if err := v.ReadInConfig(); err != nil {
		// It's okay if .env doesn't exist, we'll use environment variables
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// Only log if it's not a "file not found" error
			// We still continue because env vars might be set
		}
	}

	// Enable environment variable override
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults(v)

	cfg := &Config{}
	if err := bindConfig(v, cfg); err != nil {
		return nil, fmt.Errorf("failed to bind config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// LoadWithPath loads configuration from a specific path
func LoadWithPath(path string) (*Config, error) {
	v := viper.New()

	v.SetConfigFile(path)
	v.SetConfigType("env")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(v)

	cfg := &Config{}
	if err := bindConfig(v, cfg); err != nil {
		return nil, fmt.Errorf("failed to bind config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("APP_NAME", "booking-rush")
	v.SetDefault("APP_ENVIRONMENT", "development")
	v.SetDefault("APP_DEBUG", true)
	v.SetDefault("APP_VERSION", "1.0.0")

	// Server defaults
	v.SetDefault("SERVER_HOST", "0.0.0.0")
	v.SetDefault("SERVER_PORT", 8080)
	v.SetDefault("SERVER_READ_TIMEOUT", "30s")
	v.SetDefault("SERVER_WRITE_TIMEOUT", "30s")
	v.SetDefault("SERVER_IDLE_TIMEOUT", "120s")

	// Database defaults
	v.SetDefault("DATABASE_HOST", "localhost")
	v.SetDefault("DATABASE_PORT", 5432)
	v.SetDefault("DATABASE_USER", "postgres")
	v.SetDefault("DATABASE_PASSWORD", "postgres")
	v.SetDefault("DATABASE_DBNAME", "booking_rush")
	v.SetDefault("DATABASE_SSLMODE", "disable")
	v.SetDefault("DATABASE_MAX_OPEN_CONNS", 100)
	v.SetDefault("DATABASE_MAX_IDLE_CONNS", 10)
	v.SetDefault("DATABASE_CONN_MAX_LIFETIME", "1h")
	v.SetDefault("DATABASE_CONN_MAX_IDLE_TIME", "30m")

	// Redis defaults
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 0)
	v.SetDefault("REDIS_POOL_SIZE", 100)
	v.SetDefault("REDIS_MIN_IDLE_CONNS", 10)
	v.SetDefault("REDIS_DIAL_TIMEOUT", "5s")
	v.SetDefault("REDIS_READ_TIMEOUT", "3s")
	v.SetDefault("REDIS_WRITE_TIMEOUT", "3s")

	// Kafka defaults
	v.SetDefault("KAFKA_BROKERS", "localhost:9092")
	v.SetDefault("KAFKA_CONSUMER_GROUP", "booking-rush")
	v.SetDefault("KAFKA_CLIENT_ID", "booking-rush")

	// MongoDB defaults
	v.SetDefault("MONGODB_URI", "mongodb://localhost:27017")
	v.SetDefault("MONGODB_DATABASE", "booking_rush")

	// JWT defaults
	v.SetDefault("JWT_SECRET", "your-secret-key-change-in-production")
	v.SetDefault("JWT_ACCESS_TOKEN_TTL", "15m")
	v.SetDefault("JWT_REFRESH_TOKEN_TTL", "168h") // 7 days
	v.SetDefault("JWT_ISSUER", "booking-rush")

	// OTel defaults
	v.SetDefault("OTEL_ENABLED", true)
	v.SetDefault("OTEL_SERVICE_NAME", "booking-rush")
	v.SetDefault("OTEL_COLLECTOR_ADDR", "localhost:4317")
}

func bindConfig(v *viper.Viper, cfg *Config) error {
	// App
	cfg.App.Name = v.GetString("APP_NAME")
	cfg.App.Environment = v.GetString("APP_ENVIRONMENT")
	cfg.App.Debug = v.GetBool("APP_DEBUG")
	cfg.App.Version = v.GetString("APP_VERSION")

	// Server
	cfg.Server.Host = v.GetString("SERVER_HOST")
	cfg.Server.Port = v.GetInt("SERVER_PORT")
	cfg.Server.ReadTimeout = v.GetDuration("SERVER_READ_TIMEOUT")
	cfg.Server.WriteTimeout = v.GetDuration("SERVER_WRITE_TIMEOUT")
	cfg.Server.IdleTimeout = v.GetDuration("SERVER_IDLE_TIMEOUT")

	// Database
	cfg.Database.Host = v.GetString("DATABASE_HOST")
	cfg.Database.Port = v.GetInt("DATABASE_PORT")
	cfg.Database.User = v.GetString("DATABASE_USER")
	cfg.Database.Password = v.GetString("DATABASE_PASSWORD")
	cfg.Database.DBName = v.GetString("DATABASE_DBNAME")
	cfg.Database.SSLMode = v.GetString("DATABASE_SSLMODE")
	cfg.Database.MaxOpenConns = v.GetInt("DATABASE_MAX_OPEN_CONNS")
	cfg.Database.MaxIdleConns = v.GetInt("DATABASE_MAX_IDLE_CONNS")
	cfg.Database.ConnMaxLifetime = v.GetDuration("DATABASE_CONN_MAX_LIFETIME")
	cfg.Database.ConnMaxIdleTime = v.GetDuration("DATABASE_CONN_MAX_IDLE_TIME")

	// Redis
	cfg.Redis.Host = v.GetString("REDIS_HOST")
	cfg.Redis.Port = v.GetInt("REDIS_PORT")
	cfg.Redis.Password = v.GetString("REDIS_PASSWORD")
	cfg.Redis.DB = v.GetInt("REDIS_DB")
	cfg.Redis.PoolSize = v.GetInt("REDIS_POOL_SIZE")
	cfg.Redis.MinIdleConns = v.GetInt("REDIS_MIN_IDLE_CONNS")
	cfg.Redis.DialTimeout = v.GetDuration("REDIS_DIAL_TIMEOUT")
	cfg.Redis.ReadTimeout = v.GetDuration("REDIS_READ_TIMEOUT")
	cfg.Redis.WriteTimeout = v.GetDuration("REDIS_WRITE_TIMEOUT")

	// Kafka
	brokersStr := v.GetString("KAFKA_BROKERS")
	cfg.Kafka.Brokers = strings.Split(brokersStr, ",")
	cfg.Kafka.ConsumerGroup = v.GetString("KAFKA_CONSUMER_GROUP")
	cfg.Kafka.ClientID = v.GetString("KAFKA_CLIENT_ID")

	// MongoDB
	cfg.MongoDB.URI = v.GetString("MONGODB_URI")
	cfg.MongoDB.Database = v.GetString("MONGODB_DATABASE")

	// JWT
	cfg.JWT.Secret = v.GetString("JWT_SECRET")
	cfg.JWT.AccessTokenTTL = v.GetDuration("JWT_ACCESS_TOKEN_TTL")
	cfg.JWT.RefreshTokenTTL = v.GetDuration("JWT_REFRESH_TOKEN_TTL")
	cfg.JWT.Issuer = v.GetString("JWT_ISSUER")

	// OTel
	cfg.OTel.Enabled = v.GetBool("OTEL_ENABLED")
	cfg.OTel.ServiceName = v.GetString("OTEL_SERVICE_NAME")
	cfg.OTel.CollectorAddr = v.GetString("OTEL_COLLECTOR_ADDR")

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app name is required")
	}

	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if c.Database.DBName == "" {
		return fmt.Errorf("database name is required")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	// Warn if using default JWT secret in production
	if c.App.Environment == "production" && c.JWT.Secret == "your-secret-key-change-in-production" {
		return fmt.Errorf("JWT secret must be changed in production")
	}

	return nil
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}
