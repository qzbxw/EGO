// Package middleware provides HTTP middleware handlers.
package middleware

import (
	"log"
	"net/http"
	"strings"

	"egobackend/internal/database"
	"egobackend/internal/handlers"
	"egobackend/internal/templates"
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

			if !status.IsEnabled {
				next.ServeHTTP(w, r)
				return
			}
			
			// Allow access to maintenance status and token validation endpoints.
			path := r.URL.Path
			if strings.HasPrefix(path, "/api/maintenance/") || strings.HasPrefix(path, "/maintenance/") {
				next.ServeHTTP(w, r)
				return
			}

			// Check for a bypass token.
			token := r.Header.Get("X-Bypass-Token")
			if token == "" {
				// Support both bypass_token and token aliases
				q := r.URL.Query()
				token = q.Get("bypass_token")
				if token == "" {
					token = q.Get("token")
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

			// For browsers, render the beautiful HTML page.
			if handlers.IsBrowserRequest(r) { // Using the function from handlers.utils
				if err := templates.RenderStatusPage(w, true, message); err != nil {
					log.Printf("MaintenanceMiddleware: Error rendering HTML page: %v", err)
					// Fallback to JSON if HTML rendering fails.
					handlers.RespondWithError(w, http.StatusServiceUnavailable, message)
				}
				return
			}

			// For API clients, return a JSON error.
			handlers.RespondWithError(w, http.StatusServiceUnavailable, message)
		})
	}
}