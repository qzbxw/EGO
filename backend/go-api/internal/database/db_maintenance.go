// This file defines the MaintenanceMode model and provides database methods
// for managing the application's maintenance state.

package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	// bypassTokenLifetime specifies how long a bypass token is valid after creation.
	bypassTokenLifetime = 24 * time.Hour
	// bypassTokenMinLength is the minimum expected length for a token to be considered valid.
	bypassTokenMinLength = 16
	// bypassTokenEntropyBytes is the number of random bytes for generating the token.
	bypassTokenEntropyBytes = 32
	// bypassTokenPrefix helps identify the token type in logs or databases.
	bypassTokenPrefix = "ego_bypass_"
)

// MaintenanceMode represents the state of the application's maintenance mode.
type MaintenanceMode struct {
	ID          int        `db:"id" json:"id"`
	IsEnabled   bool       `db:"is_enabled" json:"is_enabled"`
	IsChatOnly  bool       `db:"is_chat_only" json:"is_chat_only"`
	Message     *string    `db:"message" json:"message,omitempty"`
	BypassToken *string    `db:"bypass_token" json:"bypass_token,omitempty"`
	EnabledAt   *time.Time `db:"enabled_at" json:"enabled_at,omitempty"`
	DisabledAt  *time.Time `db:"disabled_at" json:"disabled_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
}

// GetMaintenanceMode retrieves the current maintenance mode status from the database.
func (db *DB) GetMaintenanceMode() (*MaintenanceMode, error) {
	var maintenance MaintenanceMode
	query := `
        SELECT id, is_enabled, is_chat_only, message, bypass_token, enabled_at, disabled_at, created_at, updated_at
        FROM maintenance_mode
        ORDER BY id DESC
        LIMIT 1`

	err := db.Get(&maintenance, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance mode status: %w", err)
	}

	return &maintenance, nil
}

// IsMaintenanceModeEnabled checks if maintenance mode is currently active.
func (db *DB) IsMaintenanceModeEnabled() (bool, error) {
	var maintenance struct {
		IsEnabled  bool `db:"is_enabled"`
		IsChatOnly bool `db:"is_chat_only"`
	}
	query := `SELECT is_enabled, is_chat_only FROM maintenance_mode ORDER BY id DESC LIMIT 1`

	err := db.Get(&maintenance, query)
	if err != nil {
		return false, fmt.Errorf("failed to check maintenance mode: %w", err)
	}

	// For general middleware, only full is_enabled = true without is_chat_only should block everything.
	// If it's chat-only maintenance, we allow the request to pass to handlers.
	return maintenance.IsEnabled && !maintenance.IsChatOnly, nil
}

// EnableMaintenanceMode activates maintenance mode with a given message and generates a new bypass token.
func (db *DB) EnableMaintenanceMode(message string, chatOnly bool) (string, error) {
	bypassToken, err := generateBypassToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate bypass token: %w", err)
	}

	now := time.Now().UTC()
	query := `
        UPDATE maintenance_mode SET
            is_enabled = true,
            is_chat_only = $4,
            bypass_token = $1,
            enabled_at = $2,
            updated_at = $2,
            message = $3
        WHERE id = (SELECT id FROM maintenance_mode ORDER BY id DESC LIMIT 1)`

	_, err = db.Exec(query, bypassToken, now, message, chatOnly)
	if err != nil {
		return "", fmt.Errorf("failed to enable maintenance mode: %w", err)
	}

	return bypassToken, nil
}

// DisableMaintenanceMode deactivates maintenance mode.
func (db *DB) DisableMaintenanceMode() error {
	now := time.Now().UTC()
	query := `
        UPDATE maintenance_mode SET
            is_enabled = false,
            is_chat_only = false,
            bypass_token = NULL,
            disabled_at = $1,
            updated_at = $1,
            message = NULL
        WHERE id = (SELECT id FROM maintenance_mode ORDER BY id DESC LIMIT 1)`

	_, err := db.Exec(query, now)
	if err != nil {
		return fmt.Errorf("failed to disable maintenance mode: %w", err)
	}
	return nil
}

// ValidateBypassToken checks if a given token is a valid and non-expired bypass token.
func (db *DB) ValidateBypassToken(token string) (bool, error) {
	if len(token) < bypassTokenMinLength {
		return false, nil
	}

	var isValid bool
	// This query checks for an active, non-null token that was enabled within the allowed lifetime.
	query := `
        SELECT EXISTS (
            SELECT 1 FROM maintenance_mode
            WHERE is_enabled = true
              AND bypass_token = $1
              AND bypass_token IS NOT NULL
              AND enabled_at >= NOW() - $2::interval
            ORDER BY id DESC
            LIMIT 1
        )`

	// Format the lifetime into a string representation for the interval.
	interval := fmt.Sprintf("%f seconds", bypassTokenLifetime.Seconds())
	err := db.Get(&isValid, query, token, interval)

	if err != nil {
		// sql.ErrNoRows is not expected with SELECT EXISTS, but we handle it just in case.
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to validate bypass token: %w", err)
	}

	return isValid, nil
}

// generateBypassToken creates a cryptographically secure random token for bypassing maintenance mode.
func generateBypassToken() (string, error) {
	bytes := make([]byte, bypassTokenEntropyBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes for token: %w", err)
	}

	return bypassTokenPrefix + hex.EncodeToString(bytes), nil
}
