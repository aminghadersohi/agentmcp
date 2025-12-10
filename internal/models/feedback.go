package models

import (
	"time"

	"github.com/google/uuid"
)

// Feedback represents user feedback on an agent's performance
type Feedback struct {
	ID      uuid.UUID `json:"id" db:"id"`
	AgentID uuid.UUID `json:"agent_id" db:"agent_id"`

	// Who gave feedback
	SessionID  string `json:"session_id" db:"session_id"`
	ClientType string `json:"client_type" db:"client_type"`

	// Feedback data
	Rating       int    `json:"rating" db:"rating"` // 1-5
	TaskSuccess  bool   `json:"task_success" db:"task_success"`
	TaskType     string `json:"task_type" db:"task_type"`
	FeedbackText string `json:"feedback_text,omitempty" db:"feedback_text"`

	// Context
	InteractionDurationMs int `json:"interaction_duration_ms,omitempty" db:"interaction_duration_ms"`
	TokensUsed            int `json:"tokens_used,omitempty" db:"tokens_used"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// FeedbackInput is the input for submitting feedback
type FeedbackInput struct {
	AgentName    string `json:"agent_name"`
	Rating       int    `json:"rating"`
	TaskSuccess  bool   `json:"task_success"`
	TaskType     string `json:"task_type,omitempty"`
	FeedbackText string `json:"feedback_text,omitempty"`
}

// AgentReputation contains detailed reputation information
type AgentReputation struct {
	AgentID         uuid.UUID `json:"agent_id"`
	AgentName       string    `json:"agent_name"`
	ReputationScore float64   `json:"reputation_score"`
	UsageCount      int       `json:"usage_count"`
	FeedbackCount   int       `json:"feedback_count"`
	AvgRating       float64   `json:"avg_rating"`
	SuccessRate     float64   `json:"success_rate"`

	// Breakdown
	RatingDistribution map[int]int    `json:"rating_distribution"` // rating -> count
	TaskTypeBreakdown  map[string]int `json:"task_type_breakdown"` // task_type -> count

	// Recent activity
	RecentFeedback []Feedback `json:"recent_feedback,omitempty"`
}
