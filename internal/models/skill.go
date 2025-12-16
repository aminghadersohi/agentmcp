// Package models contains data structures for the agent ecosystem
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

// SkillStatus represents the current state of a skill
type SkillStatus string

const (
	SkillStatusActive     SkillStatus = "active"
	SkillStatusDeprecated SkillStatus = "deprecated"
	SkillStatusDisabled   SkillStatus = "disabled"
)

// Skill represents packaged knowledge/documentation for a specific tool
// Examples: kubectl, docker-cli, curl, jenkins, datadog, terraform
type Skill struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Version     string         `json:"version" db:"version"`
	Description string         `json:"description" db:"description"`
	Category    string         `json:"category" db:"category"` // devops, api, database, cloud, cli

	// The actual skill content (knowledge, documentation, patterns)
	Content  string   `json:"content" db:"content"`
	Examples []Example `json:"examples" db:"examples"`

	// Metadata and discovery
	Metadata map[string]any `json:"metadata" db:"metadata"`
	Tags     []string       `json:"tags" db:"tags"`

	// Embeddings for semantic search
	Embedding *pgvector.Vector `json:"-" db:"embedding"`

	// Reputation tracking
	ReputationScore float64 `json:"reputation_score" db:"reputation_score"`
	UsageCount      int     `json:"usage_count" db:"usage_count"`
	FeedbackCount   int     `json:"feedback_count" db:"feedback_count"`
	AvgRating       float64 `json:"avg_rating" db:"avg_rating"`

	// Status
	Status   SkillStatus `json:"status" db:"status"`
	IsSystem bool        `json:"is_system" db:"is_system"`

	// Audit
	CreatedBy *string   `json:"created_by,omitempty" db:"created_by"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Example represents a code example with description
type Example struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Code        string `json:"code"`
	Language    string `json:"language,omitempty"`
}

// SkillSummary is a lightweight version for listing
type SkillSummary struct {
	ID              uuid.UUID   `json:"id"`
	Name            string      `json:"name"`
	Version         string      `json:"version"`
	Description     string      `json:"description"`
	Category        string      `json:"category"`
	Tags            []string    `json:"tags"`
	ReputationScore float64     `json:"reputation_score"`
	AvgRating       float64     `json:"avg_rating"`
	UsageCount      int         `json:"usage_count"`
	Status          SkillStatus `json:"status"`
}

// ToSummary converts a Skill to SkillSummary
func (s *Skill) ToSummary() SkillSummary {
	return SkillSummary{
		ID:              s.ID,
		Name:            s.Name,
		Version:         s.Version,
		Description:     s.Description,
		Category:        s.Category,
		Tags:            s.Tags,
		ReputationScore: s.ReputationScore,
		AvgRating:       s.AvgRating,
		UsageCount:      s.UsageCount,
		Status:          s.Status,
	}
}

// SimilarSkill represents a search result with similarity score
type SimilarSkill struct {
	Skill      SkillSummary `json:"skill"`
	Similarity float64      `json:"similarity"`
}

// SkillFeedback records feedback for a skill
type SkillFeedback struct {
	ID           uuid.UUID `json:"id" db:"id"`
	SkillID      uuid.UUID `json:"skill_id" db:"skill_id"`
	Rating       int       `json:"rating" db:"rating"`
	TaskSuccess  bool      `json:"task_success" db:"task_success"`
	FeedbackText string    `json:"feedback_text" db:"feedback_text"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}
