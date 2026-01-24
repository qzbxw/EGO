// Package main is the entry point for the EGO backend API server.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"egobackend/internal/auth"
	"egobackend/internal/config"
	"egobackend/internal/database"
	"egobackend/internal/engine"
	"egobackend/internal/handlers"
	"egobackend/internal/middleware"
	"egobackend/internal/storage"
	"egobackend/internal/telemetry"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

// main initializes the application, sets up dependencies, defines routes,
// and starts the HTTP server with graceful shutdown.
func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Critical error loading configuration: %v", err)
	}

	// --- Dependency Injection ---
	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Critical error! Failed to connect to the database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(cfg.DBPath, cfg.MigrationsPath); err != nil {
		log.Fatalf("Critical error during database migration: %v", err)
	}

	s3Service, err := storage.NewS3Service(cfg.S3)
	if err != nil {
		log.Fatalf("Critical error! Failed to create S3 service: %v", err)
	}

	authSvc, err := auth.NewAuthService(cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Critical error: failed to create authentication service: %v", err)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{DisableCompression: true},
		Timeout:   cfg.HTTPClientTimeout,
	}
	validate := validator.New()
	processor := engine.NewProcessor(db, cfg, s3Service, httpClient)

	// --- Background Goroutines ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go startFileCleanupRoutine(ctx, db, s3Service, cfg)
	go startOrphanedFileCleanupRoutine(ctx, db, s3Service, cfg)
	go startKeepAlive(ctx, cfg)
	go telemetry.InitializeBot(db)
	
	// Start Profile Summarizer
	profileSummarizer := engine.NewProfileSummarizer(db, processor.GetLLMClient())
	go profileSummarizer.Start(ctx)

	// --- Router and Server Setup ---
	router := setupRouter(db, cfg, authSvc, s3Service, processor, validate)
	srv := &http.Server{Addr: cfg.ServerAddr, Handler: router}

	go func() {
		log.Printf("Server is ready for connections and listening on %s", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed with error: %v", err)
		}
	}()

	<-ctx.Done()

	log.Println("Shutdown signal received. Starting graceful shutdown...")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Error during graceful server shutdown: %v", err)
	}

	log.Printf("Server stopped successfully. Background tasks can continue for up to %v.", cfg.ShutdownFinalSleep)
	time.Sleep(cfg.ShutdownFinalSleep)
	log.Println("Exiting.")
}

