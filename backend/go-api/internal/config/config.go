// Package config handles the loading and parsing of application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"egobackend/internal/models"
)

// AppConfig holds all configuration settings for the application.
type AppConfig struct {
	// --- Core Settings ---
	DBPath           string // Database connection string (e.g., PostgreSQL DSN).
	ServerAddr       string // Address for the HTTP server to listen on (e.g., ":8080").
	APIEncryptionKey string // 32-byte key for encrypting sensitive data like API keys.

	// --- Authentication ---
	JWTSecret      string // Secret key for signing JWT tokens.
	GoogleClientID string // Client ID for Google OAuth. Optional.

	// --- External Services ---
	PythonBackendURL string          // URL of the Python backend service.
	S3               models.S3Config // Configuration for S3-compatible storage. Optional.

	// --- Application Logic ---
	MaxThinkingIterations int    // Max iterations for the agent's thinking cycle.
	MigrationsPath        string // Path to the database migration files.
	CORSAllowedOrigins    string // Comma-separated list of allowed CORS origins.

	// --- Timeouts and Intervals ---
	HTTPClientTimeout           time.Duration // Timeout for the general HTTP client.
	ShutdownTimeout             time.Duration // Graceful shutdown timeout.
	ShutdownFinalSleep          time.Duration // Final sleep duration before exit.
	FileCleanupInterval         time.Duration // How often to run the old file cleanup job.
	FileCleanupAge              time.Duration // Age threshold for deleting old files.
	OrphanCleanupInterval       time.Duration // How often to run the orphaned file cleanup job.
	OrphanCleanupAge            time.Duration // Age threshold for deleting orphaned files.
	KeepAliveBootstrapInterval  time.Duration // Initial, frequent keep-alive interval for Python backend.
	KeepAliveRegularInterval    time.Duration // Regular keep-alive interval for Python backend.
	KeepAliveHealthcheckTimeout time.Duration // Timeout for the keep-alive health check request.
	KeepAliveWarmupAttempts     int           // Max attempts to warm up the Python backend.
	CORSMaxAge                  int           // Max age for CORS preflight requests in seconds.
}

// Load reads environment variables and populates the AppConfig struct.
// It sets sensible defaults for non-critical values.
func Load() (*AppConfig, error) {
	// Normalize S3 endpoint: ensure it has a scheme for AWS SDK endpoint resolver
	normalizeEndpoint := func(raw string) string {
		if raw == "" {
			return raw
		}
		if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
			return raw
		}
		return "https://" + raw
	}

	// Read S3 credentials with backward-compatible aliases
	s3KeyID := getEnv("S3_ACCESS_KEY", "")
	if s3KeyID == "" {
		s3KeyID = getEnv("S3_ACCESS_KEY_ID", "")
	}
	s3Secret := getEnv("S3_SECRET_KEY", "")
	if s3Secret == "" {
		s3Secret = getEnv("S3_SECRET_ACCESS_KEY", "")
	}

	cfg := &AppConfig{
		// --- Core Settings ---
		DBPath:           getEnv("DB_PATH", ""),
		ServerAddr:       getEnv("SERVER_ADDR", ":8080"),
		APIEncryptionKey: getEnv("API_ENCRYPTION_KEY", ""),

		// --- Authentication ---
		JWTSecret:      getEnv("JWT_SECRET", ""),
		GoogleClientID: getEnv("GOOGLE_CLIENT_ID", ""),

		// --- External Services ---
		PythonBackendURL: getEnv("PYTHON_BACKEND_URL", ""),
		S3: models.S3Config{
			Endpoint: normalizeEndpoint(getEnv("S3_ENDPOINT", "")),
			Region:   getEnv("S3_REGION", ""),
			KeyID:    s3KeyID,
			AppKey:   s3Secret,
			Bucket:   getEnv("S3_BUCKET_NAME", ""),
		},

		// --- Application Logic ---
		MaxThinkingIterations: getEnvAsInt("MAX_THINKING_ITERATIONS", 10),
		MigrationsPath:        getEnv("MIGRATIONS_PATH", "migrations"),
		// Include common dev origins and the production Vercel domain by default.
		// Can be overridden via CORS_ALLOWED_ORIGINS env.
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:4173,https://ego-on.vercel.app"),

		// --- Timeouts and Intervals ---
		HTTPClientTimeout:           getEnvAsDuration("HTTP_CLIENT_TIMEOUT", 15*time.Minute),
		ShutdownTimeout:             getEnvAsDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		ShutdownFinalSleep:          getEnvAsDuration("SHUTDOWN_FINAL_SLEEP", 5*time.Second),
		FileCleanupInterval:         getEnvAsDuration("FILE_CLEANUP_INTERVAL", 24*time.Hour),
		FileCleanupAge:              getEnvAsDuration("FILE_CLEANUP_AGE", 24*time.Hour),
		OrphanCleanupInterval:       getEnvAsDuration("ORPHAN_CLEANUP_INTERVAL", 6*time.Hour),
		OrphanCleanupAge:            getEnvAsDuration("ORPHAN_CLEANUP_AGE", 1*time.Hour),
		KeepAliveBootstrapInterval:  getEnvAsDuration("KEEPALIVE_BOOTSTRAP_INTERVAL", 30*time.Second),
		KeepAliveRegularInterval:    getEnvAsDuration("KEEPALIVE_REGULAR_INTERVAL", 4*time.Minute),
		KeepAliveHealthcheckTimeout: getEnvAsDuration("KEEPALIVE_HEALTHCHECK_TIMEOUT", 30*time.Second),
		KeepAliveWarmupAttempts:     getEnvAsInt("KEEPALIVE_WARMUP_ATTEMPTS", 10),
		CORSMaxAge:                  getEnvAsInt("CORS_MAX_AGE", 300),
	}

	// Validate critical environment variables.
	if err := validateCriticalConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validateCriticalConfig checks that essential configuration values are set.
func validateCriticalConfig(cfg *AppConfig) error {
	criticalVars := map[string]string{
		"DB_PATH":            cfg.DBPath,
		"JWT_SECRET":         cfg.JWTSecret,
		"PYTHON_BACKEND_URL": cfg.PythonBackendURL,
		"API_ENCRYPTION_KEY": cfg.APIEncryptionKey,
	}
	var missing []string
	for name, value := range criticalVars {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing critical environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

// --- Helper Functions for robust environment variable loading ---

// getEnv retrieves a string environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves an integer environment variable or returns a default value.
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getEnvAsDuration retrieves a time.Duration environment variable or returns a default value.
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if duration, err := time.ParseDuration(valueStr); err == nil {
		return duration
	}
	return defaultValue
}
