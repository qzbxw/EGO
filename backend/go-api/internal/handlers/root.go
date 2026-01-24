package handlers

import (
	"log"
	"net/http"

	"egobackend/internal/database"
)

// RootHandler handles requests to the root path ('/').
// It checks for maintenance mode and returns a consistent JSON status.
type RootHandler struct {
	db *database.DB
}

// NewRootHandler creates a new RootHandler.
func NewRootHandler(db *database.DB) *RootHandler {
	return &RootHandler{db: db}
}

// HandleRoot serves the response for the root path.
// It checks for maintenance mode and returns a consistent JSON status.
func (h *RootHandler) HandleRoot(w http.ResponseWriter, r *http.Request) {
	// Prevent search engines from indexing this root/status page.
	w.Header().Set("X-Robots-Tag", "noindex")

	// Fetch the current maintenance status.
	status, err := h.db.GetMaintenanceMode()
	if err != nil {
		log.Printf("Error getting maintenance status for root handler: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to get maintenance status")
		return
	}

	if status.IsEnabled {
		RespondWithError(w, http.StatusServiceUnavailable, "Service temporarily unavailable due to maintenance.")
	} else {
		RespondWithJSON(w, http.StatusOK, map[string]string{"status": "Service is available"})
	}
}