// setupRouter initializes all handlers and registers all API routes.
func setupRouter(db *database.DB, cfg *config.AppConfig, authSvc *auth.AuthService, s3Service *storage.S3Service, processor *engine.Processor, validate *validator.Validate) *chi.Mux {
	// Initialize handlers using their constructors
	authHandler := &handlers.AuthHandler{DB: db, AuthService: authSvc, GoogleClientID: cfg.GoogleClientID}
	sessionHandler := &handlers.SessionHandler{DB: db, S3Service: s3Service, LLMClient: processor}
	userHandler := handlers.NewUserHandler(db, cfg.APIEncryptionKey)
	chatHandler := handlers.NewChatHandler(db, processor, validate, cfg)
	uploadHandler := handlers.NewUploadHandler(chatHandler)
	maintenanceHandler := handlers.NewMaintenanceHandler(db)
	statusHandler := handlers.NewStatusHandler(db)
	staticHandler := handlers.NewStaticHandler("static") // Assuming 'static' is the directory name
	rootHandler := handlers.NewRootHandler(db)
	statisticsHandler := handlers.NewStatisticsHandler(db)
	titleProxyHandler, err := handlers.NewTitleProxyHandler(cfg.PythonBackendURL)
	if err != nil {
		log.Fatalf("Failed to create title proxy handler: %v", err)
	}
	pythonProxyHandler, err := handlers.NewPythonProxyHandler(cfg.PythonBackendURL)
	if err != nil {
		log.Fatalf("Failed to create python proxy handler: %v", err)
	}

	r := chi.NewRouter()

	// --- Middleware Stack ---
	setupCORS(r, cfg)
	r.Use(chimiddleware.Logger, chimiddleware.Recoverer, CoopMiddleware)
	r.Use(middleware.MaintenanceMiddleware(db))

	// --- Route Registration ---

	// Root and error handlers
	r.Get("/", rootHandler.HandleRoot) // Handles all methods via RegisterRoutes in refactor
	r.NotFound(func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})

	// Public unauthenticated routes
	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/google", authHandler.GoogleLogin)
	r.Post("/auth/refresh", authHandler.Refresh)
	r.Get("/api/public-stats", statisticsHandler.GetPublicStats)

	// Maintenance, status, and static file routes
	maintenanceHandler.RegisterRoutes(r)
	statusHandler.RegisterRoutes(r)
	staticHandler.RegisterRoutes(r)

	// Proxy all /py requests to Python backend (protected by MaintenanceMiddleware)
	r.Mount("/py", http.StripPrefix("/py", http.HandlerFunc(pythonProxyHandler.HandleProxy)))

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Public API aliases
		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/google", authHandler.GoogleLogin)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Post("/generate_title", titleProxyHandler.HandleProxy)

		// Authenticated API routes
		r.Group(func(r chi.Router) {
			r.Use(authHandler.AuthMiddleware)

			r.Get("/me", authHandler.Me)
			r.Post("/chat/stream", chatHandler.HandleChatStream)

			// Sessions
			r.Get("/sessions", sessionHandler.GetSessions)
			r.Post("/sessions", sessionHandler.CreateSession)
			r.Get("/sessions/{sessionID}", sessionHandler.GetSession)
			r.Patch("/sessions/{sessionID}", sessionHandler.UpdateSession)
			r.Delete("/sessions/{sessionID}", sessionHandler.DeleteSession)
			r.Delete("/sessions", sessionHandler.ClearAllSessions)
			r.Get("/sessions/{sessionID}/history", sessionHandler.GetHistory)

			// Logs
			r.Get("/logs/{logID}", sessionHandler.GetLog)
			r.Patch("/logs/{logID}", sessionHandler.EditLog)

			// User
			r.Get("/user/settings", userHandler.GetSettings)
			r.Put("/user/settings", userHandler.UpdateSettings)
			r.Delete("/user/settings", userHandler.DeleteSettings)
			r.Delete("/user/account", userHandler.DeleteAccount)

			// Attachments
			r.Get("/attachments/{logID}/{fileName}", sessionHandler.PreviewAttachment)
			r.Post("/chat-multipart", sessionHandler.ChatMultipartUpload)

			// Statistics (User)
			r.Get("/statistics", statisticsHandler.GetUserStatistics)
			r.Get("/statistics/provider_tokens", statisticsHandler.GetUserProviderTokens)

			// Uploads
			r.Post("/upload", uploadHandler.HandleUpload)

			// Statistics (Admin)
			r.Group(func(r chi.Router) {
				// r.Use(middleware.AdminOnlyMiddleware) // To be added
				r.Get("/statistics/user/{userID}", statisticsHandler.GetUserStatisticsByID)
				r.Get("/statistics/global", statisticsHandler.GetGlobalStatistics)
				r.Get("/statistics/users", statisticsHandler.GetAllUsersStatistics)
			})
		})
	})

	return r
}

// --- Background Routines ---

func startFileCleanupRoutine(ctx context.Context, db *database.DB, s3Service *storage.S3Service, cfg *config.AppConfig) {
	log.Println("[CLEANUP] Starting background routine for old file cleanup from S3...")
	ticker := time.NewTicker(cfg.FileCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("[CLEANUP] Performing scheduled cleanup of old files (older than %v) from S3...", cfg.FileCleanupAge)
			deletedURIs, err := db.DeleteOldFileAttachments(cfg.FileCleanupAge)
			if err != nil {
				log.Printf("!!! [CLEANUP] ERROR while deleting records from DB: %v", err)
				continue
			}
			if len(deletedURIs) > 0 {
				err = s3Service.DeleteFiles(context.Background(), deletedURIs)
				if err != nil {
					log.Printf("!!! [CLEANUP] ERROR deleting files from S3: %v", err)
				} else {
					log.Printf("[CLEANUP] S3 cleanup finished. Successfully deleted %d objects.", len(deletedURIs))
				}
			} else {
				log.Println("[CLEANUP] No old files found to delete.")
			}
		case <-ctx.Done():
			log.Println("[CLEANUP] Cleanup routine stopped due to server shutdown.")
			return
		}
	}
}

