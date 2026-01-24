package handlers

import (
	"log"
	"net/http"

	"egobackend/internal/database"

	"github.com/go-chi/chi/v5"
)

// StatusHandler handles requests for the general system status.
// It is intended primarily for API clients and monitoring tools.
type StatusHandler struct {
	db *database.DB
}

// NewStatusHandler creates a new StatusHandler.
func NewStatusHandler(db *database.DB) *StatusHandler {
	return &StatusHandler{db: db}
}

// RegisterRoutes registers the system status endpoints with the Chi router.
// It registers routes under both `/status` and `/api/status` for compatibility.
func (h *StatusHandler) RegisterRoutes(r chi.Router) {
	r.Get("/status", h.GetSystemStatus)
	r.Get("/api/status", h.GetSystemStatus)
}

// GetSystemStatus returns the current operational status of the system.
// It checks for maintenance mode and provides a consistent JSON response.
func (h *StatusHandler) GetSystemStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.db.GetMaintenanceMode()
	if err != nil {
		log.Printf("Error getting maintenance status for status handler: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve system status")
		return
	}

	var statusText, messageText string

	if status.IsEnabled {
		statusText = "unavailable"
		// Use the custom message from the DB if available, otherwise use a default.
		if status.Message != nil && *status.Message != "" {
			messageText = *status.Message
		} else {
			messageText = "Service is temporarily unavailable due to maintenance."
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		statusText = "available"
		messageText = "System is operating normally."
		w.WriteHeader(http.StatusOK)
	}

	response := map[string]interface{}{
		"maintenance_enabled": status.IsEnabled,
		"status":              statusText,
		"message":             messageText,
	}

	// We call RespondWithJSON after setting the status code.
	// Note: The helper function should not set the status code again. We should improve it later.
	RespondWithJSON(w, 0, response) // Passing 0 because status is already set.
}
