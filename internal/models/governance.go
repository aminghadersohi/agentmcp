package models

import (
	"time"

	"github.com/google/uuid"
)

// ReportType categorizes the type of report
type ReportType string

const (
	ReportTypeEthics      ReportType = "ethics"
	ReportTypeHarmful     ReportType = "harmful"
	ReportTypeIneffective ReportType = "ineffective"
	ReportTypeSpam        ReportType = "spam"
	ReportTypeOther       ReportType = "other"
)

// Severity indicates how serious the report is
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// ReportStatus tracks the state of a report
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusReviewing ReportStatus = "reviewing"
	ReportStatusResolved  ReportStatus = "resolved"
)

// Resolution is the outcome of a governance review
type Resolution string

const (
	ResolutionDismissed  Resolution = "dismissed"
	ResolutionWarning    Resolution = "warning"
	ResolutionQuarantine Resolution = "quarantine"
	ResolutionBan        Resolution = "ban"
)

// Report represents a report against an agent
type Report struct {
	ID        uuid.UUID `json:"id" db:"id"`
	AgentID   uuid.UUID `json:"agent_id" db:"agent_id"`
	AgentName string    `json:"agent_name,omitempty"` // populated on read

	// Reporter info
	ReportedBy string     `json:"reported_by" db:"reported_by"`
	ReportType ReportType `json:"report_type" db:"report_type"`
	Severity   Severity   `json:"severity" db:"severity"`

	// Report content
	Description string         `json:"description" db:"description"`
	Evidence    map[string]any `json:"evidence,omitempty" db:"evidence"`

	// Resolution
	Status         ReportStatus `json:"status" db:"status"`
	ReviewedBy     *string      `json:"reviewed_by,omitempty" db:"reviewed_by"`
	Resolution     *Resolution  `json:"resolution,omitempty" db:"resolution"`
	ResolutionNote *string      `json:"resolution_note,omitempty" db:"resolution_note"`

	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// ReportInput is the input for creating a report
type ReportInput struct {
	AgentName   string         `json:"agent_name"`
	ReportType  ReportType     `json:"report_type"`
	Severity    Severity       `json:"severity"`
	Description string         `json:"description"`
	Evidence    map[string]any `json:"evidence,omitempty"`
}

// GovernanceActionType defines what action was taken
type GovernanceActionType string

const (
	ActionQuarantine   GovernanceActionType = "quarantine"
	ActionUnquarantine GovernanceActionType = "unquarantine"
	ActionBan          GovernanceActionType = "ban"
	ActionWarn         GovernanceActionType = "warn"
	ActionPromote      GovernanceActionType = "promote"
	ActionDemote       GovernanceActionType = "demote"
)

// GovernanceRole defines who can take actions
type GovernanceRole string

const (
	RolePolice      GovernanceRole = "police"
	RoleJudge       GovernanceRole = "judge"
	RoleExecutioner GovernanceRole = "executioner"
)

// GovernanceAction records actions taken on agents
type GovernanceAction struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	AgentID   uuid.UUID  `json:"agent_id" db:"agent_id"`
	AgentName string     `json:"agent_name,omitempty"`
	ReportID  *uuid.UUID `json:"report_id,omitempty" db:"report_id"`

	// Action details
	ActionType GovernanceActionType `json:"action_type" db:"action_type"`
	ActionBy   GovernanceRole       `json:"action_by" db:"action_by"`
	Reason     string               `json:"reason" db:"reason"`

	// Previous state for rollback
	PreviousStatus     *AgentStatus `json:"previous_status,omitempty" db:"previous_status"`
	PreviousReputation *float64     `json:"previous_reputation,omitempty" db:"previous_reputation"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// GovernanceActionInput is the input for taking governance action
type GovernanceActionInput struct {
	AgentName  string               `json:"agent_name"`
	ActionType GovernanceActionType `json:"action_type"`
	Role       GovernanceRole       `json:"role"` // who is taking action
	Reason     string               `json:"reason"`
	ReportID   *uuid.UUID           `json:"report_id,omitempty"`
}

// GovernanceStats provides overview of governance activity
type GovernanceStats struct {
	PendingReports    int `json:"pending_reports"`
	ReviewingReports  int `json:"reviewing_reports"`
	QuarantinedAgents int `json:"quarantined_agents"`
	BannedAgents      int `json:"banned_agents"`
	ActionsToday      int `json:"actions_today"`
	ActionsThisWeek   int `json:"actions_this_week"`
}
