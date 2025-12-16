// Package database provides PostgreSQL database operations
package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aminghadersohi/agentmcp/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

// DB wraps the PostgreSQL connection pool
type DB struct {
	pool *pgxpool.Pool
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	MaxConns int
}

// New creates a new database connection
func New(cfg Config) (*DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s pool_max_conns=%d",
		cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password, cfg.MaxConns,
	)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close closes the database connection
func (db *DB) Close() {
	db.pool.Close()
}

// Pool returns the underlying connection pool (for migrations)
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// ============ Agent Operations ============

// CreateAgent inserts a new agent
func (db *DB) CreateAgent(ctx context.Context, agent *models.Agent) error {
	if agent.ID == uuid.Nil {
		agent.ID = uuid.New()
	}
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()

	toolsJSON, _ := json.Marshal(agent.Tools)
	metadataJSON, _ := json.Marshal(agent.Metadata)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO agents (
			id, name, version, description, model, tools, metadata, prompt,
			embedding, skills, reputation_score, status, is_system, is_generated,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14,
			$15, $16, $17
		)
	`,
		agent.ID, agent.Name, agent.Version, agent.Description, agent.Model,
		toolsJSON, metadataJSON, agent.Prompt,
		agent.Embedding, agent.Skills, agent.ReputationScore, agent.Status,
		agent.IsSystem, agent.IsGenerated, agent.CreatedBy, agent.CreatedAt, agent.UpdatedAt,
	)

	return err
}

// GetAgent retrieves an agent by name
func (db *DB) GetAgent(ctx context.Context, name string) (*models.Agent, error) {
	var agent models.Agent
	var toolsJSON, metadataJSON []byte

	err := db.pool.QueryRow(ctx, `
		SELECT id, name, version, description, model, tools, metadata, prompt,
			   embedding, skills, reputation_score, usage_count, feedback_count,
			   avg_rating, status, is_system, is_generated, created_by, created_at, updated_at
		FROM agents WHERE name = $1
	`, name).Scan(
		&agent.ID, &agent.Name, &agent.Version, &agent.Description, &agent.Model,
		&toolsJSON, &metadataJSON, &agent.Prompt,
		&agent.Embedding, &agent.Skills, &agent.ReputationScore, &agent.UsageCount,
		&agent.FeedbackCount, &agent.AvgRating, &agent.Status, &agent.IsSystem,
		&agent.IsGenerated, &agent.CreatedBy, &agent.CreatedAt, &agent.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(toolsJSON, &agent.Tools)
	json.Unmarshal(metadataJSON, &agent.Metadata)

	return &agent, nil
}

// GetAgentByID retrieves an agent by ID
func (db *DB) GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	var agent models.Agent
	var toolsJSON, metadataJSON []byte

	err := db.pool.QueryRow(ctx, `
		SELECT id, name, version, description, model, tools, metadata, prompt,
			   embedding, skills, reputation_score, usage_count, feedback_count,
			   avg_rating, status, is_system, is_generated, created_by, created_at, updated_at
		FROM agents WHERE id = $1
	`, id).Scan(
		&agent.ID, &agent.Name, &agent.Version, &agent.Description, &agent.Model,
		&toolsJSON, &metadataJSON, &agent.Prompt,
		&agent.Embedding, &agent.Skills, &agent.ReputationScore, &agent.UsageCount,
		&agent.FeedbackCount, &agent.AvgRating, &agent.Status, &agent.IsSystem,
		&agent.IsGenerated, &agent.CreatedBy, &agent.CreatedAt, &agent.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(toolsJSON, &agent.Tools)
	json.Unmarshal(metadataJSON, &agent.Metadata)

	return &agent, nil
}

// ListAgents returns all active agents
func (db *DB) ListAgents(ctx context.Context, tags []string) ([]models.AgentSummary, error) {
	query := `
		SELECT id, name, version, description, skills, reputation_score, avg_rating, usage_count, status
		FROM agents
		WHERE status = 'active'
	`
	args := []any{}

	if len(tags) > 0 {
		// Check both skills array and metadata->tags JSON array
		query += ` AND (
			skills && $1
			OR metadata->'tags' ?| $1
		)`
		args = append(args, tags)
	}

	query += " ORDER BY reputation_score DESC, usage_count DESC"

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := []models.AgentSummary{} // Initialize as empty slice, not nil
	for rows.Next() {
		var a models.AgentSummary
		err := rows.Scan(&a.ID, &a.Name, &a.Version, &a.Description, &a.Skills,
			&a.ReputationScore, &a.AvgRating, &a.UsageCount, &a.Status)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}

	return agents, nil
}

// escapeLikePattern escapes SQL LIKE pattern special characters
func escapeLikePattern(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// SearchAgents searches agents by keyword
func (db *DB) SearchAgents(ctx context.Context, query string) ([]models.AgentSummary, error) {
	// Escape SQL wildcards to prevent injection
	escaped := escapeLikePattern(strings.ToLower(query))
	q := "%" + escaped + "%"

	rows, err := db.pool.Query(ctx, `
		SELECT id, name, version, description, skills, reputation_score, avg_rating, usage_count, status
		FROM agents
		WHERE status = 'active'
		  AND (
			LOWER(name) LIKE $1 ESCAPE '\'
			OR LOWER(description) LIKE $1 ESCAPE '\'
			OR $2 = ANY(skills)
			OR EXISTS (
				SELECT 1 FROM jsonb_array_elements_text(metadata->'tags') AS tag
				WHERE LOWER(tag) LIKE $1 ESCAPE '\'
			)
		  )
		ORDER BY reputation_score DESC
	`, q, strings.ToLower(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := []models.AgentSummary{} // Initialize as empty slice, not nil
	for rows.Next() {
		var a models.AgentSummary
		err := rows.Scan(&a.ID, &a.Name, &a.Version, &a.Description, &a.Skills,
			&a.ReputationScore, &a.AvgRating, &a.UsageCount, &a.Status)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}

	return agents, nil
}

// FindSimilarAgents finds agents by embedding similarity
func (db *DB) FindSimilarAgents(ctx context.Context, embedding pgvector.Vector, limit int, threshold float64) ([]models.SimilarAgent, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, name, version, description, skills, reputation_score, avg_rating, usage_count, status,
			   1 - (embedding <=> $1) as similarity
		FROM agents
		WHERE status = 'active' AND embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2
	`, embedding, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []models.SimilarAgent{} // Initialize as empty slice, not nil
	for rows.Next() {
		var a models.AgentSummary
		var similarity float64
		err := rows.Scan(&a.ID, &a.Name, &a.Version, &a.Description, &a.Skills,
			&a.ReputationScore, &a.AvgRating, &a.UsageCount, &a.Status, &similarity)
		if err != nil {
			return nil, err
		}
		if similarity >= threshold {
			results = append(results, models.SimilarAgent{Agent: a, Similarity: similarity})
		}
	}

	return results, nil
}

