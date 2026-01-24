// Package middleware provides HTTP middleware handlers.
package middleware

import (
	"log"
	"net/http"
	"strings"

	"egobackend/internal/database"
	"egobackend/internal/handlers"
)

// MaintenanceMiddleware checks if the application is in maintenance mode.
// If it is, it blocks most requests, allowing only specific status endpoints
// and requests with a valid bypass token.
func MaintenanceMiddleware(db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Always allow CORS preflight requests.
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			status, err := db.GetMaintenanceMode()
			if err != nil {
				log.Printf("MaintenanceMiddleware: Error checking maintenance status: %v. Allowing request to proceed.", err)
				next.ServeHTTP(w, r) // Fail open for safety if DB is down.
				return
			}

			if !status.IsEnabled || status.IsChatOnly {
				next.ServeHTTP(w, r)
				return
			}

			// Allow access to maintenance status, token validation, and static assets.
			path := r.URL.Path
			if strings.HasPrefix(path, "/api/maintenance/") ||
				strings.HasPrefix(path, "/maintenance/") ||
				strings.HasPrefix(path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}

			// Check for a bypass token.
			token := r.Header.Get("X-Bypass-Token")
			if token == "" {
				token = r.URL.Query().Get("bypass_token")
			}
			if token == "" {
				if cookie, err := r.Cookie("maintenance_bypass_token"); err == nil {
					token = cookie.Value
				}
			}

			if token != "" {
				isValid, err := db.ValidateBypassToken(token)
				if err == nil && isValid {
					next.ServeHTTP(w, r) // Token is valid, allow access.
					return
				}
			}

			// If we reach here, the request is blocked.
			message := ""
			if status.Message != nil {
				message = *status.Message
			}

			// For all requests (API and browsers hitting API routes), return a JSON error.
			// This allows the Svelte frontend to handle the maintenance UI.
			handlers.RespondWithError(w, http.StatusServiceUnavailable, message)
		})
	}
}
