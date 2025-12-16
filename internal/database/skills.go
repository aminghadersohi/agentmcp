// Package database provides PostgreSQL database operations
package database

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/aminghadersohi/agentmcp/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

// ============ Skill Operations ============

// CreateSkill inserts a new skill
func (db *DB) CreateSkill(ctx context.Context, skill *models.Skill) error {
	if skill.ID == uuid.Nil {
		skill.ID = uuid.New()
	}
	skill.CreatedAt = time.Now()
	skill.UpdatedAt = time.Now()

	examplesJSON, _ := json.Marshal(skill.Examples)
	metadataJSON, _ := json.Marshal(skill.Metadata)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO skills (
			id, name, version, description, category, content, examples,
			metadata, tags, embedding, reputation_score, status, is_system,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16
		)
	`,
		skill.ID, skill.Name, skill.Version, skill.Description, skill.Category,
		skill.Content, examplesJSON, metadataJSON, skill.Tags, skill.Embedding,
		skill.ReputationScore, skill.Status, skill.IsSystem,
		skill.CreatedBy, skill.CreatedAt, skill.UpdatedAt,
	)

	return err
}

// GetSkill retrieves a skill by name
func (db *DB) GetSkill(ctx context.Context, name string) (*models.Skill, error) {
	var skill models.Skill
	var examplesJSON, metadataJSON []byte

	err := db.pool.QueryRow(ctx, `
		SELECT id, name, version, description, category, content, examples,
			   metadata, tags, embedding, reputation_score, usage_count, feedback_count,
			   avg_rating, status, is_system, created_by, created_at, updated_at
		FROM skills WHERE name = $1
	`, name).Scan(
		&skill.ID, &skill.Name, &skill.Version, &skill.Description, &skill.Category,
		&skill.Content, &examplesJSON, &metadataJSON, &skill.Tags, &skill.Embedding,
		&skill.ReputationScore, &skill.UsageCount, &skill.FeedbackCount,
		&skill.AvgRating, &skill.Status, &skill.IsSystem,
		&skill.CreatedBy, &skill.CreatedAt, &skill.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(examplesJSON, &skill.Examples)
	json.Unmarshal(metadataJSON, &skill.Metadata)

	return &skill, nil
}

// GetSkillByID retrieves a skill by ID
func (db *DB) GetSkillByID(ctx context.Context, id uuid.UUID) (*models.Skill, error) {
	var skill models.Skill
	var examplesJSON, metadataJSON []byte

	err := db.pool.QueryRow(ctx, `
		SELECT id, name, version, description, category, content, examples,
			   metadata, tags, embedding, reputation_score, usage_count, feedback_count,
			   avg_rating, status, is_system, created_by, created_at, updated_at
		FROM skills WHERE id = $1
	`, id).Scan(
		&skill.ID, &skill.Name, &skill.Version, &skill.Description, &skill.Category,
		&skill.Content, &examplesJSON, &metadataJSON, &skill.Tags, &skill.Embedding,
		&skill.ReputationScore, &skill.UsageCount, &skill.FeedbackCount,
		&skill.AvgRating, &skill.Status, &skill.IsSystem,
		&skill.CreatedBy, &skill.CreatedAt, &skill.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(examplesJSON, &skill.Examples)
	json.Unmarshal(metadataJSON, &skill.Metadata)

	return &skill, nil
}

// ListSkills returns all active skills
func (db *DB) ListSkills(ctx context.Context, category string, tags []string) ([]models.SkillSummary, error) {
	query := `
		SELECT id, name, version, description, category, tags, reputation_score, avg_rating, usage_count, status
		FROM skills
		WHERE status = 'active'
	`
	args := []any{}
	argIndex := 1

	if category != "" {
		query += ` AND category = $` + string(rune('0'+argIndex))
		args = append(args, category)
		argIndex++
	}

	if len(tags) > 0 {
		query += ` AND tags && $` + string(rune('0'+argIndex))
		args = append(args, tags)
		argIndex++
	}

	query += " ORDER BY reputation_score DESC, usage_count DESC"

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	skills := []models.SkillSummary{}
	for rows.Next() {
		var s models.SkillSummary
		err := rows.Scan(&s.ID, &s.Name, &s.Version, &s.Description, &s.Category,
			&s.Tags, &s.ReputationScore, &s.AvgRating, &s.UsageCount, &s.Status)
		if err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}

	return skills, nil
}

// SearchSkills searches skills by keyword
func (db *DB) SearchSkills(ctx context.Context, query string) ([]models.SkillSummary, error) {
	escaped := escapeLikePattern(strings.ToLower(query))
	q := "%" + escaped + "%"

	rows, err := db.pool.Query(ctx, `
		SELECT id, name, version, description, category, tags, reputation_score, avg_rating, usage_count, status
		FROM skills
		WHERE status = 'active'
		  AND (
			LOWER(name) LIKE $1 ESCAPE '\'
			OR LOWER(description) LIKE $1 ESCAPE '\'
			OR LOWER(content) LIKE $1 ESCAPE '\'
			OR $2 = ANY(tags)
		  )
		ORDER BY reputation_score DESC
	`, q, strings.ToLower(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	skills := []models.SkillSummary{}
	for rows.Next() {
		var s models.SkillSummary
		err := rows.Scan(&s.ID, &s.Name, &s.Version, &s.Description, &s.Category,
			&s.Tags, &s.ReputationScore, &s.AvgRating, &s.UsageCount, &s.Status)
		if err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}

	return skills, nil
}

// FindSimilarSkills finds skills by embedding similarity
func (db *DB) FindSimilarSkills(ctx context.Context, embedding pgvector.Vector, limit int, threshold float64) ([]models.SimilarSkill, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, name, version, description, category, tags, reputation_score, avg_rating, usage_count, status,
			   1 - (embedding <=> $1) as similarity
		FROM skills
		WHERE status = 'active' AND embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2
	`, embedding, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []models.SimilarSkill{}
	for rows.Next() {
		var s models.SkillSummary
		var similarity float64
		err := rows.Scan(&s.ID, &s.Name, &s.Version, &s.Description, &s.Category,
			&s.Tags, &s.ReputationScore, &s.AvgRating, &s.UsageCount, &s.Status, &similarity)
		if err != nil {
			return nil, err
		}
		if similarity >= threshold {
			results = append(results, models.SimilarSkill{Skill: s, Similarity: similarity})
		}
	}

	return results, nil
}

// IncrementSkillUsage increments the usage count
func (db *DB) IncrementSkillUsage(ctx context.Context, skillID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE skills SET usage_count = usage_count + 1, updated_at = NOW() WHERE id = $1
	`, skillID)
	return err
}

// SubmitSkillFeedback records feedback for a skill
func (db *DB) SubmitSkillFeedback(ctx context.Context, feedback *models.SkillFeedback) error {
	feedback.ID = uuid.New()
	feedback.CreatedAt = time.Now()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO skill_feedback (id, skill_id, rating, task_success, feedback_text, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, feedback.ID, feedback.SkillID, feedback.Rating, feedback.TaskSuccess, feedback.FeedbackText, feedback.CreatedAt)
	if err != nil {
		return err
	}

	// Update skill statistics
	_, err = db.pool.Exec(ctx, `
		UPDATE skills SET
			feedback_count = feedback_count + 1,
			avg_rating = (SELECT AVG(rating)::float FROM skill_feedback WHERE skill_id = $1),
			updated_at = NOW()
		WHERE id = $1
	`, feedback.SkillID)

	return err
}
