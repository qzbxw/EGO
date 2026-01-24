// Package handlers contains the HTTP handlers for the application's API endpoints.
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"egobackend/internal/auth"
	"egobackend/internal/database"
	"egobackend/internal/models"
	"egobackend/internal/telemetry"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

// UserContextKey is the key used to store the user object in the request context.
const UserContextKey = ContextKey("user")

// AuthHandler handles all authentication-related HTTP requests.
type AuthHandler struct {
	DB             *database.DB
	AuthService    *auth.AuthService
	GoogleClientID string
}

// AuthMiddleware is a middleware that validates a JWT token and injects the user
// into the request context. It handles tokens from both 'Authorization' headers
// and 'token' query parameters (for WebSocket connections).
func (h *AuthHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := extractToken(r)
		if tokenString == "" {
			RespondWithError(w, http.StatusUnauthorized, "Authorization token is missing")
			return
		}

		username, err := h.AuthService.ValidateJWT(tokenString)
		if err != nil {
			log.Printf("Token validation failed for %s: %v", r.URL.Path, err)
			RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		user, err := h.DB.GetUserByUsername(username)
		if err != nil {
			if err == sql.ErrNoRows {
				RespondWithError(w, http.StatusUnauthorized, "User from token not found")
			} else {
				log.Printf("Server error looking up user '%s': %v", username, err)
				RespondWithError(w, http.StatusInternalServerError, "Server error while looking up user")
			}
			return
		}

		// Log user ID and path for debugging token/user mismatch issues
		if strings.Contains(r.URL.Path, "/sessions") || strings.Contains(r.URL.Path, "/ws") {
			log.Printf("[AuthMiddleware] %s %s - User ID: %d (username: %s), token: %s...",
				r.Method, r.URL.Path, user.ID, user.Username, tokenString[:min(20, len(tokenString))])
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Login handles user login with a username and password.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}
	if req.Username == "" || req.Password == "" {
		RespondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	user, err := h.DB.GetUserByUsername(req.Username)
	if err != nil || user.Provider != "password" {
		log.Printf("Login failed for user '%s': user not found or not a password-based account. IP: %s", req.Username, getClientIP(r))
		telemetry.LogAuthFailure(req.Username, "User not found or invalid provider", getClientIP(r))
		RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	if !auth.CheckPasswordHash(req.Password, user.HashedPassword) {
		log.Printf("Login failed for user '%s': invalid password. IP: %s", req.Username, getClientIP(r))
		telemetry.LogAuthFailure(req.Username, "Invalid password", getClientIP(r))
		RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	h.issueTokens(w, user)
	telemetry.LogAuthSuccess(user.Username, int(user.ID), getClientIP(r))
	log.Printf("User '%s' logged in successfully.", user.Username)
}

// Register handles the creation of a new user account.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}
	if req.Username == "" || req.Password == "" {
		RespondWithError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	_, err := h.DB.GetUserByUsername(req.Username)
	if err == nil {
		RespondWithError(w, http.StatusConflict, "A user with this username already exists")
		return
	}
	if err != sql.ErrNoRows {
		log.Printf("Server error checking for user existence: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Server error while checking for user")
		return
	}

	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Server error hashing password: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Server error while hashing password")
		return
	}

	newUser, err := h.DB.CreateUser(req.Username, hashedPassword)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	log.Printf("New user registered: %s (ID: %d)", newUser.Username, newUser.ID)
	response := models.UserResponse{ID: newUser.ID, Username: newUser.Username, Role: newUser.Role, CreatedAt: newUser.CreatedAt}
	RespondWithJSON(w, http.StatusCreated, response)
}

// Refresh issues a new access token using a valid refresh token.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	username, err := h.AuthService.ValidateJWT(req.RefreshToken)
	if err != nil {
		log.Printf("Invalid refresh token received: %v", err)
		telemetry.LogTokenRefresh(0, false, getClientIP(r))
		RespondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	user, err := h.DB.GetUserByUsername(username)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "User from token not found")
		return
	}

	newAccessToken, err := h.AuthService.CreateAccessToken(user.Username, user.Role)
	if err != nil {
		log.Printf("Failed to create new access token for user '%s': %v", user.Username, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to create new access token")
		return
	}

	response := models.RefreshResponse{AccessToken: newAccessToken}
	RespondWithJSON(w, http.StatusOK, response)

	telemetry.LogTokenRefresh(int(user.ID), true, getClientIP(r))
	log.Printf("Token for user '%s' was successfully refreshed.", user.Username)
}

// Me returns the details of the currently authenticated user.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		RespondWithError(w, http.StatusInternalServerError, "Could not retrieve user from context")
		return
	}
	response := models.UserResponse{ID: user.ID, Username: user.Username, Role: user.Role, CreatedAt: user.CreatedAt}
	RespondWithJSON(w, http.StatusOK, response)
}

// GoogleLogin handles user sign-in via a Google ID token.
func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.GoogleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	payload, err := h.AuthService.ValidateGoogleJWT(req.Token, h.GoogleClientID)
	if err != nil {
		log.Printf("Google token verification failed: %v", err)
		telemetry.LogGoogleAuth("", 0, false, getClientIP(r))
		RespondWithError(w, http.StatusUnauthorized, "Invalid Google token")
		return
	}

	user, err := h.DB.FindOrCreateGoogleUser(payload.Email, payload.Subject)
	if err != nil {
		log.Printf("!!! CRITICAL ERROR in FindOrCreateGoogleUser: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to find or create user")
		return
	}

	h.issueTokens(w, user)
	telemetry.LogGoogleAuth(user.Username, int(user.ID), true, getClientIP(r))
	log.Printf("User '%s' successfully logged in via Google.", user.Username)
}

// issueTokens is a helper function to generate and return access and refresh tokens.
func (h *AuthHandler) issueTokens(w http.ResponseWriter, user *models.User) {
	accessToken, err := h.AuthService.CreateAccessToken(user.Username, user.Role)
	if err != nil {
		log.Printf("Failed to create access token for user '%s': %v", user.Username, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to create access token")
		return
	}
	refreshToken, err := h.AuthService.CreateRefreshToken(user.Username)
	if err != nil {
		log.Printf("Failed to create refresh token for user '%s': %v", user.Username, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to create refresh token")
		return
	}

	response := map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          models.UserResponse{ID: user.ID, Username: user.Username, Role: user.Role, CreatedAt: user.CreatedAt},
	}
	RespondWithJSON(w, http.StatusOK, response)
}

// extractToken retrieves the JWT from the Authorization header or the 'token' query parameter.
func extractToken(r *http.Request) string {
	// For WebSocket, the token is passed as a query parameter.
	if strings.Contains(r.URL.Path, "/ws") {
		return r.URL.Query().Get("token")
	}

	// For standard HTTP requests, the token is in the Authorization header.
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}
