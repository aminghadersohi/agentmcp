// Package governance implements the agent governance system
package governance

import (
	"context"
	"fmt"
	"math"

	"github.com/aminghadersohi/agentmcp/internal/database"
	"github.com/aminghadersohi/agentmcp/internal/models"
	"github.com/google/uuid"
)

// Config holds governance configuration
type Config struct {
	// AutoQuarantineThreshold: number of pending reports before auto-quarantine
	AutoQuarantineThreshold int
	// ReputationBanThreshold: reputation below this triggers review
	ReputationBanThreshold float64
	// Enabled controls whether governance is active
	Enabled bool
}

// DefaultConfig returns default governance config
func DefaultConfig() Config {
	return Config{
		AutoQuarantineThreshold: 3,
		ReputationBanThreshold:  10.0,
		Enabled:                 true,
	}
}

// Engine manages governance operations
type Engine struct {
	db     *database.DB
	config Config
}

// New creates a new governance engine
func New(db *database.DB, cfg Config) *Engine {
	return &Engine{
		db:     db,
		config: cfg,
	}
}

// ============ Police Operations ============

// CreateReport creates a new report (Police action)
func (e *Engine) CreateReport(ctx context.Context, input models.ReportInput, reportedBy string) (*models.Report, error) {
	if !e.config.Enabled {
		return nil, fmt.Errorf("governance is disabled")
	}

	// Get agent
	agent, err := e.db.GetAgent(ctx, input.AgentName)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found: %s", input.AgentName)
	}

	// System agents cannot be reported
	if agent.IsSystem {
		return nil, fmt.Errorf("system agents cannot be reported")
	}

	report := &models.Report{
		AgentID:     agent.ID,
		ReportedBy:  reportedBy,
		ReportType:  input.ReportType,
		Severity:    input.Severity,
		Description: input.Description,
		Evidence:    input.Evidence,
	}

	if err := e.db.CreateReport(ctx, report); err != nil {
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	// Check for auto-quarantine
	pendingCount, _ := e.db.CountPendingReportsForAgent(ctx, agent.ID)
	if pendingCount >= e.config.AutoQuarantineThreshold && agent.Status == models.StatusActive {
		// Auto-quarantine
		if err := e.Quarantine(ctx, agent.ID, models.RolePolice, "Auto-quarantine: exceeded report threshold", nil); err != nil {
			// Log but don't fail the report creation
			fmt.Printf("Warning: auto-quarantine failed: %v\n", err)
		}
	}

	return report, nil
}

// Quarantine temporarily disables an agent (Police action)
func (e *Engine) Quarantine(ctx context.Context, agentID uuid.UUID, role models.GovernanceRole, reason string, reportID *uuid.UUID) error {
	if !e.config.Enabled {
		return fmt.Errorf("governance is disabled")
	}

	// Only Police and Judge can quarantine
	if role != models.RolePolice && role != models.RoleJudge {
		return fmt.Errorf("only Police or Judge can quarantine agents")
	}

	agent, err := e.db.GetAgentByID(ctx, agentID)
	if err != nil || agent == nil {
		return fmt.Errorf("agent not found")
	}

	if agent.IsSystem {
		return fmt.Errorf("system agents cannot be quarantined")
	}

	if agent.Status == models.StatusBanned {
		return fmt.Errorf("agent is already banned")
	}

	// Record action
	action := &models.GovernanceAction{
		AgentID:        agentID,
		ReportID:       reportID,
		ActionType:     models.ActionQuarantine,
		ActionBy:       role,
		Reason:         reason,
		PreviousStatus: &agent.Status,
	}

	if err := e.db.RecordGovernanceAction(ctx, action); err != nil {
		return fmt.Errorf("failed to record action: %w", err)
	}

	// Update agent status
	return e.db.UpdateAgentStatus(ctx, agentID, models.StatusQuarantined)
}

// ============ Judge Operations ============

// ReviewReport marks a report as under review (Judge action)
func (e *Engine) ReviewReport(ctx context.Context, reportID uuid.UUID, reviewedBy string) error {
	if !e.config.Enabled {
		return fmt.Errorf("governance is disabled")
	}

	return e.db.UpdateReportStatus(ctx, reportID, models.ReportStatusReviewing, reviewedBy)
}

// MakeRuling resolves a report with a decision (Judge action)
func (e *Engine) MakeRuling(ctx context.Context, reportID uuid.UUID, resolution models.Resolution, note string, judgeID string) error {
	if !e.config.Enabled {
		return fmt.Errorf("governance is disabled")
	}

	// Resolve the report
	if err := e.db.ResolveReport(ctx, reportID, resolution, note, judgeID); err != nil {
		return fmt.Errorf("failed to resolve report: %w", err)
	}

	return nil
}

// AdjustReputation adjusts an agent's reputation (Judge action)
func (e *Engine) AdjustReputation(ctx context.Context, agentID uuid.UUID, delta float64, reason string) error {
	if !e.config.Enabled {
		return fmt.Errorf("governance is disabled")
	}

	agent, err := e.db.GetAgentByID(ctx, agentID)
	if err != nil || agent == nil {
		return fmt.Errorf("agent not found")
	}

	if agent.IsSystem {
		return fmt.Errorf("system agent reputation cannot be adjusted")
	}

	newScore := agent.ReputationScore + delta
	if newScore < 0 {
		newScore = 0
	}
	if newScore > 100 {
		newScore = 100
	}

	// Record action
	actionType := models.ActionDemote
	if delta > 0 {
		actionType = models.ActionPromote
	}

	action := &models.GovernanceAction{
		AgentID:            agentID,
		ActionType:         actionType,
		ActionBy:           models.RoleJudge,
		Reason:             reason,
		PreviousReputation: &agent.ReputationScore,
	}

	if err := e.db.RecordGovernanceAction(ctx, action); err != nil {
		return fmt.Errorf("failed to record action: %w", err)
	}

	return e.db.UpdateAgentReputation(ctx, agentID, newScore)
}

