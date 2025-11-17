package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TestConfig holds configuration for integration tests
type TestConfig struct {
	DBHost     string
	DBPort     string
	DBDatabase string
	DBUser     string
	DBPassword string
	AuthURL    string
	CommURL    string
	FeatURL    string
	LevelsURL  string
}

// GetTestConfig returns test configuration from environment
func GetTestConfig() *TestConfig {
	return &TestConfig{
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBDatabase: getEnv("DB_DATABASE", "metargb_test"),
		DBUser:     getEnv("DB_USER", "test_user"),
		DBPassword: getEnv("DB_PASSWORD", "test_password"),
		AuthURL:    getEnv("AUTH_SERVICE_URL", "localhost:50051"),
		CommURL:    getEnv("COMM_SERVICE_URL", "localhost:50052"),
		FeatURL:    getEnv("FEATURES_SERVICE_URL", "localhost:50053"),
		LevelsURL:  getEnv("LEVELS_SERVICE_URL", "localhost:50054"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ConnectDB creates a database connection for testing
func (cfg *TestConfig) ConnectDB(t *testing.T) *sql.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBDatabase)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	return db
}

// ConnectGRPC creates a gRPC client connection
func ConnectGRPC(t *testing.T, address string) *grpc.ClientConn {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("Failed to connect to gRPC service at %s: %v", address, err)
	}

	return conn
}

// CleanupDB cleans up test data from database
func CleanupDB(t *testing.T, db *sql.DB, tables ...string) {
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE created_at > ?", table), time.Now().Add(-1*time.Hour))
		if err != nil {
			t.Logf("Warning: failed to cleanup table %s: %v", table, err)
		}
	}
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, db *sql.DB, username, email string) int64 {
	result, err := db.Exec(`
		INSERT INTO users (username, email, password, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
	`, username, email, "$2a$10$test_hash")

	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	id, _ := result.LastInsertId()
	return id
}

// CreateTestWallet creates a test wallet for a user
func CreateTestWallet(t *testing.T, db *sql.DB, userID int64, psc, rgb string) {
	_, err := db.Exec(`
		INSERT INTO wallets (user_id, psc, rgb, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
	`, userID, psc, rgb)

	if err != nil {
		t.Fatalf("Failed to create test wallet: %v", err)
	}
}

// CreateTestFeature creates a test feature in the database
func CreateTestFeature(t *testing.T, db *sql.DB, userID *int64) string {
	featureID := fmt.Sprintf("F-%d", time.Now().UnixNano())
	
	_, err := db.Exec(`
		INSERT INTO features (id, user_id, status, created_at, updated_at)
		VALUES (?, ?, 'active', NOW(), NOW())
	`, featureID, userID)

	if err != nil {
		t.Fatalf("Failed to create test feature: %v", err)
	}

	return featureID
}

// AssertDBValue asserts a value exists in the database
func AssertDBValue(t *testing.T, db *sql.DB, query string, expected interface{}, args ...interface{}) {
	var actual interface{}
	err := db.QueryRow(query, args...).Scan(&actual)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if actual != expected {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// WaitForService waits for a service to be ready
func WaitForService(address string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for service at %s", address)
		case <-ticker.C:
			conn, err := grpc.DialContext(ctx, address,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			if err == nil {
				conn.Close()
				return nil
			}
		}
	}
}

