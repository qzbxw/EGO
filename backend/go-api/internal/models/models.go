// Package models defines the core data structures used throughout the application,
// representing database entities, API request/response bodies, and internal data contracts.
package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// --- API Request Payloads ---

// StreamRequest is the main structure for a client's request over WebSocket.
type StreamRequest struct {
	Query               string        `json:"query" validate:"max=200000"`
	Mode                string        `json:"mode" validate:"required"`
	SessionUUID         *string       `json:"session_uuid,omitempty"`
	Files               []FilePayload `json:"files,omitempty" validate:"max=5,dive"`
	CustomInstructions  *string       `json:"custom_instructions,omitempty"`
	IsRegeneration      bool          `json:"is_regeneration,omitempty"`
	RequestLogIDToRegen int64         `json:"request_log_id_to_regen,omitempty"`
	TempID              int64         `json:"temp_id,omitempty"` // A temporary ID assigned by the client.
	MemoryEnabled       *bool         `json:"memory_enabled,omitempty"`
	UploadID            *string       `json:"upload_id,omitempty"` // ID for files uploaded via multipart.
}

// FilePayload represents a file sent inline from the client, typically as Base64.
type FilePayload struct {
	Base64Data string `json:"base64_data" validate:"required"`
	MimeType   string `json:"mime_type" validate:"required"`
	FileName   string `json:"file_name" validate:"required"`
}

// AuthRequest is used for user login and registration.
type AuthRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RefreshTokenRequest is used to request a new access token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// GoogleAuthRequest is used for authentication via a Google ID token.
type GoogleAuthRequest struct {
	Token string `json:"token" validate:"required"`
}

// UpdateLogRequest is used to edit the user query of an existing log.
type UpdateLogRequest struct {
	Query string `json:"query" validate:"required"`
}

// UserLLMSettings defines the structure for updating a user's LLM provider settings.
type UserLLMSettings struct {
	LLMProvider string `json:"llm_provider" validate:"required,oneof=ego openai anthropic gemini grok openrouter_free openrouter_billing"`
	LLMModel    string `json:"llm_model" validate:"omitempty"`
	APIKey      string `json:"api_key" validate:"omitempty"`
}

// CreateSessionRequest defines the optional body for creating a new session.
type CreateSessionRequest struct {
	Mode               string  `json:"mode"`
	Title              *string `json:"title"`
	CustomInstructions *string `json:"custom_instructions"`
}

// UpdateSessionRequest defines the body for updating a session's metadata.
type UpdateSessionRequest struct {
	Title              *string `json:"title"`
	CustomInstructions *string `json:"custom_instructions"`
}

// --- Internal & Service-to-Service Models ---

// PythonRequest defines the data structure sent to the Python backend for processing.
type PythonRequest struct {
	Query              string        `json:"query"`
	Mode               string        `json:"mode"`
	ChatHistory        string        `json:"chat_history"`
	ThoughtsHistory    string        `json:"thoughts_history,omitempty"`
	CustomInstructions *string       `json:"custom_instructions,omitempty"`
	Files              []FilePayload `json:"files,omitempty"`        // Files sent inline
	CachedFiles        []CachedFile  `json:"cached_files,omitempty"` // Files referenced from storage
	UserID             string        `json:"user_id,omitempty"`
	SessionUUID        string        `json:"session_uuid,omitempty"`
	LLMSettings        *LLMSettings  `json:"llm_settings,omitempty"`
	MemoryEnabled      bool          `json:"memory_enabled,omitempty"`
	LogID              int64         `json:"log_id,omitempty"`
	RegeneratedLogID   int64         `json:"regenerated_log_id,omitempty"`
	SessionCreatedAt   string        `json:"session_created_at,omitempty"`
	CurrentPlan        *SessionPlan  `json:"current_plan,omitempty"` // Inject active plan into the prompt
	CurrentDate        string        `json:"current_date,omitempty"` // Inject current date into the prompt
	UserProfile        string        `json:"user_profile,omitempty"`
}

