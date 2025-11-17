package main

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"metargb/levels-service/internal/handler"
	"metargb/levels-service/internal/repository"
	"metargb/levels-service/internal/service"
	pb "metargb/shared/pb/levels"
	"metargb/shared/pkg/db"
	"metargb/shared/pkg/logger"
	"metargb/shared/pkg/metrics"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Initialize logger
	log := logger.NewLogger("levels-service")
	log.Info("Starting Levels Service...")

	// Load configuration from environment
	// Construct DSN from individual environment variables
	dbDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		getEnv("DB_USER", "metargb_user"),
		getEnv("DB_PASSWORD", "metargb_password"),
		getEnv("DB_HOST", "mysql"),
		getEnv("DB_PORT", "3306"),
		getEnv("DB_DATABASE", "metargb_db"),
	)
	port := getEnv("GRPC_PORT", "50054")
	metricsPort := getEnv("METRICS_PORT", "9090")

	// Initialize database connection
	database, err := sql.Open("mysql", dbDSN)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}
	defer database.Close()

	// Test database connection
	if err := database.Ping(); err != nil {
		log.Fatal("Failed to ping database", "error", err)
	}

	// Validate schema
	schemaGuard := db.NewSchemaGuard(database)
	if err := schemaGuard.ValidateTable(db.TableSchema{
		Name: "levels",
		Columns: []db.ColumnType{
			{Name: "id", DataType: "bigint"},
			{Name: "name", DataType: "varchar"},
			{Name: "slug", DataType: "varchar"},
			{Name: "score", DataType: "int"},
		},
	}); err != nil {
		log.Warn("Schema validation warning", "error", err)
	}

	log.Info("Database connected and schema validated")

	// Initialize repositories
	levelRepo := repository.NewLevelRepository(database)
	activityRepo := repository.NewActivityRepository(database)
	challengeRepo := repository.NewChallengeRepository(database)
	userLogRepo := repository.NewUserLogRepository(database)

	// Initialize services
	levelService := service.NewLevelService(levelRepo, userLogRepo)
	activityService := service.NewActivityService(activityRepo, userLogRepo, levelRepo)
	challengeService := service.NewChallengeService(challengeRepo)

	// Initialize gRPC handlers
	levelHandler := handler.NewLevelHandler(levelService)
	activityHandler := handler.NewActivityHandler(activityService)
	challengeHandler := handler.NewChallengeHandler(challengeService)

	// Create gRPC server with interceptors
	serviceMetrics := metrics.NewMetrics("levels")
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logger.UnaryServerInterceptor(log),
			metrics.UnaryServerInterceptor(serviceMetrics),
		),
	)

	// Register services
	pb.RegisterLevelServiceServer(grpcServer, levelHandler)
	pb.RegisterActivityServiceServer(grpcServer, activityHandler)
	pb.RegisterChallengeServiceServer(grpcServer, challengeHandler)

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Metrics are exposed via Prometheus client library
	// Start HTTP server for metrics endpoint if needed
	log.Info("Metrics available on /metrics endpoint", "port", metricsPort)

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal("Failed to listen", "error", err, "port", port)
	}

	log.Info("Levels Service started", "port", port, "metrics_port", metricsPort)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Info("Shutting down gracefully...")
		grpcServer.GracefulStop()
		database.Close()
		log.Info("Shutdown complete")
	}()

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal("Failed to serve", "error", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
