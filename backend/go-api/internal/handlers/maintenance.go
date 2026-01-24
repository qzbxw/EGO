package handlers

import (
	"log"
	"net/http"

	"egobackend/internal/database"

	"github.com/go-chi/chi/v5"
)

// MaintenanceHandler handles HTTP requests related to the application's maintenance mode.
type MaintenanceHandler struct {
	db *database.DB
}

// NewMaintenanceHandler creates a new MaintenanceHandler.
func NewMaintenanceHandler(db *database.DB) *MaintenanceHandler {
	return &MaintenanceHandler{db: db}
}

// RegisterRoutes registers the maintenance-related endpoints with the Chi router.
// It registers routes under both `/maintenance` and `/api/maintenance` to ensure
// compatibility with different frontend proxy configurations.
func (h *MaintenanceHandler) RegisterRoutes(r chi.Router) {
	r.Route("/maintenance", func(r chi.Router) {
		r.Get("/status", h.GetStatus)
		r.Get("/validate", h.Validate)
	})
	r.Route("/api/maintenance", func(r chi.Router) {
		r.Get("/status", h.GetStatus)
		r.Get("/validate", h.Validate)
	})
}

// GetStatus returns the current maintenance mode status of the application.
// This endpoint is always accessible, even when maintenance mode is on.
func (h *MaintenanceHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.db.GetMaintenanceMode()
	if err != nil {
		log.Printf("Error getting maintenance status from DB: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to get maintenance status")
		return
	}

	// Default message if none is set in the database.
	message := "Service is temporarily unavailable due to maintenance."
	if status.Message != nil && *status.Message != "" {
		message = *status.Message
	}

	response := map[string]interface{}{
		"maintenance_enabled": status.IsEnabled,
		"is_chat_only":        status.IsChatOnly,
		"message":             message,
	}
	RespondWithJSON(w, http.StatusOK, response)
}

// Validate checks if a provided bypass token is valid and not expired.
// The token can be provided via the 'X-Bypass-Token' header or 'bypass_token' query parameter.
func (h *MaintenanceHandler) Validate(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Bypass-Token")
	if token == "" {
		token = r.URL.Query().Get("bypass_token")
	}

	if token == "" {
		RespondWithError(w, http.StatusUnauthorized, "Bypass token is missing")
		return
	}

	valid, err := h.db.ValidateBypassToken(token)
	if err != nil {
		log.Printf("Error validating bypass token: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Error during token validation")
		return
	}

	if !valid {
		RespondWithError(w, http.StatusUnauthorized, "Invalid or expired bypass token")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]bool{"valid": true})
}