// UpdateAgentStatus updates an agent's status
func (db *DB) UpdateAgentStatus(ctx context.Context, agentID uuid.UUID, status models.AgentStatus) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE agents SET status = $1, updated_at = NOW() WHERE id = $2
	`, status, agentID)
	return err
}

// UpdateAgentReputation updates reputation metrics
func (db *DB) UpdateAgentReputation(ctx context.Context, agentID uuid.UUID, score float64) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE agents SET reputation_score = $1, updated_at = NOW() WHERE id = $2
	`, score, agentID)
	return err
}

// IncrementUsage increments the usage count
func (db *DB) IncrementUsage(ctx context.Context, agentID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE agents SET usage_count = usage_count + 1, updated_at = NOW() WHERE id = $1
	`, agentID)
	return err
}

// GetTopAgents returns the highest-rated agents
func (db *DB) GetTopAgents(ctx context.Context, limit int, category string) ([]models.AgentSummary, error) {
	query := `
		SELECT id, name, version, description, skills, reputation_score, avg_rating, usage_count, status
		FROM agents
		WHERE status = 'active'
	`
	args := []any{}
	argIndex := 1

	if category != "" {
		// Check both skills array and metadata->tags JSON array
		query += fmt.Sprintf(` AND (
			$%d = ANY(skills)
			OR metadata->'tags' ? $%d
		)`, argIndex, argIndex)
		args = append(args, category)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY reputation_score DESC, avg_rating DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := []models.AgentSummary{} // Initialize as empty slice, not nil
	for rows.Next() {
		var a models.AgentSummary
		err := rows.Scan(&a.ID, &a.Name, &a.Version, &a.Description, &a.Skills,
			&a.ReputationScore, &a.AvgRating, &a.UsageCount, &a.Status)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}

	return agents, nil
}

// ============ Skill Request Cache ============

// HashSkills creates a consistent hash for a set of skills
func HashSkills(skills []string) string {
	// Normalize: lowercase, sort, join
	normalized := make([]string, len(skills))
	for i, s := range skills {
		normalized[i] = strings.ToLower(strings.TrimSpace(s))
	}
	sort.Strings(normalized)

	hash := sha256.Sum256([]byte(strings.Join(normalized, ",")))
	return hex.EncodeToString(hash[:])
}

// GetCachedAgentBySkills checks if we have a cached agent for these skills
func (db *DB) GetCachedAgentBySkills(ctx context.Context, skills []string) (*models.Agent, error) {
	hash := HashSkills(skills)

	var agentID uuid.UUID
	err := db.pool.QueryRow(ctx, `
		SELECT agent_id FROM skill_requests WHERE skills_hash = $1
	`, hash).Scan(&agentID)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Update request count
	db.pool.Exec(ctx, `
		UPDATE skill_requests SET request_count = request_count + 1, last_requested = NOW()
		WHERE skills_hash = $1
	`, hash)

	return db.GetAgentByID(ctx, agentID)
}

// CacheSkillRequest caches a skill->agent mapping
func (db *DB) CacheSkillRequest(ctx context.Context, skills []string, agentID uuid.UUID) error {
	hash := HashSkills(skills)

	// Normalize skills
	normalized := make([]string, len(skills))
	for i, s := range skills {
		normalized[i] = strings.ToLower(strings.TrimSpace(s))
	}

	_, err := db.pool.Exec(ctx, `
		INSERT INTO skill_requests (id, skills_hash, skills, agent_id, request_count, created_at, last_requested)
		VALUES ($1, $2, $3, $4, 1, NOW(), NOW())
		ON CONFLICT (skills_hash) DO UPDATE SET request_count = skill_requests.request_count + 1, last_requested = NOW()
	`, uuid.New(), hash, normalized, agentID)

	return err
}

// ============ Feedback Operations ============

// SubmitFeedback records feedback for an agent
func (db *DB) SubmitFeedback(ctx context.Context, feedback *models.Feedback) error {
	feedback.ID = uuid.New()
	feedback.CreatedAt = time.Now()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO feedback (id, agent_id, session_id, client_type, rating, task_success,
							  task_type, feedback_text, interaction_duration_ms, tokens_used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		feedback.ID, feedback.AgentID, feedback.SessionID, feedback.ClientType,
		feedback.Rating, feedback.TaskSuccess, feedback.TaskType, feedback.FeedbackText,
		feedback.InteractionDurationMs, feedback.TokensUsed, feedback.CreatedAt,
	)
	if err != nil {
		return err
	}

	// Update agent statistics
	_, err = db.pool.Exec(ctx, `
		UPDATE agents SET
			feedback_count = feedback_count + 1,
			avg_rating = (
				SELECT AVG(rating)::float FROM feedback WHERE agent_id = $1
			),
			updated_at = NOW()
		WHERE id = $1
	`, feedback.AgentID)

	return err
}

// GetAgentReputation calculates reputation details
func (db *DB) GetAgentReputation(ctx context.Context, agentID uuid.UUID) (*models.AgentReputation, error) {
	agent, err := db.GetAgentByID(ctx, agentID)
	if err != nil || agent == nil {
		return nil, err
	}

	rep := &models.AgentReputation{
		AgentID:            agent.ID,
		AgentName:          agent.Name,
		ReputationScore:    agent.ReputationScore,
		UsageCount:         agent.UsageCount,
		FeedbackCount:      agent.FeedbackCount,
		AvgRating:          agent.AvgRating,
		RatingDistribution: make(map[int]int),
		TaskTypeBreakdown:  make(map[string]int),
	}

	// Get rating distribution
	rows, err := db.pool.Query(ctx, `
		SELECT rating, COUNT(*) FROM feedback WHERE agent_id = $1 GROUP BY rating
	`, agentID)
	if err == nil {
		for rows.Next() {
			var rating, count int
			rows.Scan(&rating, &count)
			rep.RatingDistribution[rating] = count
		}
		rows.Close()
	}

	// Get success rate
	var successCount int
	db.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM feedback WHERE agent_id = $1 AND task_success = true
	`, agentID).Scan(&successCount)
	if rep.FeedbackCount > 0 {
		rep.SuccessRate = float64(successCount) / float64(rep.FeedbackCount)
	}

	// Get task type breakdown
	rows, err = db.pool.Query(ctx, `
		SELECT task_type, COUNT(*) FROM feedback WHERE agent_id = $1 AND task_type != '' GROUP BY task_type
	`, agentID)
	if err == nil {
		for rows.Next() {
			var taskType string
			var count int
			rows.Scan(&taskType, &count)
			rep.TaskTypeBreakdown[taskType] = count
		}
		rows.Close()
	}

	return rep, nil
}

