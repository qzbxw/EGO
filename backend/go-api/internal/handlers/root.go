package handlers

import (
	"log"
	"net/http"

	"egobackend/internal/database"
	"egobackend/internal/templates"
)

// RootHandler handles requests to the root path ('/').
// Its primary purpose is to act as a status page, providing a human-readable
// HTML response to browsers and a simple text-based status for API clients.
type RootHandler struct {
	db *database.DB
}

// NewRootHandler creates a new RootHandler.
func NewRootHandler(db *database.DB) *RootHandler {
	return &RootHandler{db: db}
}

// HandleRoot serves the response for the root path.
// It checks for maintenance mode and serves different content based on
// whether the request is from a browser or an API client.
func (h *RootHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	// Prevent search engines from indexing this root/status page.
	w.Header().Set("X-Robots-Tag", "noindex")

	// Fetch the current maintenance status.
	status, err := h.db.GetMaintenanceMode()
	if err != nil {
		log.Printf("Error getting maintenance status for root handler: %v", err)
		// Provide a different error response for browsers vs. API clients.
		if IsBrowserRequest(r) {
			http.Error(w, "Error retrieving service status.", http.StatusInternalServerError)
		} else {
			RespondWithError(w, http.StatusInternalServerError, "Failed to get maintenance status")
		}
		return
	}

	// For browser requests, always render a user-friendly HTML status page.
	if IsBrowserRequest(r) {
		message := "" // The template has a default message.
		if status.IsEnabled && status.Message != nil {
			message = *status.Message
		}
		// RenderMaintenancePage is assumed to be in a shared utility package (e.g., utils.go)
		if err := templates.RenderStatusPage(w, status.IsEnabled, message); err != nil {
			log.Printf("Error rendering HTML status page: %v", err)
			// Fallback to a simple text error if the template fails to render.
			http.Error(w, "Service status page is currently unavailable.", http.StatusInternalServerError)
		}
		return
	}

	// For API clients (non-browser requests), return a simple text status.
	if status.IsEnabled {
		RespondWithError(w, http.StatusServiceUnavailable, "Service temporarily unavailable due to maintenance.")
	} else {
		// Using RespondWithJSON for consistency in API responses.
		RespondWithJSON(w, http.StatusOK, map[string]string{"status": "Service is available"})
	}
}