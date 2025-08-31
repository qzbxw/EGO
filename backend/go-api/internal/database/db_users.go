// This file contains database methods related to user management.

package database

import (
	"fmt"
	"strings"

	"egobackend/internal/crypto"
	"egobackend/internal/models"
)

// CreateUser creates a new user with a username and hashed password.
func (db *DB) CreateUser(username, hashedPassword string) (*models.User, error) {
	// Explicitly list columns for safety and clarity instead of using 'RETURNING *'.
	query := `
        INSERT INTO users (username, hashed_password, provider)
        VALUES ($1, $2, 'password')
        RETURNING id, username, role, provider, provider_id, created_at,
                  llm_provider, llm_model, encrypted_api_key`
	var newUser models.User
	err := db.Get(&newUser, query, username, hashedPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &newUser, nil
}

// FindOrCreateGoogleUser finds a user by their Google provider ID or creates a new one.
// If a user with the same email already exists (as a local account), it links the Google ID to it.
// This is achieved in a single, atomic, and non-locking query using ON CONFLICT.
func (db *DB) FindOrCreateGoogleUser(email, providerID string) (*models.User, error) {
	var user models.User
	// This single query handles three cases atomically:
	// 1. A user with this google provider_id already exists -> DO NOTHING, then the outer SELECT finds them.
	// 2. A user with this email (username) exists, but not linked to google -> DO UPDATE, link the account.
	// 3. No conflicting user exists -> The initial INSERT succeeds.
	query := `
        WITH ins AS (
            INSERT INTO users (username, provider, provider_id)
            VALUES ($1, 'google', $2)
            ON CONFLICT (username) DO UPDATE
                SET provider = 'google', provider_id = EXCLUDED.provider_id, hashed_password = NULL
                WHERE users.provider = 'password' -- Only overwrite local password accounts
            RETURNING id
        )
        SELECT id, username, role, provider, provider_id, created_at,
               llm_provider, llm_model, encrypted_api_key
        FROM users
        WHERE id = (
            SELECT id FROM ins
            UNION ALL
            SELECT id FROM users WHERE provider = 'google' AND provider_id = $2 AND NOT EXISTS (SELECT 1 FROM ins)
            LIMIT 1
        )`
	err := db.Get(&user, query, email, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create google user: %w", err)
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by their username.
// Returns sql.ErrNoRows if the user is not found.
func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE username = $1`
	err := db.Get(&user, query, username)
	if err != nil {
		return nil, err // The caller should handle sql.ErrNoRows
	}
	return &user, nil
}

// GetUserByID retrieves a user by their ID.
// Returns sql.ErrNoRows if the user is not found.
func (db *DB) GetUserByID(userID int) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = $1`
	err := db.Get(&user, query, userID)
	if err != nil {
		return nil, err // The caller should handle sql.ErrNoRows
	}
	return &user, nil
}

// UpdateUserRole updates the role for a specific user.
func (db *DB) UpdateUserRole(userID int, newRole string) error {
	query := `UPDATE users SET role = $1 WHERE id = $2`
	_, err := db.Exec(query, newRole, userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	return nil
}

// UpdateUserSettings saves LLM provider settings for a user, encrypting the API key.
func (db *DB) UpdateUserSettings(userID int, settings models.UserLLMSettings, encryptionKey string) error {
	var modelPtr *string
	trimmedModel := strings.TrimSpace(settings.LLMModel)
	if trimmedModel != "" {
		modelPtr = &trimmedModel
	}

	var encryptedKeyPtr *string
	trimmedKey := strings.TrimSpace(settings.APIKey)
	if trimmedKey != "" {
		encryptedKey, err := crypto.Encrypt(trimmedKey, encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt API key: %w", err)
		}
		encryptedKeyPtr = &encryptedKey
	}

	query := `UPDATE users SET llm_provider = $1, llm_model = $2, encrypted_api_key = $3 WHERE id = $4`
	_, err := db.Exec(query, settings.LLMProvider, modelPtr, encryptedKeyPtr, userID)
	if err != nil {
		return fmt.Errorf("failed to update user settings: %w", err)
	}
	return nil
}

// DeleteUserSettings resets a user's LLM settings to their default values.
func (db *DB) DeleteUserSettings(userID int) error {
	// Resets settings to application defaults ('ego' provider and no model/key).
	query := `UPDATE users SET llm_provider = 'ego', llm_model = NULL, encrypted_api_key = NULL WHERE id = $1`
	_, err := db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user settings: %w", err)
	}
	return nil
}

// DeleteUser permanently deletes a user and all their associated data via cascading deletes.
func (db *DB) DeleteUser(userID int) error {
	_, err := db.Exec(`DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}