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
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"metargb/auth-service/internal/handler"
	"metargb/auth-service/internal/pubsub"
	"metargb/auth-service/internal/repository"
	"metargb/auth-service/internal/service"
	notificationspb "metargb/shared/pb/notifications"
)

func main() {
	// Panic recovery to catch any early failures
	defer func() {
		if r := recover(); r != nil {
			log.Fatalf("Panic: %v", r)
		}
	}()

	// Load environment variables from config.env
	// Try multiple possible paths for config.env
	configPaths := []string{
		"config.env",
		"./config.env",
		"../config.env",
		"../../config.env",
		"services/auth-service/config.env",
	}
	var configLoaded bool
	for _, configPath := range configPaths {
		if err := godotenv.Load(configPath); err == nil {
			configLoaded = true
			break
		}
	}
	if !configLoaded {
		// Fallback to .env if config.env not found
		if err2 := godotenv.Load(); err2 != nil {
			log.Printf("Warning: config.env and .env files not found, using environment variables only")
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
	if err := db.PingContext(ctx); err != nil {
		cancel()
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to database")

	// Initialize Redis connection for caching and pub/sub
	redisURL := getEnv("REDIS_URL", "")
	if redisURL == "" {
		// Construct REDIS_URL from individual components if not set
		redisHost := getEnv("REDIS_HOST", "localhost")
		redisPort := getEnv("REDIS_PORT", "6379")
		redisPassword := getEnv("REDIS_PASSWORD", "")
		redisDB := getEnv("REDIS_DB", "0")
		if redisPassword != "" {
			redisURL = fmt.Sprintf("redis://:%s@%s:%s/%s", redisPassword, redisHost, redisPort, redisDB)
		} else {
			redisURL = fmt.Sprintf("redis://%s:%s/%s", redisHost, redisPort, redisDB)
		}
	}

	// Parse Redis URL for cache client
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		cancel()
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	// Test Redis connection
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		cancel()
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	cancel()
	log.Println("Successfully connected to Redis")

	// Initialize Redis publisher for WebSocket broadcasting
	redisPublisher, err := pubsub.NewRedisPublisher(redisURL)
	if err != nil {
		log.Fatalf("Failed to create Redis publisher: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	cacheRepo := repository.NewCacheRepository(redisClient)
	accountSecurityRepo := repository.NewAccountSecurityRepository(db)
	kycRepo := repository.NewKYCRepository(db)
	activityRepo := repository.NewActivityRepository(db)
	citizenRepo := repository.NewCitizenRepository(db)
	personalInfoRepo := repository.NewPersonalInfoRepository(db)
	profilePhotoRepo := repository.NewProfilePhotoRepository(db)
	settingsRepo := repository.NewSettingsRepository(db)
	searchRepo := repository.NewSearchRepository(db)

	// Initialize observer service for activity tracking and events
	observerService := service.NewObserverServiceWithSettings(
		userRepo,
		settingsRepo,
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
	notificationsAddr := getEnv("NOTIFICATIONS_SERVICE_ADDR", "notifications-service:50058")
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
		cacheRepo,
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
	// Initialize user service with all dependencies for Users API
	userService := service.NewUserServiceWithDependencies(
		userRepo,
		kycRepo,
		settingsRepo,
		profilePhotoRepo,
	)
	kycService := service.NewKYCService(kycRepo, userRepo)
	citizenService := service.NewCitizenService(citizenRepo, userRepo)
	personalInfoService := service.NewPersonalInfoService(personalInfoRepo)
	profileLimitationRepo := repository.NewProfileLimitationRepository(db)
	profileLimitationService := service.NewProfileLimitationService(profileLimitationRepo, userRepo)
	settingsService := service.NewSettingsService(settingsRepo)

	// Initialize profile photo service (storage client can be added later when proto files are generated)
	// For now, service works without storage client (files can be uploaded via HTTP endpoint)
	profilePhotoService := service.NewProfilePhotoService(profilePhotoRepo, nil)

	// Initialize user events service
	userEventsService := service.NewUserEventsService(activityRepo, userRepo)

	// Initialize search service
	searchService := service.NewSearchService(searchRepo)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register handlers
	handler.RegisterAuthHandler(grpcServer, authService, tokenRepo)
	handler.RegisterUserHandler(grpcServer, userService, profileLimitationService)
	handler.RegisterKYCHandler(grpcServer, kycService)
	handler.RegisterCitizenHandler(grpcServer, citizenService)
	handler.RegisterPersonalInfoHandler(grpcServer, personalInfoService)
	handler.RegisterProfileLimitationHandler(grpcServer, profileLimitationService)
	handler.RegisterProfilePhotoHandler(grpcServer, profilePhotoService)
	handler.RegisterSettingsHandler(grpcServer, settingsService)
	handler.RegisterUserEventsHandler(grpcServer, userEventsService, userRepo)
	handler.RegisterSearchHandler(grpcServer, searchService)

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
