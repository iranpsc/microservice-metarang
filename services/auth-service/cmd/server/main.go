package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"metargb/auth-service/internal/handler"
	"metargb/auth-service/internal/pubsub"
	"metargb/auth-service/internal/repository"
	"metargb/auth-service/internal/service"
	notificationspb "metargb/shared/pb/notifications"
)

func main() {
	// Load environment variables from config.env
	if err := godotenv.Load("config.env"); err != nil {
		// Fallback to .env if config.env not found
		if err2 := godotenv.Load(); err2 != nil {
			log.Printf("Warning: config.env and .env files not found: %v, %v", err, err2)
		}
	}

	// Database connection
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		getEnv("DB_USER", "root"),
		getEnv("DB_PASSWORD", ""),
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "3306"),
		getEnv("DB_DATABASE", "metargb_db"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to database")

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	accountSecurityRepo := repository.NewAccountSecurityRepository(db)
	kycRepo := repository.NewKYCRepository(db)
	activityRepo := repository.NewActivityRepository(db)

	// Initialize Redis publisher for WebSocket broadcasting
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379/0")
	redisPublisher, err := pubsub.NewRedisPublisher(redisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// Initialize observer service for activity tracking and events
	observerService := service.NewObserverService(
		userRepo,
		activityRepo,
		redisPublisher,
	)

	// Initialize helper service for cross-service integrations
	helperService := service.NewHelperService(
		getEnv("LEVELS_SERVICE_ADDR", "levels-service:50051"),
		getEnv("FEATURES_SERVICE_ADDR", "features-service:50051"),
	)

	// Initialize notifications SMS client (optional - service can work without it)
	var smsClient notificationspb.SMSServiceClient
	notificationsAddr := getEnv("NOTIFICATIONS_SERVICE_ADDR", "notifications-service:50051")
	notificationsConn, err := grpc.Dial(notificationsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Warning: Failed to connect to notifications service: %v (continuing without SMS support)", err)
	} else {
		defer notificationsConn.Close()
		smsClient = notificationspb.NewSMSServiceClient(notificationsConn)
		log.Println("Successfully connected to notifications service")
	}

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		tokenRepo,
		accountSecurityRepo,
		activityRepo,
		observerService,
		helperService,
		smsClient,
		getEnv("OAUTH_SERVER_URL", ""),
		getEnv("OAUTH_CLIENT_ID", ""),
		getEnv("OAUTH_CLIENT_SECRET", ""),
		getEnv("APP_URL", "http://localhost:8000"),
		getEnv("FRONT_END_URL", "http://localhost:3000"),
	)
	userService := service.NewUserService(userRepo)
	kycService := service.NewKYCService(kycRepo)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register handlers
	handler.RegisterAuthHandler(grpcServer, authService, tokenRepo)
	handler.RegisterUserHandler(grpcServer, userService)
	handler.RegisterKYCHandler(grpcServer, kycService)

	// Start gRPC server
	port := getEnv("GRPC_PORT", "50051")
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	log.Printf("Auth service listening on port %s", port)

	// Graceful shutdown
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	grpcServer.GracefulStop()
	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
