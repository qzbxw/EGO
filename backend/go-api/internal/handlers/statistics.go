package handlers

import (
	"log"
	"net/http"

	"egobackend/internal/database"
	"egobackend/internal/models"
)

// StatisticsHandler handles HTTP requests for user and system statistics.
type StatisticsHandler struct {
	DB *database.DB
}

// NewStatisticsHandler creates a new StatisticsHandler.
func NewStatisticsHandler(db *database.DB) *StatisticsHandler {
	return &StatisticsHandler{DB: db}
}

// GetUserStatistics retrieves statistics for the currently authenticated user.
func (h *StatisticsHandler) GetUserStatistics(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)

	stats, err := h.DB.GetUserStatistics(user.ID)
	if err != nil {
		log.Printf("Error getting user statistics for user %d: %v", user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve user statistics")
		return
	}

	RespondWithJSON(w, http.StatusOK, stats)
}

// GetUserStatisticsByID retrieves statistics for a specific user by their ID.
// This endpoint is protected and only accessible by administrators.
func (h *StatisticsHandler) GetUserStatisticsByID(w http.ResponseWriter, r *http.Request) {
	userID, err := parseIDFromURL(r, "userID")
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	stats, err := h.DB.GetUserStatistics(int(userID))
	if err != nil {
		log.Printf("Error getting statistics for user ID %d: %v", userID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve user statistics")
		return
	}

	RespondWithJSON(w, http.StatusOK, stats)
}

// GetGlobalStatistics retrieves system-wide statistics.
// This endpoint is protected and only accessible by administrators.
func (h *StatisticsHandler) GetGlobalStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.DB.GetGlobalStatistics()
	if err != nil {
		log.Printf("Error getting global statistics: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve global statistics")
		return
	}

	RespondWithJSON(w, http.StatusOK, stats)
}

// GetAllUsersStatistics retrieves statistics for all users in the system.
// This endpoint is protected and only accessible by administrators.
func (h *StatisticsHandler) GetAllUsersStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.DB.GetAllUsersStatistics()
	if err != nil {
		log.Printf("Error getting all users statistics: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve all users statistics")
		return
	}

	RespondWithJSON(w, http.StatusOK, stats)
}

// GetUserProviderTokens retrieves the token usage aggregated by provider for the current user.
func (h *StatisticsHandler) GetUserProviderTokens(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)

	tokenUsage, err := h.DB.GetUserProviderTokenUsage(user.ID)
	if err != nil {
		log.Printf("Error getting provider token usage for user %d: %v", user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve provider token usage")
		return
	}

	RespondWithJSON(w, http.StatusOK, tokenUsage)
}

// GetPublicStats provides a small, curated set of system-wide statistics
// for public display (e.g., on a landing page). No authentication is required.
func (h *StatisticsHandler) GetPublicStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.DB.GetGlobalStatistics()
	if err != nil {
		log.Printf("Error getting global statistics for public stats: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Could not retrieve public statistics")
		return
	}

	// Expose only a safe subset of the global statistics.
	publicStats := map[string]interface{}{
		"total_tokens":   stats.TotalTokens,
		"total_sessions": stats.TotalSessions,
		"total_requests": stats.TotalRequests,
	}

	// Set permissive CORS headers for public, unauthenticated endpoints.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	RespondWithJSON(w, http.StatusOK, publicStats)
}
