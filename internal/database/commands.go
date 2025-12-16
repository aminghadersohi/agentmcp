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

// ============ Command Operations ============

// CreateCommand inserts a new command
func (db *DB) CreateCommand(ctx context.Context, cmd *models.Command) error {
	if cmd.ID == uuid.Nil {
		cmd.ID = uuid.New()
	}
	cmd.CreatedAt = time.Now()
	cmd.UpdatedAt = time.Now()

	argumentsJSON, _ := json.Marshal(cmd.Arguments)
	metadataJSON, _ := json.Marshal(cmd.Metadata)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO commands (
			id, name, version, description, prompt, arguments, metadata,
			tags, category, embedding, reputation_score, status, is_system,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13,
			$14, $15, $16
		)
	`,
		cmd.ID, cmd.Name, cmd.Version, cmd.Description, cmd.Prompt,
		argumentsJSON, metadataJSON, cmd.Tags, cmd.Category, cmd.Embedding,
		cmd.ReputationScore, cmd.Status, cmd.IsSystem,
		cmd.CreatedBy, cmd.CreatedAt, cmd.UpdatedAt,
	)

	return err
}

// GetCommand retrieves a command by name
func (db *DB) GetCommand(ctx context.Context, name string) (*models.Command, error) {
	var cmd models.Command
	var argumentsJSON, metadataJSON []byte

	err := db.pool.QueryRow(ctx, `
		SELECT id, name, version, description, prompt, arguments, metadata,
			   tags, category, embedding, reputation_score, usage_count, feedback_count,
			   avg_rating, status, is_system, created_by, created_at, updated_at
		FROM commands WHERE name = $1
	`, name).Scan(
		&cmd.ID, &cmd.Name, &cmd.Version, &cmd.Description, &cmd.Prompt,
		&argumentsJSON, &metadataJSON, &cmd.Tags, &cmd.Category, &cmd.Embedding,
		&cmd.ReputationScore, &cmd.UsageCount, &cmd.FeedbackCount,
		&cmd.AvgRating, &cmd.Status, &cmd.IsSystem,
		&cmd.CreatedBy, &cmd.CreatedAt, &cmd.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(argumentsJSON, &cmd.Arguments)
	json.Unmarshal(metadataJSON, &cmd.Metadata)

	return &cmd, nil
}

// GetCommandByID retrieves a command by ID
func (db *DB) GetCommandByID(ctx context.Context, id uuid.UUID) (*models.Command, error) {
	var cmd models.Command
	var argumentsJSON, metadataJSON []byte

	err := db.pool.QueryRow(ctx, `
		SELECT id, name, version, description, prompt, arguments, metadata,
			   tags, category, embedding, reputation_score, usage_count, feedback_count,
			   avg_rating, status, is_system, created_by, created_at, updated_at
		FROM commands WHERE id = $1
	`, id).Scan(
		&cmd.ID, &cmd.Name, &cmd.Version, &cmd.Description, &cmd.Prompt,
		&argumentsJSON, &metadataJSON, &cmd.Tags, &cmd.Category, &cmd.Embedding,
		&cmd.ReputationScore, &cmd.UsageCount, &cmd.FeedbackCount,
		&cmd.AvgRating, &cmd.Status, &cmd.IsSystem,
		&cmd.CreatedBy, &cmd.CreatedAt, &cmd.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(argumentsJSON, &cmd.Arguments)
	json.Unmarshal(metadataJSON, &cmd.Metadata)

	return &cmd, nil
}

// ListCommands returns all active commands
func (db *DB) ListCommands(ctx context.Context, category string, tags []string) ([]models.CommandSummary, error) {
	query := `
		SELECT id, name, version, description, category, tags, arguments, reputation_score, avg_rating, usage_count, status
		FROM commands
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

	commands := []models.CommandSummary{}
	for rows.Next() {
		var c models.CommandSummary
		var argumentsJSON []byte
		err := rows.Scan(&c.ID, &c.Name, &c.Version, &c.Description, &c.Category,
			&c.Tags, &argumentsJSON, &c.ReputationScore, &c.AvgRating, &c.UsageCount, &c.Status)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(argumentsJSON, &c.Arguments)
		commands = append(commands, c)
	}

	return commands, nil
}

// SearchCommands searches commands by keyword
func (db *DB) SearchCommands(ctx context.Context, query string) ([]models.CommandSummary, error) {
	escaped := escapeLikePattern(strings.ToLower(query))
	q := "%" + escaped + "%"

	rows, err := db.pool.Query(ctx, `
		SELECT id, name, version, description, category, tags, arguments, reputation_score, avg_rating, usage_count, status
		FROM commands
		WHERE status = 'active'
		  AND (
			LOWER(name) LIKE $1 ESCAPE '\'
			OR LOWER(description) LIKE $1 ESCAPE '\'
			OR LOWER(prompt) LIKE $1 ESCAPE '\'
			OR $2 = ANY(tags)
		  )
		ORDER BY reputation_score DESC
	`, q, strings.ToLower(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commands := []models.CommandSummary{}
	for rows.Next() {
		var c models.CommandSummary
		var argumentsJSON []byte
		err := rows.Scan(&c.ID, &c.Name, &c.Version, &c.Description, &c.Category,
			&c.Tags, &argumentsJSON, &c.ReputationScore, &c.AvgRating, &c.UsageCount, &c.Status)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(argumentsJSON, &c.Arguments)
		commands = append(commands, c)
	}

	return commands, nil
}

// FindSimilarCommands finds commands by embedding similarity
func (db *DB) FindSimilarCommands(ctx context.Context, embedding pgvector.Vector, limit int, threshold float64) ([]models.SimilarCommand, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, name, version, description, category, tags, arguments, reputation_score, avg_rating, usage_count, status,
			   1 - (embedding <=> $1) as similarity
		FROM commands
		WHERE status = 'active' AND embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2
	`, embedding, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []models.SimilarCommand{}
	for rows.Next() {
		var c models.CommandSummary
		var argumentsJSON []byte
		var similarity float64
		err := rows.Scan(&c.ID, &c.Name, &c.Version, &c.Description, &c.Category,
			&c.Tags, &argumentsJSON, &c.ReputationScore, &c.AvgRating, &c.UsageCount, &c.Status, &similarity)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(argumentsJSON, &c.Arguments)
		if similarity >= threshold {
			results = append(results, models.SimilarCommand{Command: c, Similarity: similarity})
		}
	}

	return results, nil
}

// IncrementCommandUsage increments the usage count
func (db *DB) IncrementCommandUsage(ctx context.Context, commandID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE commands SET usage_count = usage_count + 1, updated_at = NOW() WHERE id = $1
	`, commandID)
	return err
}

// SubmitCommandFeedback records feedback for a command
func (db *DB) SubmitCommandFeedback(ctx context.Context, feedback *models.CommandFeedback) error {
	feedback.ID = uuid.New()
	feedback.CreatedAt = time.Now()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO command_feedback (id, command_id, rating, task_success, feedback_text, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, feedback.ID, feedback.CommandID, feedback.Rating, feedback.TaskSuccess, feedback.FeedbackText, feedback.CreatedAt)
	if err != nil {
		return err
	}

	// Update command statistics
	_, err = db.pool.Exec(ctx, `
		UPDATE commands SET
			feedback_count = feedback_count + 1,
			avg_rating = (SELECT AVG(rating)::float FROM command_feedback WHERE command_id = $1),
			updated_at = NOW()
		WHERE id = $1
	`, feedback.CommandID)

	return err
}
