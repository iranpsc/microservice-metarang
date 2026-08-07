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
	var dynastyConn, trainingConn, supportConn *grpc.ClientConn

	if cfg.DynastyServiceAddr != "" {
		dynastyConn, err = grpcutil.NewClient(
			cfg.DynastyServiceAddr,
		)
		if err != nil {
			log.Printf("⚠️  Failed to connect to dynasty service: %v", err)
		} else {
			defer func() { _ = dynastyConn.Close() }()
			log.Printf("✅ Connected to dynasty service at %s", cfg.DynastyServiceAddr)
		}
	}

	if cfg.TrainingServiceAddr != "" {
		trainingConn, err = grpcutil.NewClient(
			cfg.TrainingServiceAddr,
		)
		if err != nil {
			log.Printf("⚠️  Failed to connect to training service: %v", err)
			log.Printf("⚠️  Training routes will not be available until service is running")
			trainingConn = nil
		} else {
			defer func() { _ = trainingConn.Close() }()
			log.Printf("✅ Connected to training service at %s", cfg.TrainingServiceAddr)
		}
	} else {
		log.Printf("⚠️  TRAINING_SERVICE_ADDR not set - training routes will not be available")
	}

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
	optionalAuthMiddleware := middleware.OptionalAuthMiddleware(authClient)

	var dynastyHandler *handler.DynastyHandler
	if dynastyConn != nil {
		dynastyHandler = handler.NewDynastyHandler(dynastyConn, authConn)
	}

	var trainingHandler *handler.TrainingHandler
	if trainingConn != nil {
		trainingHandler = handler.NewTrainingHandler(trainingConn, authConn)
		log.Printf("✅ Training handler created")
	} else {
		log.Printf("⚠️  Training handler NOT created - trainingConn is nil (check TRAINING_SERVICE_ADDR config)")
	}

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

	// Dynasty routes
	if dynastyHandler != nil {
		// GET /api/dynasty - Get user's dynasty or available features/intro prizes
		mux.Handle("/api/dynasty", authMiddleware(http.HandlerFunc(dynastyHandler.GetDynasty)))

		// POST /api/dynasty/create/{feature} - Create dynasty with feature
		mux.Handle("/api/dynasty/create/", authMiddleware(http.HandlerFunc(dynastyHandler.CreateDynasty)))

		// POST /api/dynasty/{dynasty}/update/{feature} - Update dynasty feature
		mux.Handle("/api/dynasty/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			// Handle /api/dynasty/{dynasty}/update/{feature}
			if strings.Contains(path, "/update/") {
				if r.Method == http.MethodPost {
					dynastyHandler.UpdateDynastyFeature(w, r)
				} else {
					http.NotFound(w, r)
				}
				return
			}
			// Handle /api/dynasty/{dynasty}/family/{family}
			if strings.Contains(path, "/family/") {
				if r.Method == http.MethodGet {
					dynastyHandler.GetFamily(w, r)
				} else {
					http.NotFound(w, r)
				}
				return
			}
			http.NotFound(w, r)
		})))

		// GET /api/dynasty/requests/sent - List sent join requests
		mux.Handle("/api/dynasty/requests/sent", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				dynastyHandler.GetSentRequests(w, r)
			} else {
				http.NotFound(w, r)
			}
		})))
		// GET /api/dynasty/requests/sent/{joinRequest} - View sent request
		// DELETE /api/dynasty/requests/sent/{joinRequest} - Delete sent request
		mux.Handle("/api/dynasty/requests/sent/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				dynastyHandler.GetSentRequest(w, r)
			case http.MethodDelete:
				dynastyHandler.DeleteJoinRequest(w, r)
			default:
				http.NotFound(w, r)
			}
		})))

		// GET /api/dynasty/requests/recieved - List received join requests
		mux.Handle("/api/dynasty/requests/recieved", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				dynastyHandler.GetReceivedRequests(w, r)
			} else {
				http.NotFound(w, r)
			}
		})))
		// GET /api/dynasty/requests/recieved/{joinRequest} - View received request
		// POST /api/dynasty/requests/recieved/{joinRequest} - Accept request
		// DELETE /api/dynasty/requests/recieved/{joinRequest} - Reject request
		mux.Handle("/api/dynasty/requests/recieved/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				dynastyHandler.GetReceivedRequest(w, r)
			case http.MethodPost:
				dynastyHandler.AcceptJoinRequest(w, r)
			case http.MethodDelete:
				dynastyHandler.RejectJoinRequest(w, r)
			default:
				http.NotFound(w, r)
			}
		})))

		// POST /api/dynasty/add/member/get/permissions - Get default permissions
		mux.Handle("/api/dynasty/add/member/get/permissions", authMiddleware(http.HandlerFunc(dynastyHandler.GetDefaultPermissions)))

		// POST /api/dynasty/add/member - Send join request
		mux.Handle("/api/dynasty/add/member", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				dynastyHandler.SendJoinRequest(w, r)
			} else {
				http.NotFound(w, r)
			}
		})))

		// POST /api/dynasty/search - Search users
		mux.Handle("/api/dynasty/search", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				dynastyHandler.SearchUsers(w, r)
			} else {
				http.NotFound(w, r)
			}
		})))

		// GET /api/dynasty/prizes - List prizes
		// GET /api/dynasty/prizes/{recievedPrize} - View prize
		// POST /api/dynasty/prizes/{recievedPrize} - Claim prize
		mux.Handle("/api/dynasty/prizes", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				dynastyHandler.GetPrizes(w, r)
			} else {
				http.NotFound(w, r)
			}
		})))
		mux.Handle("/api/dynasty/prizes/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				// View prize handler would go here
				http.NotFound(w, r)
			case http.MethodPost:
				dynastyHandler.ClaimPrize(w, r)
			default:
				http.NotFound(w, r)
			}
		})))

		// POST /api/dynasty/children/{user} - Update child permissions
		mux.Handle("/api/dynasty/children/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				dynastyHandler.UpdateChildPermissions(w, r)
			} else {
				http.NotFound(w, r)
			}
		})))
	}

	// Training routes
	if trainingHandler != nil {
		log.Printf("✅ Registering training service routes...")

		// Register more specific routes FIRST (before catch-all routes)

		// V1 modal lookup route (completely separate path) - public
		mux.Handle("/api/video-tutorials", optionalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				trainingHandler.GetVideoByFileName(w, r)
			} else {
				http.NotFound(w, r)
			}
		})))
		log.Printf("  ✅ Registered: POST /api/video-tutorials")

		// Category routes (must be before /api/tutorials/ catch-all) - public viewing
		mux.Handle("/api/tutorials/categories", optionalAuthMiddleware(http.HandlerFunc(trainingHandler.GetCategories)))
		mux.Handle("/api/tutorials/categories/", optionalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			// Remove prefix to get the category path
			categoryPath := strings.TrimPrefix(path, "/api/tutorials/categories/")
			parts := strings.Split(categoryPath, "/")

			if len(parts) == 1 && parts[0] != "" {
				// /api/tutorials/categories/{category:slug}
				if r.Method == http.MethodGet {
					trainingHandler.GetCategory(w, r)
				} else {
					http.NotFound(w, r)
				}
			} else if len(parts) == 2 && parts[1] == "videos" {
				// /api/tutorials/categories/{category:slug}/videos
				if r.Method == http.MethodGet {
					trainingHandler.GetCategoryVideos(w, r)
				} else {
					http.NotFound(w, r)
				}
			} else if len(parts) == 2 && parts[1] != "" {
				// /api/tutorials/categories/{category:slug}/{subCategory:slug}
				if r.Method == http.MethodGet {
					trainingHandler.GetSubCategory(w, r)
				} else {
					http.NotFound(w, r)
				}
			} else {
				http.NotFound(w, r)
			}
		})))

		// Search route (must be before /api/tutorials/ catch-all) - public
		mux.Handle("/api/tutorials/search", optionalAuthMiddleware(http.HandlerFunc(trainingHandler.SearchVideos)))

		// Dynamic video routes - catch-all for /api/tutorials/{...}
		// This must be registered AFTER more specific routes like /api/tutorials/categories
		// But BEFORE /api/tutorials to handle /api/tutorials/ properly
		// Uses conditional middleware: authMiddleware for authenticated routes, optionalAuthMiddleware for others
		mux.Handle("/api/tutorials/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			videoPath := strings.TrimPrefix(path, "/api/tutorials/")
			videoPath = strings.Trim(videoPath, "/")
			parts := strings.Split(videoPath, "/")

			// Video like/dislike requires auth
			if len(parts) == 2 && parts[1] == "interactions" && r.Method == http.MethodPost {
				authMiddleware(http.HandlerFunc(trainingHandler.AddInteraction)).ServeHTTP(w, r)
				return
			}

			// For all other routes, use optionalAuthMiddleware
			optionalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// If empty, this is exactly /api/tutorials/ - handle as list
				if videoPath == "" {
					if r.Method == http.MethodGet {
						trainingHandler.GetVideos(w, r)
						return
					} else {
						http.NotFound(w, r)
						return
					}
				}

				// Check for comment routes: /api/tutorials/{video}/comments/...
				if len(parts) >= 2 && parts[1] == "comments" {
					if len(parts) >= 4 {
						// /api/tutorials/{video}/comments/{comment}/{action}
						action := parts[3]
						switch action {
						case "interactions":
							if r.Method == http.MethodPost {
								authMiddleware(http.HandlerFunc(trainingHandler.AddCommentInteraction)).ServeHTTP(w, r)
							} else {
								http.NotFound(w, r)
							}
						case "like":
							if r.Method == http.MethodPost {
								authMiddleware(http.HandlerFunc(trainingHandler.AddCommentLike)).ServeHTTP(w, r)
							} else {
								http.NotFound(w, r)
							}
						case "dislike":
							if r.Method == http.MethodPost {
								authMiddleware(http.HandlerFunc(trainingHandler.AddCommentDislike)).ServeHTTP(w, r)
							} else {
								http.NotFound(w, r)
							}
						case "report":
							if r.Method == http.MethodPost {
								authMiddleware(http.HandlerFunc(trainingHandler.ReportComment)).ServeHTTP(w, r)
							} else {
								http.NotFound(w, r)
							}
						default:
							http.NotFound(w, r)
						}
					} else if len(parts) == 3 && parts[1] == "comments" {
						// /api/tutorials/{video}/comments/{comment}
						switch r.Method {
						case http.MethodPut, http.MethodPost:
							authMiddleware(http.HandlerFunc(trainingHandler.UpdateComment)).ServeHTTP(w, r)
						case http.MethodDelete:
							authMiddleware(http.HandlerFunc(trainingHandler.DeleteComment)).ServeHTTP(w, r)
						default:
							http.NotFound(w, r)
						}
					} else if len(parts) == 2 {
						// /api/tutorials/{video}/comments
						switch r.Method {
						case http.MethodGet:
							trainingHandler.GetComments(w, r)
						case http.MethodPost:
							authMiddleware(http.HandlerFunc(trainingHandler.AddComment)).ServeHTTP(w, r)
						default:
							http.NotFound(w, r)
						}
					} else {
						http.NotFound(w, r)
					}
				} else if len(parts) == 1 {
					// /api/tutorials/{slug} - Get video by slug
					if r.Method == http.MethodGet {
						trainingHandler.GetVideo(w, r)
					} else {
						http.NotFound(w, r)
					}
				} else {
					// Unmatched path
					http.NotFound(w, r)
				}
			})).ServeHTTP(w, r)
		}))

		// Video tutorials list route - exact match (no trailing slash)
		// This is registered AFTER /api/tutorials/ to handle exact /api/tutorials
		// Public viewing route
		mux.Handle("/api/tutorials", optionalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				trainingHandler.GetVideos(w, r)
			} else {
				http.NotFound(w, r)
			}
		})))

		// Comment replies routes: /api/comments/{comment}/...
		// Uses optionalAuthMiddleware - handlers will enforce auth for actions
		mux.Handle("/api/comments/", optionalAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			// Remove prefix to get the comment path
			commentPath := strings.TrimPrefix(path, "/api/comments/")
			parts := strings.Split(commentPath, "/")

			if len(parts) >= 2 && parts[1] == "replies" {
				if len(parts) == 2 {
					// /api/comments/{comment}/replies
					if r.Method == http.MethodGet {
						trainingHandler.GetReplies(w, r)
					} else {
						http.NotFound(w, r)
					}
				} else if len(parts) == 3 {
					// /api/comments/{comment}/replies/{reply}
					switch r.Method {
					case http.MethodPut:
						trainingHandler.UpdateReply(w, r)
					case http.MethodDelete:
						trainingHandler.DeleteReply(w, r)
					default:
						http.NotFound(w, r)
					}
				} else if len(parts) == 4 && parts[3] == "interactions" {
					// /api/comments/{comment}/replies/{reply}/interactions
					if r.Method == http.MethodPost {
						trainingHandler.AddReplyInteraction(w, r)
					} else {
						http.NotFound(w, r)
					}
				} else {
					http.NotFound(w, r)
				}
			} else if len(parts) == 2 && parts[1] == "reply" {
				// /api/comments/{comment}/reply
				if r.Method == http.MethodPost {
					authMiddleware(http.HandlerFunc(trainingHandler.AddReply)).ServeHTTP(w, r)
				} else {
					http.NotFound(w, r)
				}
			} else {
				http.NotFound(w, r)
			}
		})))
		log.Printf("✅ All training service routes registered successfully")
	} else {
		log.Printf("⚠️  Training routes NOT registered - trainingHandler is nil")
		log.Printf("   Check if TRAINING_SERVICE_ADDR is set and training service is running")
		log.Printf("   Current TRAINING_SERVICE_ADDR: %s", cfg.TrainingServiceAddr)
		log.Printf("   trainingConn value: %v", trainingConn)
	}

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