// ============ Governance Operations ============

// CreateReport creates a new report
func (db *DB) CreateReport(ctx context.Context, report *models.Report) error {
	report.ID = uuid.New()
	report.Status = models.ReportStatusPending
	report.CreatedAt = time.Now()

	evidenceJSON, _ := json.Marshal(report.Evidence)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO reports (id, agent_id, reported_by, report_type, severity, description, evidence, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		report.ID, report.AgentID, report.ReportedBy, report.ReportType,
		report.Severity, report.Description, evidenceJSON, report.Status, report.CreatedAt,
	)
	return err
}

// GetPendingReports returns reports awaiting review
func (db *DB) GetPendingReports(ctx context.Context) ([]models.Report, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT r.id, r.agent_id, a.name, r.reported_by, r.report_type, r.severity,
			   r.description, r.evidence, r.status, r.reviewed_by, r.resolution,
			   r.resolution_note, r.created_at, r.resolved_at
		FROM reports r
		JOIN agents a ON r.agent_id = a.id
		WHERE r.status IN ('pending', 'reviewing')
		ORDER BY
			CASE r.severity
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 3
				ELSE 4
			END,
			r.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := []models.Report{} // Initialize as empty slice, not nil
	for rows.Next() {
		var r models.Report
		var evidenceJSON []byte
		err := rows.Scan(&r.ID, &r.AgentID, &r.AgentName, &r.ReportedBy, &r.ReportType,
			&r.Severity, &r.Description, &evidenceJSON, &r.Status, &r.ReviewedBy,
			&r.Resolution, &r.ResolutionNote, &r.CreatedAt, &r.ResolvedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(evidenceJSON, &r.Evidence)
		reports = append(reports, r)
	}

	return reports, nil
}

// UpdateReportStatus updates a report's status
func (db *DB) UpdateReportStatus(ctx context.Context, reportID uuid.UUID, status models.ReportStatus, reviewedBy string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE reports SET status = $1, reviewed_by = $2 WHERE id = $3
	`, status, reviewedBy, reportID)
	return err
}

// ResolveReport resolves a report
func (db *DB) ResolveReport(ctx context.Context, reportID uuid.UUID, resolution models.Resolution, note string, resolvedBy string) error {
	now := time.Now()
	_, err := db.pool.Exec(ctx, `
		UPDATE reports SET status = 'resolved', resolution = $1, resolution_note = $2,
						   reviewed_by = $3, resolved_at = $4
		WHERE id = $5
	`, resolution, note, resolvedBy, now, reportID)
	return err
}

// RecordGovernanceAction records an action taken on an agent
func (db *DB) RecordGovernanceAction(ctx context.Context, action *models.GovernanceAction) error {
	action.ID = uuid.New()
	action.CreatedAt = time.Now()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO governance_actions (id, agent_id, report_id, action_type, action_by, reason,
										previous_status, previous_reputation, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		action.ID, action.AgentID, action.ReportID, action.ActionType, action.ActionBy,
		action.Reason, action.PreviousStatus, action.PreviousReputation, action.CreatedAt,
	)
	return err
}

// GetGovernanceStats returns governance statistics
func (db *DB) GetGovernanceStats(ctx context.Context) (*models.GovernanceStats, error) {
	stats := &models.GovernanceStats{}

	db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'pending'`).Scan(&stats.PendingReports)
	db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM reports WHERE status = 'reviewing'`).Scan(&stats.ReviewingReports)
	db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE status = 'quarantined'`).Scan(&stats.QuarantinedAgents)
	db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE status = 'banned'`).Scan(&stats.BannedAgents)
	db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM governance_actions WHERE created_at > NOW() - INTERVAL '1 day'`).Scan(&stats.ActionsToday)
	db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM governance_actions WHERE created_at > NOW() - INTERVAL '7 days'`).Scan(&stats.ActionsThisWeek)

	return stats, nil
}

// CountPendingReportsForAgent counts pending reports for an agent
func (db *DB) CountPendingReportsForAgent(ctx context.Context, agentID uuid.UUID) (int, error) {
	var count int
	err := db.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM reports WHERE agent_id = $1 AND status = 'pending'
	`, agentID).Scan(&count)
	return count, err
}
