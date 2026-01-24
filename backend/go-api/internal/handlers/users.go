package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"egobackend/internal/database"
	"egobackend/internal/models"

	"github.com/go-playground/validator/v10"
)

// UserHandler handles HTTP requests related to user settings and account management.
type UserHandler struct {
	DB            *database.DB
	EncryptionKey string
	validate      *validator.Validate
}

// NewUserHandler creates a new UserHandler.
// It initializes a validator instance to be reused across requests.
func NewUserHandler(db *database.DB, encryptionKey string) *UserHandler {
	return &UserHandler{
		DB:            db,
		EncryptionKey: encryptionKey,
		validate:      validator.New(),
	}
}

// GetSettings retrieves the LLM settings for the authenticated user.
// Note: It intentionally does not return the API key for security reasons.
func (h *UserHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)

	// We fetch the full user object to ensure we have the latest settings.
	fullUser, err := h.DB.GetUserByID(user.ID)
	if err != nil {
		log.Printf("[UserHandler] Error fetching user %d: %v", user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Could not fetch user settings")
		return
	}

	response := models.UserLLMSettingsResponse{
		LLMProvider: "ego", // Default value
	}
	if fullUser.LLMProvider != nil {
		response.LLMProvider = *fullUser.LLMProvider
	}
	if fullUser.LLMModel != nil {
		response.LLMModel = *fullUser.LLMModel
	}

	RespondWithJSON(w, http.StatusOK, response)
}

// UpdateSettings validates and saves new LLM settings for the authenticated user.
// The API key is encrypted before being stored in the database.
func (h *UserHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)

	var settings models.UserLLMSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	if err := h.validate.Struct(settings); err != nil {
		// Provide a more user-friendly validation error message if possible.
		RespondWithError(w, http.StatusBadRequest, "Validation failed: "+err.Error())
		return
	}

	if err := h.DB.UpdateUserSettings(user.ID, settings, h.EncryptionKey); err != nil {
		log.Printf("[UserHandler] Error saving settings for user %d: %v", user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Could not save user settings")
		return
	}

	log.Printf("[UserHandler] Settings successfully saved for user %d", user.ID)
	w.WriteHeader(http.StatusNoContent)
}

// DeleteSettings resets the LLM settings for the authenticated user to their default values.
func (h *UserHandler) DeleteSettings(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)

	if err := h.DB.DeleteUserSettings(user.ID); err != nil {
		log.Printf("[UserHandler] Error deleting settings for user %d: %v", user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Could not delete user settings")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteAccount handles the permanent deletion of the authenticated user's account
// and all associated data (sessions, logs, files, etc.). This is an irreversible action.
func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(UserContextKey).(*models.User)

	// First, delete all user sessions. The database's ON DELETE CASCADE constraints
	// will handle the removal of all related data.
	if err := h.DB.DeleteAllSessionsForUser(user.ID); err != nil {
		log.Printf("Failed to delete sessions for user %d during account deletion: %v", user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete user's sessions")
		return
	}

	// Finally, delete the user record itself.
	if err := h.DB.DeleteUser(user.ID); err != nil {
		log.Printf("Failed to delete user record for user %d during account deletion: %v", user.ID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete user account")
		return
	}

	log.Printf("User account %d (%s) deleted successfully.", user.ID, user.Username)
	w.WriteHeader(http.StatusNoContent)
}
