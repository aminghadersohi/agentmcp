// Package models contains data structures for the agent ecosystem
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// AgentStatus represents the current state of an agent
type AgentStatus string

const (
	StatusActive      AgentStatus = "active"
	StatusQuarantined AgentStatus = "quarantined"
	StatusBanned      AgentStatus = "banned"
)

// Agent represents an AI agent definition with reputation tracking
type Agent struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Version     string         `json:"version" db:"version"`
	Description string         `json:"description" db:"description"`
	Model       string         `json:"model" db:"model"`
	Tools       []string       `json:"tools" db:"tools"`
	Metadata    map[string]any `json:"metadata" db:"metadata"`
	Prompt      string         `json:"prompt" db:"prompt"`

	// Embeddings for semantic search
	Embedding *pgvector.Vector `json:"-" db:"embedding"`
	Skills    []string         `json:"skills" db:"skills"`

	// Reputation
	ReputationScore float64 `json:"reputation_score" db:"reputation_score"`
	UsageCount      int     `json:"usage_count" db:"usage_count"`
	FeedbackCount   int     `json:"feedback_count" db:"feedback_count"`
	AvgRating       float64 `json:"avg_rating" db:"avg_rating"`

	// Status
	Status      AgentStatus `json:"status" db:"status"`
	IsSystem    bool        `json:"is_system" db:"is_system"`
	IsGenerated bool        `json:"is_generated" db:"is_generated"`

	// Audit
	CreatedBy *string   `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AgentSummary is a lightweight version for listing
type AgentSummary struct {
	ID              uuid.UUID   `json:"id"`
	Name            string      `json:"name"`
	Version         string      `json:"version"`
	Description     string      `json:"description"`
	Skills          []string    `json:"skills"`
	ReputationScore float64     `json:"reputation_score"`
	AvgRating       float64     `json:"avg_rating"`
	UsageCount      int         `json:"usage_count"`
	Status          AgentStatus `json:"status"`
}

// ToSummary converts an Agent to AgentSummary
func (a *Agent) ToSummary() AgentSummary {
	return AgentSummary{
		ID:              a.ID,
		Name:            a.Name,
		Version:         a.Version,
		Description:     a.Description,
		Skills:          a.Skills,
		ReputationScore: a.ReputationScore,
		AvgRating:       a.AvgRating,
		UsageCount:      a.UsageCount,
		Status:          a.Status,
	}
}

// SimilarAgent represents a search result with similarity score
type SimilarAgent struct {
	Agent      AgentSummary `json:"agent"`
	Similarity float64      `json:"similarity"`
}
