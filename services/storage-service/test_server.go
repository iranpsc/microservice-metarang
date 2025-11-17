package main

import (
	"log"
	"os"

	"metargb/storage-service/internal/ftp"
	"metargb/storage-service/internal/handler"
	"metargb/storage-service/internal/service"
)

func main() {
	log.Println("🚀 Starting Storage Service Test Server...")

	// Initialize Mock FTP client (saves files locally for testing)
	uploadDir := getEnv("UPLOAD_DIR", "/tmp/storage-uploads")
	ftpClient := ftp.NewMockFTPClient(
		uploadDir,
		"http://localhost:8059/uploads",
	)
	log.Printf("✅ Mock FTP client initialized. Files will be saved to: %s", uploadDir)

	// Initialize chunk manager
	tempDir := getEnv("TEMP_DIR", "/tmp/storage-chunks")
	chunkManager, err := service.NewChunkManager(tempDir)
	if err != nil {
		log.Fatalf("Failed to initialize chunk manager: %v", err)
	}
	log.Printf("✅ Chunk manager initialized with temp directory: %s", tempDir)

	// Initialize storage service
	storageService := service.NewStorageService(ftpClient, chunkManager)

	// Create HTTP handler
	httpHandler := handler.NewHTTPHandler(storageService)

	// Start HTTP server
	httpPort := getEnv("HTTP_PORT", "8059")
	log.Printf("✅ HTTP server listening on port %s", httpPort)
	log.Printf("📤 Chunk upload endpoint: http://localhost:%s/upload", httpPort)
	log.Printf("📤 API upload endpoint: http://localhost:%s/api/upload", httpPort)
	log.Printf("🏥 Health check: http://localhost:%s/health", httpPort)
	log.Println("")
	log.Println("Ready to accept uploads! Press Ctrl+C to stop.")

	if err := handler.StartHTTPServer(httpHandler, httpPort); err != nil {
		log.Fatalf("Failed to serve HTTP: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
