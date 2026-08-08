package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strings"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	"metarang/grpc-gateway/internal/config"
	"metarang/grpc-gateway/internal/handler"
	"metarang/grpc-gateway/internal/middleware"
	pb "metarang/shared/pb/auth"
	grpcutil "metarang/shared/pkg/grpc"
	"metarang/shared/pkg/sentry"
)

func main() {
	// Load environment variables
	configPaths := []string{
		"config.env",
		"./config.env",
		"../config.env",
		"../../config.env",
		"services/grpc-gateway/config.env",
	}
	var configLoaded bool
	for _, configPath := range configPaths {
		if err := godotenv.Load(configPath); err == nil {
			configLoaded = true
			log.Printf("✅ Loaded config from: %s", configPath)
			break
		}
	}
	if !configLoaded {
		log.Println("⚠️  No config.env found, using environment variables")
	}

	if err := sentry.InitFromEnv("grpc-gateway"); err != nil {
		log.Printf("Warning: failed to initialize Sentry: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	cfg := config.Load()

	// Create gRPC connections
	authConn, err := grpcutil.NewClient(
		cfg.AuthServiceAddr,
	)
	if err != nil {
		log.Fatalf("Failed to connect to auth service: %v", err)
	}
	defer func() { _ = authConn.Close() }()
	log.Printf("✅ Created auth service client for %s (connection will be established on first RPC call)", cfg.AuthServiceAddr)

	// Create connections to other services (with fallback if not configured)
	var supportConn *grpc.ClientConn

	if cfg.SupportServiceAddr != "" {
		supportConn, err = grpcutil.NewClient(
			cfg.SupportServiceAddr,
		)
		if err != nil {
			log.Printf("⚠️  Failed to connect to support service: %v", err)
		} else {
			defer func() { _ = supportConn.Close() }()
			log.Printf("✅ Connected to support service at %s", cfg.SupportServiceAddr)
		}
	}

	// Create auth client for middleware (non-auth routes still validate tokens via auth-service gRPC).
	authClient := pb.NewAuthServiceClient(authConn)

	// Create authentication middleware
	authMiddleware := middleware.AuthMiddleware(authClient)

	var supportHandler *handler.SupportHandler
	if supportConn != nil {
		supportHandler = handler.NewSupportHandler(supportConn, authConn, cfg.StorageServiceAddr, cfg.AppURL)
	}

	// Create storage handler (HTTP reverse proxy, no gRPC connection needed)
	var storageHandler *handler.StorageHandler
	if cfg.StorageServiceAddr != "" {
		storageHandler = handler.NewStorageHandler(cfg.StorageServiceAddr)
		log.Printf("✅ Created storage handler for %s", cfg.StorageServiceAddr)
	} else {
		log.Printf("⚠️  STORAGE_SERVICE_ADDR not set - upload routes will not be available")
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Auth-domain routes are served directly by auth-service HTTP (see services/auth-service).
	// Dynasty routes are served directly by dynasty-service HTTP (see services/dynasty-service).
	// Training routes are served directly by training-service HTTP (see services/training-service).

	// Support routes
	if supportHandler != nil {
		mux.Handle("/api/support/tickets", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				supportHandler.ListTickets(w, r)
			case http.MethodPost:
				supportHandler.CreateTicket(w, r)
			default:
				http.NotFound(w, r)
			}
		})))
		mux.Handle("/api/support/tickets/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if strings.Contains(path, "/response/") {
				supportHandler.AddTicketResponse(w, r)
			} else if strings.Contains(path, "/close/") {
				supportHandler.CloseTicket(w, r)
			} else if r.Method == http.MethodGet {
				supportHandler.GetTicket(w, r)
			} else if r.Method == http.MethodPut || r.Method == http.MethodPatch {
				supportHandler.UpdateTicket(w, r)
			} else {
				http.NotFound(w, r)
			}
		})))
		mux.Handle("/api/support/reports", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				supportHandler.ListReports(w, r)
			case http.MethodPost:
				supportHandler.CreateReport(w, r)
			default:
				http.NotFound(w, r)
			}
		})))
		mux.Handle("/api/support/reports/", authMiddleware(http.HandlerFunc(supportHandler.GetReport)))
		// Direct routes (without /support prefix) - for Kong compatibility
		mux.Handle("/api/tickets", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				supportHandler.ListTickets(w, r)
			case http.MethodPost:
				supportHandler.CreateTicket(w, r)
			default:
				http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			}
		})))
		mux.Handle("/api/tickets/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if strings.Contains(path, "/response/") {
				supportHandler.AddTicketResponse(w, r)
				return
			}
			if strings.Contains(path, "/close/") {
				supportHandler.CloseTicket(w, r)
				return
			}
			if r.Method == http.MethodPut || r.Method == http.MethodPatch {
				supportHandler.UpdateTicket(w, r)
				return
			}
			supportHandler.GetTicket(w, r)
		})))
		mux.Handle("/api/reports", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				supportHandler.ListReports(w, r)
			case http.MethodPost:
				supportHandler.CreateReport(w, r)
			default:
				http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			}
		})))
		mux.Handle("/api/reports/", authMiddleware(http.HandlerFunc(supportHandler.GetReport)))
		mux.Handle("/api/notes", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				supportHandler.ListNotes(w, r)
			case http.MethodPost:
				supportHandler.CreateNote(w, r)
			default:
				http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			}
		})))
		mux.Handle("/api/notes/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method := handler.EffectiveHTTPMethod(r)
			switch method {
			case http.MethodDelete:
				supportHandler.DeleteNote(w, r)
			case http.MethodPut, http.MethodPatch:
				supportHandler.UpdateNote(w, r)
			case http.MethodGet:
				supportHandler.GetNote(w, r)
			default:
				http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			}
		})))
	}

	// Storage routes (public endpoint, no authentication required)
	if storageHandler != nil {
		mux.HandleFunc("/api/upload", storageHandler.HandleUpload)
		log.Printf("✅ Registered storage upload route: /api/upload")
	}

	// Note: We don't register a catch-all "/" handler because it would interfere with route matching
	// Instead, unmatched routes will naturally return 404 from ServeMux

	// CORS is handled exclusively by Kong (credentials + allowlist). Do not set
	// Access-Control-Allow-Origin here — ACAO:* conflicts with Kong credentials:true.
	handler := sentry.HTTPMiddleware(middleware.LoggingMiddleware(mux))

	// Start HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: handler,
	}

	// Graceful shutdown
	go func() {
		log.Printf("🚀 gRPC Gateway starting on port %s", cfg.HTTPPort)
		log.Printf("🏥 Health check: http://localhost:%s/health", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// registerExactAndTrailingSlash registers a handler for each path and its trailing-slash variant.
// Go 1.22+ ServeMux matches exact paths only unless the pattern ends with "/".
func registerExactAndTrailingSlash(mux *http.ServeMux, handler http.Handler, paths ...string) {
	for _, path := range paths {
		mux.Handle(path, handler)
		if !strings.HasSuffix(path, "/") {
			mux.Handle(path+"/", handler)
		}
	}
}