// SessionPlan represents a high-level plan for a session.
type SessionPlan struct {
	ID          int        `db:"id" json:"id"`
	SessionUUID string     `db:"session_uuid" json:"session_uuid"`
	Title       string     `db:"title" json:"title"`
	IsActive    bool       `db:"is_active" json:"is_active"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	Steps       []PlanStep `json:"steps"`
}

// PlanStep represents a single step in a session plan.
type PlanStep struct {
	ID          int       `db:"id" json:"id"`
	PlanID      int       `db:"plan_id" json:"plan_id"`
	Description string    `db:"description" json:"description"`
	Status      string    `db:"status" json:"status"` // pending, in_progress, completed, failed, skipped
	StepOrder   int       `db:"step_order" json:"step_order"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// CachedFile represents a file that already exists in storage and is referenced by its URI.
type CachedFile struct {
	URI      string `json:"URI"`
	FileName string `json:"FileName"`
	MimeType string `json:"MimeType"`
}

// LLMSettings mirrors the structure expected by the Python backend for custom provider settings.
type LLMSettings struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

// ThoughtResponse represents the structured 'thought' part of the Python service's response.
type ThoughtResponse struct {
	Thoughts          string     `json:"thoughts"`
	ToolReasoning     string     `json:"tool_reasoning"`
	ToolCalls         []ToolCall `json:"tool_calls"`
	ThoughtHeader     string     `json:"thoughts_header"`
	NextThoughtNeeded bool       `json:"next_thought_needed"`
	ConfidenceScore   float64    `json:"confidence_score"`
	SelfCritique      string     `json:"self_critique"`
	PlanStatus        string     `json:"plan_status"`
}

// ToolCall represents a single tool execution request from the LLM.
type ToolCall struct {
	ToolName  string `json:"tool_name"`
	ToolQuery string `json:"tool_query"`
}

// ThoughtResponseWithData is the full response from the non-streaming /generate_thought endpoint.
type ThoughtResponseWithData struct {
	Thought     ThoughtResponse          `json:"thought"`
	ToolResults []map[string]interface{} `json:"tool_results"`
	Usage       map[string]interface{}   `json:"usage"`
}

// StreamEvent represents a single Server-Sent Event (SSE) from the Python stream.
type StreamEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// --- Database Models ---

// User represents a user record in the 'users' table.
type User struct {
	ID              int        `db:"id" json:"id"`
	Username        string     `db:"username" json:"username"`
	HashedPassword  *string    `db:"hashed_password" json:"-"`
	Provider        string     `db:"provider" json:"provider"`
	ProviderID      *string    `db:"provider_id" json:"-"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	Role            string     `db:"role" json:"role"`
	LLMProvider     *string    `db:"llm_provider" json:"llm_provider,omitempty"`
	LLMModel        *string    `db:"llm_model" json:"llm_model,omitempty"`
	EncryptedAPIKey *string    `db:"encrypted_api_key" json:"-"` // Never send to frontend
	ProfileSummary  string     `db:"profile_summary" json:"profile_summary"`
	LastSummaryAt   *time.Time `db:"last_summary_at" json:"-"`
}

// ChatSession represents a record in the 'chat_sessions' table.
type ChatSession struct {
	UUID               string    `db:"uuid"`
	UserID             int       `db:"user_id"`
	Title              string    `db:"title"`
	Mode               string    `db:"mode"`
	CustomInstructions *string   `db:"custom_instructions"`
	CreatedAt          time.Time `db:"created_at"`
}

// RequestLog represents a record in the 'request_logs' table.
type RequestLog struct {
	ID               int            `db:"id"`
	SessionUUID      string         `db:"session_uuid"`
	UserQuery        string         `db:"user_query"`
	EgoThoughtsJSON  string         `db:"ego_thoughts_json"`
	FinalResponse    *string        `db:"final_response"`
	PromptTokens     int            `db:"prompt_tokens"`
	CompletionTokens int            `db:"completion_tokens"`
	TotalTokens      int            `db:"total_tokens"`
	AttachedFileIDs  sql.NullString `db:"attached_file_ids"`
	Timestamp        time.Time      `db:"timestamp"`
}

// FileAttachment represents a record in the 'file_attachments' table.
type FileAttachment struct {
	ID           int64          `db:"id"`
	SessionUUID  string         `db:"session_uuid"`
	UserID       int            `db:"user_id"`
	RequestLogID sql.NullInt64  `db:"request_log_id"`
	FileName     string         `db:"file_name"`
	FileURI      string         `db:"file_uri"`
	MimeType     string         `db:"mime_type"`
	Status       string         `db:"status"`
	CreatedAt    time.Time      `db:"created_at"`
	UploadID     sql.NullString `db:"upload_id"`
}

// --- API Response Payloads (DTOs) ---

// UserResponse is the safe representation of a User object sent to clients.
type UserResponse struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionResponse is the representation of a ChatSession sent to clients.
type SessionResponse struct {
	UUID               string    `json:"uuid"`
	Title              string    `json:"title"`
	Mode               string    `json:"mode"`
	CustomInstructions *string   `json:"custom_instructions,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

// FileAttachmentResponse is the representation of a FileAttachment within a log.
type FileAttachmentResponse struct {
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
}

// LogResponse is the representation of a RequestLog sent to clients.
type LogResponse struct {
	ID            int                      `json:"id"`
	LogID         int                      `json:"logId"`
	UserQuery     string                   `json:"user_query"`
	FinalResponse *string                  `json:"final_response,omitempty"`
	Thoughts      json.RawMessage          `json:"thoughts,omitempty"`
	Timestamp     time.Time                `json:"timestamp"`
	Attachments   []FileAttachmentResponse `json:"attachments"`
}

// RefreshResponse contains a new access token.
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

// UserLLMSettingsResponse is the safe representation of a user's LLM settings.
type UserLLMSettingsResponse struct {
	LLMProvider string `json:"llm_provider"`
	LLMModel    string `json:"llm_model"`
}

// --- DTO Constructor Functions ---

// ToSessionResponse converts a ChatSession DB model to its API response representation.
func ToSessionResponse(s *ChatSession) SessionResponse {
	return SessionResponse{
		UUID:               s.UUID,
		Title:              s.Title,
		Mode:               s.Mode,
		CustomInstructions: s.CustomInstructions,
		CreatedAt:          s.CreatedAt,
	}
}

// ToSessionResponseList converts a slice of ChatSession DB models to a slice of API responses.
func ToSessionResponseList(sessions []ChatSession) []SessionResponse {
	response := make([]SessionResponse, len(sessions))
	for i, s := range sessions {
		response[i] = ToSessionResponse(&s)
	}
	return response
}

// ToLogResponseList converts a slice of RequestLog DB models to a slice of API responses.
func ToLogResponseList(logs []RequestLog, attachmentsMap map[int][]FileAttachment) []LogResponse {
	response := make([]LogResponse, len(logs))
	for i, l := range logs {
		var attachments []FileAttachmentResponse
		if atts, ok := attachmentsMap[l.ID]; ok {
			for _, att := range atts {
				attachments = append(attachments, FileAttachmentResponse{
					FileName: att.FileName,
					MimeType: att.MimeType,
				})
			}
		}

		var thoughts json.RawMessage
		if l.EgoThoughtsJSON != "" {
			thoughts = json.RawMessage(l.EgoThoughtsJSON)
		}

		response[i] = LogResponse{
			ID:            l.ID,
			LogID:         l.ID,
			UserQuery:     l.UserQuery,
			FinalResponse: l.FinalResponse,
			Thoughts:      thoughts,
			Timestamp:     l.Timestamp,
			Attachments:   attachments,
		}
	}
	return response
}

// --- Configuration & Statistics Models ---

// S3Config holds the configuration for connecting to an S3-compatible service.
type S3Config struct {
	Endpoint string
	Region   string
	KeyID    string
	AppKey   string
	Bucket   string
}

// RequestMetrics represents a record in the 'request_metrics' table.
type RequestMetrics struct {
	ID                 int       `db:"id"`
	UserID             int       `db:"user_id"`
	SessionUUID        string    `db:"session_uuid"`
	RequestLogID       int64     `db:"request_log_id"`
	ResponseTimeMs     int       `db:"response_time_ms"`
	ThinkingTimeMs     int       `db:"thinking_time_ms"`
	SynthesisTimeMs    int       `db:"synthesis_time_ms"`
	ThinkingIterations int       `db:"thinking_iterations"`
	IsRegeneration     bool      `db:"is_regeneration"`
	MemoryEnabled      bool      `db:"memory_enabled"`
	FilesCount         int       `db:"files_count"`
	FilesSizeBytes     int64     `db:"files_size_bytes"`
	LLMProvider        string    `db:"llm_provider"`
	LLMModel           *string   `db:"llm_model"`
	PromptTokens       int       `db:"prompt_tokens"`
	CompletionTokens   int       `db:"completion_tokens"`
	TotalTokens        int       `db:"total_tokens"`
	CreatedAt          time.Time `db:"created_at"`
}

// UserStatistics represents a record in the 'user_statistics' table.
type UserStatistics struct {
	UserID                  int             `db:"user_id"`
	TotalRequests           int             `db:"total_requests"`
	TotalSessions           int             `db:"total_sessions"`
	TotalFilesUploaded      int             `db:"total_files_uploaded"`
	TotalFilesSizeBytes     int64           `db:"total_files_size_bytes"`
	TotalPromptTokens       int64           `db:"total_prompt_tokens"`
	TotalCompletionTokens   int64           `db:"total_completion_tokens"`
	TotalTokens             int64           `db:"total_tokens"`
	TotalResponseTimeMs     int64           `db:"total_response_time_ms"`
	MinResponseTimeMs       int             `db:"min_response_time_ms"`
	MaxResponseTimeMs       int             `db:"max_response_time_ms"`
	TotalThinkingIterations int             `db:"total_thinking_iterations"`
	TotalRegenerations      int             `db:"total_regenerations"`
	MemoryEnabledRequests   int             `db:"memory_enabled_requests"`
	ProviderUsage           json.RawMessage `db:"provider_usage"`
	ModelUsage              json.RawMessage `db:"model_usage"`
	FirstRequestAt          *time.Time      `db:"first_request_at"`
	LastRequestAt           *time.Time      `db:"last_request_at"`
	AvgResponseTimeMs       int             `db:"avg_response_time_ms"`
	AvgThinkingIterations   float64         `db:"avg_thinking_iterations"`
}

// UserStatsResponse is the API representation of a user's full statistics.
type UserStatsResponse struct {
	UserID                  int                    `json:"user_id"`
	Username                string                 `json:"username"`
	TotalRequests           int                    `json:"total_requests"`
	TotalSessions           int                    `json:"total_sessions"`
	TotalFilesUploaded      int                    `json:"total_files_uploaded"`
	TotalFilesSizeMB        float64                `json:"total_files_size_mb"`
	TotalPromptTokens       int64                  `json:"total_prompt_tokens"`
	TotalCompletionTokens   int64                  `json:"total_completion_tokens"`
	TotalTokens             int64                  `json:"total_tokens"`
	AvgResponseTimeMs       int                    `json:"avg_response_time_ms"`
	MinResponseTimeMs       int                    `json:"min_response_time_ms"`
	MaxResponseTimeMs       int                    `json:"max_response_time_ms"`
	TotalThinkingIterations int                    `json:"total_thinking_iterations"`
	AvgThinkingIterations   float64                `json:"avg_thinking_iterations"`
	TotalRegenerations      int                    `json:"total_regenerations"`
	MemoryEnabledRequests   int                    `json:"memory_enabled_requests"`
	MemoryUsagePercent      float64                `json:"memory_usage_percent"`
	ProviderUsage           map[string]int         `json:"provider_usage"`
	ModelUsage              map[string]int         `json:"model_usage"`
	FirstRequestAt          *time.Time             `json:"first_request_at"`
	LastRequestAt           *time.Time             `json:"last_request_at"`
	DaysActive              int                    `json:"days_active"`
	RequestsPerDay          float64                `json:"requests_per_day"`
	RecentActivity          []DailyActivitySummary `json:"recent_activity"`
}

// DailyActivitySummary represents a summary of activity for a single day.
type DailyActivitySummary struct {
	Date     string `json:"date"`
	Requests int    `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

// GlobalStatsResponse is the API representation of system-wide statistics.
type GlobalStatsResponse struct {
	TotalUsers           int                    `json:"total_users"`
	TotalRequests        int64                  `json:"total_requests"`
	TotalSessions        int64                  `json:"total_sessions"`
	TotalTokens          int64                  `json:"total_tokens"`
	AvgResponseTimeMs    int                    `json:"avg_response_time_ms"`
	TotalFilesUploaded   int64                  `json:"total_files_uploaded"`
	TotalFilesSizeMB     float64                `json:"total_files_size_mb"`
	TopProviders         map[string]int         `json:"top_providers"`
	TopModels            map[string]int         `json:"top_models"`
	ActiveUsersToday     int                    `json:"active_users_today"`
	ActiveUsersThisWeek  int                    `json:"active_users_this_week"`
	ActiveUsersThisMonth int                    `json:"active_users_this_month"`
	RecentActivity       []DailyActivitySummary `json:"recent_activity"`
}

// ProviderTokenUsageRow represents token usage aggregated by provider.
type ProviderTokenUsageRow struct {
	Provider         string `db:"llm_provider" json:"provider"`
	PromptTokens     int64  `db:"prompt_tokens" json:"prompt_tokens"`
	CompletionTokens int64  `db:"completion_tokens" json:"completion_tokens"`
	TotalTokens      int64  `db:"total_tokens" json:"total_tokens"`
}

// FileChunkWithEmbedding is a DTO used for retrieval, combining a chunk's text with its vector.
type FileChunkWithEmbedding struct {
	ID           int64  `db:"id"`
	ChunkText    string `db:"chunk_text"`
	Embedding    []float64
	EmbeddingRaw []byte `db:"embedding"`
}

// MessageEmbeddingWithText is a DTO used for retrieval, combining a message's text with its vector.
type MessageEmbeddingWithText struct {
	LogID        int64  `db:"log_id"`
	Text         string `db:"final_response"`
	Embedding    []float64
	EmbeddingRaw []byte `db:"embedding"`
}
