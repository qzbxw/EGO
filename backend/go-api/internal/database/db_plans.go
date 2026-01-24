package database

import (
	"database/sql"
	"fmt"
	"time"

	"egobackend/internal/models"
)

// GetActivePlan returns the currently active plan for a session, including its steps.
func (db *DB) GetActivePlan(sessionUUID string) (*models.SessionPlan, error) {
	var plan models.SessionPlan
	err := db.Get(&plan, "SELECT * FROM session_plans WHERE session_uuid = $1 AND is_active = true LIMIT 1", sessionUUID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active plan: %w", err)
	}

	var steps []models.PlanStep
	err = db.Select(&steps, "SELECT * FROM plan_steps WHERE plan_id = $1 ORDER BY step_order ASC", plan.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan steps: %w", err)
	}

	plan.Steps = steps
	return &plan, nil
}

// CreatePlan creates a new active plan and its steps.
// If an active plan already exists, it is marked as inactive (archived).
func (db *DB) CreatePlan(sessionUUID string, title string, steps []string) (*models.SessionPlan, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Deactivate existing plans
	_, err = tx.Exec("UPDATE session_plans SET is_active = false WHERE session_uuid = $1 AND is_active = true", sessionUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate old plans: %w", err)
	}

	// Create new plan
	var planID int
	err = tx.QueryRow("INSERT INTO session_plans (session_uuid, title, is_active) VALUES ($1, $2, true) RETURNING id", sessionUUID, title).Scan(&planID)
	if err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	// Create steps
	createdSteps := make([]models.PlanStep, 0, len(steps))
	for i, stepDesc := range steps {
		var step models.PlanStep
		err = tx.QueryRowx(`
			INSERT INTO plan_steps (plan_id, description, status, step_order) 
			VALUES ($1, $2, 'pending', $3) 
			RETURNING *`, planID, stepDesc, i+1).StructScan(&step)
		if err != nil {
			return nil, fmt.Errorf("failed to create step %d: %w", i, err)
		}
		createdSteps = append(createdSteps, step)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.SessionPlan{
		ID:          planID,
		SessionUUID: sessionUUID,
		Title:       title,
		IsActive:    true,
		Steps:       createdSteps,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// UpdateStepStatus updates the status of a specific step.
func (db *DB) UpdateStepStatus(planID int, stepOrder int, status string) error {
	// Validate status
	validStatuses := map[string]bool{"pending": true, "in_progress": true, "completed": true, "failed": true, "skipped": true}
	if !validStatuses[status] {
		return fmt.Errorf("invalid step status: %s", status)
	}

	_, err := db.Exec("UPDATE plan_steps SET status = $1, updated_at = NOW() WHERE plan_id = $2 AND step_order = $3", status, planID, stepOrder)
	if err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}
	return nil
}

// UpdateStepDescription updates the description of a specific step.
func (db *DB) UpdateStepDescription(planID int, stepOrder int, description string) error {
	_, err := db.Exec("UPDATE plan_steps SET description = $1, updated_at = NOW() WHERE plan_id = $2 AND step_order = $3", description, planID, stepOrder)
	if err != nil {
		return fmt.Errorf("failed to update step description: %w", err)
	}
	return nil
}

// MarkPlanComplete marks the plan as inactive (completed).
func (db *DB) MarkPlanComplete(planID int) error {
	_, err := db.Exec("UPDATE session_plans SET is_active = false, updated_at = NOW() WHERE id = $1", planID)
	return err
}
