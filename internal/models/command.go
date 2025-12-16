// Package models contains data structures for the agent ecosystem
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// CommandStatus represents the current state of a command
type CommandStatus string

const (
	CommandStatusActive     CommandStatus = "active"
	CommandStatusDeprecated CommandStatus = "deprecated"
	CommandStatusDisabled   CommandStatus = "disabled"
)

// Command represents a reusable slash command that can be synced to projects
type Command struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"` // e.g., "review-pr", "fix-tests"
	Version     string    `json:"version" db:"version"`
	Description string    `json:"description" db:"description"`

	// Command content
	Prompt    string     `json:"prompt" db:"prompt"` // The actual command prompt/template
	Arguments []Argument `json:"arguments" db:"arguments"`

	// Metadata and discovery
	Metadata map[string]any `json:"metadata" db:"metadata"`
	Tags     []string       `json:"tags" db:"tags"`
	Category string         `json:"category" db:"category"` // code, git, test, deploy

	// Embeddings for semantic search
	Embedding *pgvector.Vector `json:"-" db:"embedding"`

	// Reputation tracking
	ReputationScore float64 `json:"reputation_score" db:"reputation_score"`
	UsageCount      int     `json:"usage_count" db:"usage_count"`
	FeedbackCount   int     `json:"feedback_count" db:"feedback_count"`
	AvgRating       float64 `json:"avg_rating" db:"avg_rating"`

	// Status
	Status   CommandStatus `json:"status" db:"status"`
	IsSystem bool          `json:"is_system" db:"is_system"`

	// Audit
	CreatedBy *string   `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Argument defines an expected argument for a command
type Argument struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// CommandSummary is a lightweight version for listing
type CommandSummary struct {
	ID              uuid.UUID     `json:"id"`
	Name            string        `json:"name"`
	Version         string        `json:"version"`
	Description     string        `json:"description"`
	Category        string        `json:"category"`
	Tags            []string      `json:"tags"`
	Arguments       []Argument    `json:"arguments"`
	ReputationScore float64       `json:"reputation_score"`
	AvgRating       float64       `json:"avg_rating"`
	UsageCount      int           `json:"usage_count"`
	Status          CommandStatus `json:"status"`
}

// ToSummary converts a Command to CommandSummary
func (c *Command) ToSummary() CommandSummary {
	return CommandSummary{
		ID:              c.ID,
		Name:            c.Name,
		Version:         c.Version,
		Description:     c.Description,
		Category:        c.Category,
		Tags:            c.Tags,
		Arguments:       c.Arguments,
		ReputationScore: c.ReputationScore,
		AvgRating:       c.AvgRating,
		UsageCount:      c.UsageCount,
		Status:          c.Status,
	}
}

// SimilarCommand represents a search result with similarity score
type SimilarCommand struct {
	Command    CommandSummary `json:"command"`
	Similarity float64        `json:"similarity"`
}

// CommandFeedback records feedback for a command
type CommandFeedback struct {
	ID           uuid.UUID `json:"id" db:"id"`
	CommandID    uuid.UUID `json:"command_id" db:"command_id"`
	Rating       int       `json:"rating" db:"rating"`
	TaskSuccess  bool      `json:"task_success" db:"task_success"`
	FeedbackText string    `json:"feedback_text" db:"feedback_text"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
