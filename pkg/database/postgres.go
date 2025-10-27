package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/lib/pq"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config holds database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ServiceName     string // For OTEL instrumentation
}

// LoadConfigFromEnv loads database configuration from environment variables
func LoadConfigFromEnv(serviceName string) *Config {
	return &Config{
		Host:            getEnv("DB_HOST", "postgres"),
		Port:            getEnvAsInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "docutab"),
		Password:        getEnv("DB_PASSWORD", "docutab_dev_pass"),
		Database:        getEnv("DB_NAME", "docutab"),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		ServiceName:     serviceName,
	}
}

// NewPostgresDB creates a new PostgreSQL connection with OTEL instrumentation
func NewPostgresDB(ctx context.Context, config *Config) (*sql.DB, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
	)

	log.Printf("Connecting to PostgreSQL: host=%s port=%d dbname=%s", config.Host, config.Port, config.Database)

	// Register the instrumented driver
	driverName, err := otelsql.Register(
		"postgres",
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			DisableErrSkip:  true,
			RecordError:     otelsql.RecordErrorFunc(shouldRecordError),
			OmitRows:        false,
			OmitConnReuse:   false,
			OmitConnPrepare: false,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register OTEL driver: %w", err)
	}

	// Open database connection with instrumented driver
	db, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	log.Println("Testing database connection...")
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Record database metrics with OTEL
	if err := otelsql.RecordStats(db, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	)); err != nil {
		log.Printf("Warning: failed to record database stats: %v", err)
	}

	log.Println("Database connection established successfully")
	return db, nil
}

// shouldRecordError determines if an error should be recorded in traces
// This helps reduce noise from expected errors like "no rows"
func shouldRecordError(err error) bool {
	if err == nil {
		return false
	}
	// Don't record sql.ErrNoRows as it's an expected condition
	return err != sql.ErrNoRows
}

// Helper functions for environment variable parsing
func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultVal
}