func startOrphanedFileCleanupRoutine(ctx context.Context, db *database.DB, s3Service *storage.S3Service, cfg *config.AppConfig) {
	log.Println("[CLEANUP-ORPHANED] Starting routine for orphaned file cleanup...")
	ticker := time.NewTicker(cfg.OrphanCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("[CLEANUP-ORPHANED] Cleaning up files not linked to any logs and older than %v...", cfg.OrphanCleanupAge)
			deletedURIs, err := db.DeleteOrphanedFileAttachments(cfg.OrphanCleanupAge)
			if err != nil {
				log.Printf("!!! [CLEANUP-ORPHANED] ERROR deleting records from DB: %v", err)
				continue
			}
			if len(deletedURIs) > 0 {
				err = s3Service.DeleteFiles(context.Background(), deletedURIs)
				if err != nil {
					log.Printf("!!! [CLEANUP-ORPHANED] ERROR deleting files from S3: %v", err)
				} else {
					log.Printf("[CLEANUP-ORPHANED] S3 cleanup finished. Deleted %d orphaned objects.", len(deletedURIs))
				}
			} else {
				log.Println("[CLEANUP-ORPHANED] No orphaned files found.")
			}
		case <-ctx.Done():
			log.Println("[CLEANUP-ORPHANED] Cleanup routine stopped.")
			return
		}
	}
}

func startKeepAlive(ctx context.Context, cfg *config.AppConfig) {
	log.Println("[KEEP-ALIVE] Starting keep-alive process...")
	bootstrapTicker := time.NewTicker(cfg.KeepAliveBootstrapInterval)
	defer bootstrapTicker.Stop()
	regularTicker := time.NewTicker(cfg.KeepAliveRegularInterval)
	defer regularTicker.Stop()

	healthCheckURL := cfg.PythonBackendURL + "/healthcheck"
	client := &http.Client{Timeout: cfg.KeepAliveHealthcheckTimeout}

	warmed := false
	warmAttempts := 0
	maxWarmAttempts := cfg.KeepAliveWarmupAttempts

	for {
		select {
		case <-bootstrapTicker.C:
			if !warmed && warmAttempts < maxWarmAttempts {
				warmAttempts++
				log.Printf("[KEEP-ALIVE] Sending bootstrap warm-up request to Python (%d/%d)...", warmAttempts, maxWarmAttempts)
				resp, err := client.Get(healthCheckURL)
				if err == nil && resp.StatusCode == http.StatusOK {
					log.Println("[KEEP-ALIVE] Python backend is warmed up.")
					warmed = true
					bootstrapTicker.Stop()
				}
				if resp != nil {
					resp.Body.Close()
				}
			} else if !warmed {
				log.Println("[KEEP-ALIVE] Max warm-up attempts reached. Switching to regular keep-alive.")
				warmed = true
				bootstrapTicker.Stop()
			}
		case <-regularTicker.C:
			log.Println("[KEEP-ALIVE] Sending regular health-check request to Python service...")
			resp, err := client.Get(healthCheckURL)
			if err != nil {
				log.Printf("!!! [KEEP-ALIVE] ERROR during health-check request: %v", err)
				continue
			}
			if resp.StatusCode == http.StatusOK {
				log.Println("[KEEP-ALIVE] Python service is healthy.")
			} else {
				log.Printf("!!! [KEEP-ALIVE] Python service returned non-OK status: %d", resp.StatusCode)
			}
			if resp != nil {
				resp.Body.Close()
			}
		case <-ctx.Done():
			log.Println("[KEEP-ALIVE] Keep-alive process stopped.")
			return
		}
	}
}

// --- Middleware Configuration ---

func setupCORS(r *chi.Mux, cfg *config.AppConfig) {
	allowedOrigins := strings.Split(cfg.CORSAllowedOrigins, ",")
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Origin", "X-Requested-With", "X-User-ID", "X-Memory-Enabled", "X-Bypass-Token"},
		ExposedHeaders:   []string{"Content-Length", "Content-Type"},
		MaxAge:           cfg.CORSMaxAge,
	}).Handler)
}

func CoopMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
		w.Header().Set("Cross-Origin-Embedder-Policy", "unsafe-none")
		next.ServeHTTP(w, r)
	})
}