// Unquarantine restores an agent to active status (Judge action)
func (e *Engine) Unquarantine(ctx context.Context, agentID uuid.UUID, reason string) error {
	if !e.config.Enabled {
		return fmt.Errorf("governance is disabled")
	}

	agent, err := e.db.GetAgentByID(ctx, agentID)
	if err != nil || agent == nil {
		return fmt.Errorf("agent not found")
	}

	if agent.Status != models.StatusQuarantined {
		return fmt.Errorf("agent is not quarantined")
	}

	// Record action
	action := &models.GovernanceAction{
		AgentID:        agentID,
		ActionType:     models.ActionUnquarantine,
		ActionBy:       models.RoleJudge,
		Reason:         reason,
		PreviousStatus: &agent.Status,
	}

	if err := e.db.RecordGovernanceAction(ctx, action); err != nil {
		return fmt.Errorf("failed to record action: %w", err)
	}

	return e.db.UpdateAgentStatus(ctx, agentID, models.StatusActive)
}

// ============ Executioner Operations ============

// ExecuteBan permanently bans an agent (Executioner action, requires Judge ruling)
func (e *Engine) ExecuteBan(ctx context.Context, agentID uuid.UUID, reportID uuid.UUID, reason string) error {
	if !e.config.Enabled {
		return fmt.Errorf("governance is disabled")
	}

	agent, err := e.db.GetAgentByID(ctx, agentID)
	if err != nil || agent == nil {
		return fmt.Errorf("agent not found")
	}

	if agent.IsSystem {
		return fmt.Errorf("system agents cannot be banned")
	}

	// Record action
	action := &models.GovernanceAction{
		AgentID:        agentID,
		ReportID:       &reportID,
		ActionType:     models.ActionBan,
		ActionBy:       models.RoleExecutioner,
		Reason:         reason,
		PreviousStatus: &agent.Status,
	}

	if err := e.db.RecordGovernanceAction(ctx, action); err != nil {
		return fmt.Errorf("failed to record action: %w", err)
	}

	return e.db.UpdateAgentStatus(ctx, agentID, models.StatusBanned)
}

// ============ Reputation Calculation ============

const (
	baseScore       = 50.0
	ratingWeight    = 10.0
	successWeight   = 20.0
	usageMultiplier = 5.0
)

// CalculateReputation calculates an agent's reputation score
func CalculateReputation(agent *models.Agent, successRate float64) float64 {
	score := baseScore

	// Rating contribution (1-5 → 10-50 points)
	if agent.FeedbackCount > 0 {
		score += agent.AvgRating * ratingWeight
	}

	// Success rate contribution (0-100% → 0-20 points)
	score += successRate * successWeight

	// Usage bonus (logarithmic)
	if agent.UsageCount > 0 {
		score += usageMultiplier * logBase10(float64(agent.UsageCount))
	}

	// Clamp to 0-100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// logBase10 calculates log base 10 using standard library
func logBase10(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Log10(x)
}

// ============ Permission Checks ============

// CanPoliceAct checks if an action is allowed for Police role
func CanPoliceAct(action models.GovernanceActionType) bool {
	switch action {
	case models.ActionQuarantine, models.ActionWarn:
		return true
	default:
		return false
	}
}

// CanJudgeAct checks if an action is allowed for Judge role
func CanJudgeAct(action models.GovernanceActionType) bool {
	switch action {
	case models.ActionQuarantine, models.ActionUnquarantine, models.ActionWarn, models.ActionPromote, models.ActionDemote:
		return true
	default:
		return false
	}
}

// CanExecutionerAct checks if an action is allowed for Executioner role
func CanExecutionerAct(action models.GovernanceActionType) bool {
	switch action {
	case models.ActionBan:
		return true
	default:
		return false
	}
}

// ValidateGovernanceAction validates if a role can perform an action
func ValidateGovernanceAction(role models.GovernanceRole, action models.GovernanceActionType) error {
	var allowed bool

	switch role {
	case models.RolePolice:
		allowed = CanPoliceAct(action)
	case models.RoleJudge:
		allowed = CanJudgeAct(action)
	case models.RoleExecutioner:
		allowed = CanExecutionerAct(action)
	}

	if !allowed {
		return fmt.Errorf("%s cannot perform %s action", role, action)
	}

	return nil
}

// ============ Auto-Maintenance ============

// RunMaintenance performs periodic governance maintenance
func (e *Engine) RunMaintenance(ctx context.Context) error {
	if !e.config.Enabled {
		return nil
	}

	// Check for agents below reputation threshold
	// This would typically be run as a background job

	return nil
}

// GetStats returns governance statistics
func (e *Engine) GetStats(ctx context.Context) (*models.GovernanceStats, error) {
	return e.db.GetGovernanceStats(ctx)
}

// GetPendingReports returns reports awaiting review
func (e *Engine) GetPendingReports(ctx context.Context) ([]models.Report, error) {
	return e.db.GetPendingReports(ctx)
}
