// main_v2.go - AgentMCP v2: Agent Ecosystem
// Build with: go build -tags v2 -o agentmcp-v2 .
//go:build v2

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aminghadersohi/agentmcp/internal/database"
	"github.com/aminghadersohi/agentmcp/internal/embeddings"
	"github.com/aminghadersohi/agentmcp/internal/generator"
	"github.com/aminghadersohi/agentmcp/internal/governance"
	"github.com/aminghadersohi/agentmcp/internal/models"
	"github.com/aminghadersohi/agentmcp/internal/migrations"
	sqlmigrations "github.com/aminghadersohi/agentmcp/migrations"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const VERSION = "2.0.0"

// Input validation limits
const (
	maxTaskLength        = 2000 // Max length for task descriptions
	maxQueryLength       = 500  // Max length for search queries
	maxDescriptionLength = 5000 // Max length for feedback/report descriptions
	maxNameLength        = 100  // Max length for agent names
)

// ServerV2 is the enhanced agent server with full ecosystem support
type ServerV2 struct {
	db         *database.DB
	embedder   embeddings.Engine
	generator  *generator.Generator
	governance *governance.Engine
}

// NewServerV2 creates a new v2 server
func NewServerV2(db *database.DB, embedder embeddings.Engine, gen *generator.Generator, gov *governance.Engine) *ServerV2 {
	return &ServerV2{
		db:         db,
		embedder:   embedder,
		generator:  gen,
		governance: gov,
	}
}

// getArgString extracts a string argument from the request
func getArgString(req mcp.CallToolRequest, key string) string {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return ""
	}
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

// getArgFloat extracts a float64 argument from the request
func getArgFloat(req mcp.CallToolRequest, key string) float64 {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return 0
	}
	if v, ok := args[key].(float64); ok {
		return v
	}
	return 0
}

// getArgBool extracts a bool argument from the request
func getArgBool(req mcp.CallToolRequest, key string) bool {
	args, ok := req.Params.Arguments.(map[string]interface{})
	if !ok {
		return false
	}
	if v, ok := args[key].(bool); ok {
		return v
	}
	return false
}

// ============ Original Tools (backward compatible) ============

func (s *ServerV2) listAgents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tagsStr := getArgString(req, "tags")

	// Parse comma-separated tags
	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	agents, err := s.db.ListAgents(ctx, tags)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list agents: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"agents": agents,
		"count":  len(agents),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

func (s *ServerV2) getAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := getArgString(req, "name")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	agent, err := s.db.GetAgent(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get agent: %v", err)), nil
	}
	if agent == nil {
		return mcp.NewToolResultError(fmt.Sprintf("agent not found: %s", name)), nil
	}

	// Increment usage
	s.db.IncrementUsage(ctx, agent.ID)

	result, _ := json.MarshalIndent(agent, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

func (s *ServerV2) searchAgents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := getArgString(req, "query")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}
	if len(query) > maxQueryLength {
		return mcp.NewToolResultError(fmt.Sprintf("query too long (max %d characters)", maxQueryLength)), nil
	}

	// Try full query first
	agents, err := s.db.SearchAgents(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	// If no results, try individual words
	if len(agents) == 0 {
		words := strings.Fields(query)
		agentMap := make(map[string]models.AgentSummary)
		for _, word := range words {
			if len(word) < 3 {
				continue
			}
			wordAgents, err := s.db.SearchAgents(ctx, word)
			if err == nil {
				for _, a := range wordAgents {
					agentMap[a.ID.String()] = a
				}
			}
		}
		for _, a := range agentMap {
			agents = append(agents, a)
		}
	}

	result, _ := json.MarshalIndent(map[string]any{
		"results": agents,
		"count":   len(agents),
		"query":   query,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// ============ New v2 Tools ============

// findSimilarAgents finds semantically similar agents
func (s *ServerV2) findSimilarAgents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	description := getArgString(req, "description")
	skillsStr := getArgString(req, "skills")
	limitF := getArgFloat(req, "limit")
	threshold := getArgFloat(req, "threshold")

	// Parse comma-separated skills
	var skills []string
	if skillsStr != "" {
		for _, sk := range strings.Split(skillsStr, ",") {
			sk = strings.TrimSpace(sk)
			if sk != "" {
				skills = append(skills, sk)
			}
		}
	}

	if description == "" && len(skills) == 0 {
		return mcp.NewToolResultError("description or skills required"), nil
	}

	limit := int(limitF)
	if limit <= 0 {
		limit = 5
	}
	if threshold <= 0 {
		threshold = 0.15 // Lower default for better recall (semantic scores can be low)
	}

	// Create search text
	searchText := description
	if len(skills) > 0 {
		searchText += " Skills: " + strings.Join(skills, ", ")
	}

	// Generate embedding
	embedding, err := s.embedder.Embed(ctx, searchText)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("embedding failed: %v", err)), nil
	}

	// Find similar
	similar, err := s.db.FindSimilarAgents(ctx, embedding, limit, threshold)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"similar_agents": similar,
		"count":          len(similar),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// requestAgentBySkills finds or creates an agent with specific skills
func (s *ServerV2) requestAgentBySkills(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	skillsStr := getArgString(req, "skills")
	createIfMissing := getArgBool(req, "create_if_missing")

	// Parse comma-separated skills
	var skills []string
	for _, sk := range strings.Split(skillsStr, ",") {
		sk = strings.TrimSpace(sk)
		if sk != "" {
			skills = append(skills, sk)
		}
	}

	if len(skills) == 0 {
		return mcp.NewToolResultError("skills is required"), nil
	}

	// Check cache first
	cached, err := s.db.GetCachedAgentBySkills(ctx, skills)
	if err == nil && cached != nil {
		result, _ := json.MarshalIndent(map[string]any{
			"agent":  cached,
			"source": "cache",
		}, "", "  ")
		return mcp.NewToolResultText(string(result)), nil
	}

	// Search for similar agents - also try keyword search first
	// Check if any agent has matching skills directly
	for _, skill := range skills {
		agents, err := s.db.SearchAgents(ctx, skill)
		if err == nil && len(agents) > 0 {
			agent, _ := s.db.GetAgentByID(ctx, agents[0].ID)
			if agent != nil {
				s.db.CacheSkillRequest(ctx, skills, agent.ID)
				result, _ := json.MarshalIndent(map[string]any{
					"agent":  agent,
					"source": "skill_match",
				}, "", "  ")
				return mcp.NewToolResultText(string(result)), nil
			}
		}
	}

	// Fallback to semantic search
	searchText := "Agent with skills: " + strings.Join(skills, ", ")
	embedding, err := s.embedder.Embed(ctx, searchText)
	if err == nil {
		similar, _ := s.db.FindSimilarAgents(ctx, embedding, 1, 0.3) // Lower threshold for better matching
		if len(similar) > 0 {
			// Use existing similar agent
			agent, _ := s.db.GetAgentByID(ctx, similar[0].Agent.ID)
			if agent != nil {
				s.db.CacheSkillRequest(ctx, skills, agent.ID)
				result, _ := json.MarshalIndent(map[string]any{
					"agent":      agent,
					"source":     "similar",
					"similarity": similar[0].Similarity,
				}, "", "  ")
				return mcp.NewToolResultText(string(result)), nil
			}
		}
	}

	// Generate new agent if allowed
	if !createIfMissing {
		return mcp.NewToolResultError("no matching agent found"), nil
	}

	if s.generator == nil {
		return mcp.NewToolResultError("agent generation not configured"), nil
	}

	newAgent, err := s.generator.GenerateFromSkills(ctx, skills)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("generation failed: %v", err)), nil
	}

	// Generate embedding for new agent
	if s.embedder != nil {
		emb, _ := embeddings.CreateAgentEmbedding(
			s.embedder, ctx, newAgent.Name, newAgent.Description, newAgent.Skills,
		)
		newAgent.Embedding = &emb
	}

	// Save to database
	if err := s.db.CreateAgent(ctx, newAgent); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save agent: %v", err)), nil
	}

	// Cache the skill request
	s.db.CacheSkillRequest(ctx, skills, newAgent.ID)

	result, _ := json.MarshalIndent(map[string]any{
		"agent":  newAgent,
		"source": "generated",
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// submitFeedback records feedback for an agent
func (s *ServerV2) submitFeedback(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	agentName := getArgString(req, "agent_name")
	rating := int(getArgFloat(req, "rating"))
	taskSuccess := getArgBool(req, "task_success")
	taskType := getArgString(req, "task_type")
	feedbackText := getArgString(req, "feedback_text")

	if agentName == "" {
		return mcp.NewToolResultError("agent_name is required"), nil
	}
	if rating < 1 || rating > 5 {
		return mcp.NewToolResultError("rating must be 1-5"), nil
	}

	agent, err := s.db.GetAgent(ctx, agentName)
	if err != nil || agent == nil {
		return mcp.NewToolResultError(fmt.Sprintf("agent not found: %s", agentName)), nil
	}

	feedback := &models.Feedback{
		AgentID:      agent.ID,
		Rating:       rating,
		TaskSuccess:  taskSuccess,
		TaskType:     taskType,
		FeedbackText: feedbackText,
	}

	if err := s.db.SubmitFeedback(ctx, feedback); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to submit feedback: %v", err)), nil
	}

	return mcp.NewToolResultText(`{"status": "feedback recorded"}`), nil
}

// getAgentReputation returns reputation details
func (s *ServerV2) getAgentReputation(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := getArgString(req, "name")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	agent, err := s.db.GetAgent(ctx, name)
	if err != nil || agent == nil {
		return mcp.NewToolResultError(fmt.Sprintf("agent not found: %s", name)), nil
	}

	rep, err := s.db.GetAgentReputation(ctx, agent.ID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get reputation: %v", err)), nil
	}

	result, _ := json.MarshalIndent(rep, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

// getTopAgents returns highest-rated agents
func (s *ServerV2) getTopAgents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := int(getArgFloat(req, "limit"))
	category := getArgString(req, "category")

	if limit <= 0 {
		limit = 10
	}

	agents, err := s.db.GetTopAgents(ctx, limit, category)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get top agents: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"top_agents": agents,
		"count":      len(agents),
		"category":   category,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// ============ Governance Tools ============

// reportAgent creates a report against an agent
func (s *ServerV2) reportAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	agentName := getArgString(req, "agent_name")
	reportType := getArgString(req, "report_type")
	severity := getArgString(req, "severity")
	description := getArgString(req, "description")

	if agentName == "" || description == "" {
		return mcp.NewToolResultError("agent_name and description are required"), nil
	}
	if len(agentName) > maxNameLength {
		return mcp.NewToolResultError(fmt.Sprintf("agent_name too long (max %d characters)", maxNameLength)), nil
	}
	if len(description) > maxDescriptionLength {
		return mcp.NewToolResultError(fmt.Sprintf("description too long (max %d characters)", maxDescriptionLength)), nil
	}

	// Validate report type
	validReportTypes := map[models.ReportType]bool{
		models.ReportTypeHarmful:     true,
		models.ReportTypeEthics:      true,
		models.ReportTypeIneffective: true,
		models.ReportTypeSpam:        true,
		models.ReportTypeOther:       true,
	}
	if !validReportTypes[models.ReportType(reportType)] {
		return mcp.NewToolResultError("invalid report_type: must be harmful, ethics, ineffective, spam, or other"), nil
	}

	// Validate severity
	validSeverities := map[models.Severity]bool{
		models.SeverityLow:      true,
		models.SeverityMedium:   true,
		models.SeverityHigh:     true,
		models.SeverityCritical: true,
	}
	if !validSeverities[models.Severity(severity)] {
		return mcp.NewToolResultError("invalid severity: must be low, medium, high, or critical"), nil
	}

	args := models.ReportInput{
		AgentName:   agentName,
		ReportType:  models.ReportType(reportType),
		Severity:    models.Severity(severity),
		Description: description,
	}

	report, err := s.governance.CreateReport(ctx, args, "anonymous")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create report: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"status":    "report created",
		"report_id": report.ID,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// reviewReports returns pending reports (for governance agents)
func (s *ServerV2) reviewReports(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	reports, err := s.governance.GetPendingReports(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get reports: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"pending_reports": reports,
		"count":           len(reports),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// governanceAction executes a governance action
func (s *ServerV2) governanceAction(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	agentName := getArgString(req, "agent_name")
	action := getArgString(req, "action")
	reason := getArgString(req, "reason")
	reputationDelta := getArgFloat(req, "reputation_delta")

	if agentName == "" || action == "" || reason == "" {
		return mcp.NewToolResultError("agent_name, action, and reason are required"), nil
	}

	agent, err := s.db.GetAgent(ctx, agentName)
	if err != nil || agent == nil {
		return mcp.NewToolResultError(fmt.Sprintf("agent not found: %s", agentName)), nil
	}

	var actionErr error
	switch models.GovernanceActionType(action) {
	case models.ActionQuarantine:
		actionErr = s.governance.Quarantine(ctx, agent.ID, models.RolePolice, reason, nil)
	case models.ActionUnquarantine:
		actionErr = s.governance.Unquarantine(ctx, agent.ID, reason)
	case models.ActionBan:
		// Ban requires judge role
		actionErr = s.governance.Quarantine(ctx, agent.ID, models.RoleJudge, reason, nil)
		if actionErr == nil {
			s.db.UpdateAgentStatus(ctx, agent.ID, models.StatusBanned)
		}
	case models.ActionDemote:
		// Adjust reputation down with audit trail
		newScore := agent.ReputationScore - reputationDelta
		if newScore < 0 {
			newScore = 0
		}
		actionErr = s.db.UpdateAgentReputation(ctx, agent.ID, newScore)
		if actionErr == nil {
			s.db.RecordGovernanceAction(ctx, &models.GovernanceAction{
				AgentID:    agent.ID,
				ActionType: models.ActionDemote,
				Reason:     reason,
			})
		}
	case models.ActionPromote:
		// Adjust reputation up with audit trail
		newScore := agent.ReputationScore + reputationDelta
		if newScore > 100 {
			newScore = 100
		}
		actionErr = s.db.UpdateAgentReputation(ctx, agent.ID, newScore)
		if actionErr == nil {
			s.db.RecordGovernanceAction(ctx, &models.GovernanceAction{
				AgentID:    agent.ID,
				ActionType: models.ActionPromote,
				Reason:     reason,
			})
		}
	case "adjust_reputation":
		// Direct reputation adjustment using reputation_delta parameter
		if reputationDelta == 0 {
			return mcp.NewToolResultError("reputation_delta is required for adjust_reputation action"), nil
		}
		actionErr = s.governance.AdjustReputation(ctx, agent.ID, reputationDelta, reason)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("unsupported action: %s (use: quarantine, unquarantine, ban, demote, promote, adjust_reputation)", action)), nil
	}

	if actionErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("action failed: %v", actionErr)), nil
	}

	return mcp.NewToolResultText(`{"status": "action executed"}`), nil
}

// governanceStats returns governance statistics
func (s *ServerV2) governanceStats(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats, err := s.governance.GetStats(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	result, _ := json.MarshalIndent(stats, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

// ============ Agent Registration ============

// registerAgent creates a new agent in the system
func (s *ServerV2) registerAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := getArgString(req, "name")
	description := getArgString(req, "description")
	model := getArgString(req, "model")
	prompt := getArgString(req, "prompt")
	skillsStr := getArgString(req, "skills")
	toolsStr := getArgString(req, "tools")
	tagsStr := getArgString(req, "tags")
	version := getArgString(req, "version")

	if name == "" || description == "" || prompt == "" {
		return mcp.NewToolResultError("name, description, and prompt are required"), nil
	}
	if len(name) > maxNameLength {
		return mcp.NewToolResultError(fmt.Sprintf("name too long (max %d characters)", maxNameLength)), nil
	}
	if len(description) > maxDescriptionLength {
		return mcp.NewToolResultError(fmt.Sprintf("description too long (max %d characters)", maxDescriptionLength)), nil
	}

	// Check if agent already exists
	existing, _ := s.db.GetAgent(ctx, name)
	if existing != nil {
		return mcp.NewToolResultError(fmt.Sprintf("agent '%s' already exists", name)), nil
	}

	// Parse skills
	var skills []string
	if skillsStr != "" {
		for _, sk := range strings.Split(skillsStr, ",") {
			sk = strings.TrimSpace(sk)
			if sk != "" {
				skills = append(skills, sk)
			}
		}
	}

	// Parse tools
	var tools []string
	if toolsStr != "" {
		for _, t := range strings.Split(toolsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tools = append(tools, t)
			}
		}
	}

	// Parse tags
	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	// Set defaults
	if model == "" {
		model = "sonnet"
	}
	if version == "" {
		version = "1.0.0"
	}
	if len(skills) == 0 {
		// Default skills from tools
		skills = tools
	}

	// Create agent
	createdBy := "api"
	agent := &models.Agent{
		Name:            name,
		Version:         version,
		Description:     description,
		Model:           model,
		Tools:           tools,
		Prompt:          prompt,
		Skills:          skills,
		Status:          models.StatusActive,
		ReputationScore: 50.0, // Start at neutral
		IsSystem:        false,
		IsGenerated:     false,
		CreatedBy:       &createdBy,
		Metadata: map[string]any{
			"tags": tags,
		},
	}

	// Generate embedding if available
	if s.embedder != nil {
		emb, err := embeddings.CreateAgentEmbedding(s.embedder, ctx, name, description, skills)
		if err == nil {
			agent.Embedding = &emb
		}
	}

	// Save to database
	if err := s.db.CreateAgent(ctx, agent); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create agent: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"status":  "created",
		"agent":   agent.Name,
		"id":      agent.ID,
		"message": fmt.Sprintf("Agent '%s' registered successfully", name),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// ============ Skills Tools ============

// listSkills lists all available skills
func (s *ServerV2) listSkills(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	category := getArgString(req, "category")
	tagsStr := getArgString(req, "tags")

	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	skills, err := s.db.ListSkills(ctx, category, tags)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list skills: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"skills": skills,
		"count":  len(skills),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// getSkill retrieves a skill by name
func (s *ServerV2) getSkill(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := getArgString(req, "name")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	skill, err := s.db.GetSkill(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get skill: %v", err)), nil
	}
	if skill == nil {
		return mcp.NewToolResultError(fmt.Sprintf("skill not found: %s", name)), nil
	}

	s.db.IncrementSkillUsage(ctx, skill.ID)

	result, _ := json.MarshalIndent(skill, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

// searchSkills searches skills by keyword
func (s *ServerV2) searchSkills(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := getArgString(req, "query")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	skills, err := s.db.SearchSkills(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"results": skills,
		"count":   len(skills),
		"query":   query,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// findSimilarSkills finds semantically similar skills
func (s *ServerV2) findSimilarSkills(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	description := getArgString(req, "description")
	limitF := getArgFloat(req, "limit")
	threshold := getArgFloat(req, "threshold")

	if description == "" {
		return mcp.NewToolResultError("description is required"), nil
	}

	limit := int(limitF)
	if limit <= 0 {
		limit = 5
	}
	if threshold <= 0 {
		threshold = 0.15
	}

	embedding, err := s.embedder.Embed(ctx, description)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("embedding failed: %v", err)), nil
	}

	similar, err := s.db.FindSimilarSkills(ctx, embedding, limit, threshold)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"similar_skills": similar,
		"count":          len(similar),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// useSkill finds the best skill for a task and returns its content
func (s *ServerV2) useSkill(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	task := getArgString(req, "task")
	if task == "" {
		return mcp.NewToolResultError("task description is required"), nil
	}

	var bestSkill *models.Skill
	var matchMethod string

	// Try semantic search first
	if s.embedder != nil {
		embedding, err := s.embedder.Embed(ctx, task)
		if err == nil {
			similar, _ := s.db.FindSimilarSkills(ctx, embedding, 3, 0.25)
			if len(similar) > 0 {
				bestSkill, _ = s.db.GetSkillByID(ctx, similar[0].Skill.ID)
				matchMethod = fmt.Sprintf("semantic (%.0f%% match)", similar[0].Similarity*100)
			}
		}
	}

	// Fallback to keyword search
	if bestSkill == nil {
		skills, err := s.db.SearchSkills(ctx, task)
		if err == nil && len(skills) > 0 {
			bestSkill, _ = s.db.GetSkillByID(ctx, skills[0].ID)
			matchMethod = "keyword"
		}
	}

	// Multi-word keyword search fallback - try individual words
	if bestSkill == nil {
		words := strings.Fields(task)
		for _, word := range words {
			if len(word) < 3 {
				continue // Skip short words like "to", "a", "the"
			}
			skills, err := s.db.SearchSkills(ctx, word)
			if err == nil && len(skills) > 0 {
				bestSkill, _ = s.db.GetSkillByID(ctx, skills[0].ID)
				matchMethod = fmt.Sprintf("keyword (matched '%s')", word)
				break
			}
		}
	}

	if bestSkill == nil {
		allSkills, _ := s.db.ListSkills(ctx, "", nil)
		availableNames := make([]string, 0, len(allSkills))
		for _, s := range allSkills {
			availableNames = append(availableNames, s.Name)
		}

		result, _ := json.MarshalIndent(map[string]any{
			"found":            false,
			"message":          fmt.Sprintf("No matching skill found for: %s", task),
			"available_skills": availableNames,
		}, "", "  ")
		return mcp.NewToolResultText(string(result)), nil
	}

	s.db.IncrementSkillUsage(ctx, bestSkill.ID)

	result, _ := json.MarshalIndent(map[string]any{
		"found":        true,
		"match_method": matchMethod,
		"skill": map[string]any{
			"name":        bestSkill.Name,
			"description": bestSkill.Description,
			"category":    bestSkill.Category,
			"content":     bestSkill.Content,
			"examples":    bestSkill.Examples,
			"tags":        bestSkill.Tags,
		},
		"instructions": "Use this skill's content as reference documentation for the task.",
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// registerSkill creates a new skill
func (s *ServerV2) registerSkill(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := getArgString(req, "name")
	description := getArgString(req, "description")
	category := getArgString(req, "category")
	content := getArgString(req, "content")
	tagsStr := getArgString(req, "tags")
	version := getArgString(req, "version")

	if name == "" || description == "" || content == "" {
		return mcp.NewToolResultError("name, description, and content are required"), nil
	}

	existing, _ := s.db.GetSkill(ctx, name)
	if existing != nil {
		return mcp.NewToolResultError(fmt.Sprintf("skill '%s' already exists", name)), nil
	}

	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	if version == "" {
		version = "1.0.0"
	}

	createdBy := "api"
	skill := &models.Skill{
		Name:            name,
		Version:         version,
		Description:     description,
		Category:        category,
		Content:         content,
		Tags:            tags,
		Status:          models.SkillStatusActive,
		ReputationScore: 50.0,
		IsSystem:        false,
		CreatedBy:       &createdBy,
		Metadata:        map[string]any{},
	}

	if s.embedder != nil {
		emb, err := s.embedder.Embed(ctx, name+" "+description+" "+content)
		if err == nil {
			skill.Embedding = &emb
		}
	}

	if err := s.db.CreateSkill(ctx, skill); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create skill: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"status":  "created",
		"skill":   skill.Name,
		"id":      skill.ID,
		"message": fmt.Sprintf("Skill '%s' registered successfully", name),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// ============ Commands Tools ============

// listCommands lists all available commands
func (s *ServerV2) listCommands(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	category := getArgString(req, "category")
	tagsStr := getArgString(req, "tags")

	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	commands, err := s.db.ListCommands(ctx, category, tags)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list commands: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"commands": commands,
		"count":    len(commands),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// getCommand retrieves a command by name
func (s *ServerV2) getCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := getArgString(req, "name")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	cmd, err := s.db.GetCommand(ctx, name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get command: %v", err)), nil
	}
	if cmd == nil {
		return mcp.NewToolResultError(fmt.Sprintf("command not found: %s", name)), nil
	}

	s.db.IncrementCommandUsage(ctx, cmd.ID)

	result, _ := json.MarshalIndent(cmd, "", "  ")
	return mcp.NewToolResultText(string(result)), nil
}

// searchCommands searches commands by keyword
func (s *ServerV2) searchCommands(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := getArgString(req, "query")
	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	commands, err := s.db.SearchCommands(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"results": commands,
		"count":   len(commands),
		"query":   query,
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// registerCommand creates a new command
func (s *ServerV2) registerCommand(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := getArgString(req, "name")
	description := getArgString(req, "description")
	prompt := getArgString(req, "prompt")
	category := getArgString(req, "category")
	tagsStr := getArgString(req, "tags")
	version := getArgString(req, "version")

	if name == "" || description == "" || prompt == "" {
		return mcp.NewToolResultError("name, description, and prompt are required"), nil
	}

	existing, _ := s.db.GetCommand(ctx, name)
	if existing != nil {
		return mcp.NewToolResultError(fmt.Sprintf("command '%s' already exists", name)), nil
	}

	var tags []string
	if tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	if version == "" {
		version = "1.0.0"
	}

	createdBy := "api"
	cmd := &models.Command{
		Name:            name,
		Version:         version,
		Description:     description,
		Prompt:          prompt,
		Category:        category,
		Tags:            tags,
		Status:          models.CommandStatusActive,
		ReputationScore: 50.0,
		IsSystem:        false,
		CreatedBy:       &createdBy,
		Metadata:        map[string]any{},
	}

	if s.embedder != nil {
		emb, err := s.embedder.Embed(ctx, name+" "+description+" "+prompt)
		if err == nil {
			cmd.Embedding = &emb
		}
	}

	if err := s.db.CreateCommand(ctx, cmd); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create command: %v", err)), nil
	}

	result, _ := json.MarshalIndent(map[string]any{
		"status":  "created",
		"command": cmd.Name,
		"id":      cmd.ID,
		"message": fmt.Sprintf("Command '%s' registered successfully", name),
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

// ============ Meta Tools ============

// taskAliases maps common terms to agent-related keywords for better matching
var taskAliases = map[string][]string{
	// Code review related
	"review":   {"code", "quality", "check", "audit", "inspect", "reviw", "reveiw"},
	"code":     {"programming", "coding", "development", "software", "script"},
	"bug":      {"fix", "debug", "error", "issue", "problem", "bugs", "fixing", "code-quality"},
	"refactor": {"clean", "improve", "restructure", "optimize"},

	// Testing related
	"test": {"testing", "tests", "unittest", "unit", "qa", "quality"},
	"e2e":  {"end-to-end", "integration", "functional"},

	// Security related
	"security": {"secure", "vulnerability", "vulnerabilities", "exploit", "attack", "pentest"},
	"audit":    {"review", "check", "assess", "examine"},

	// Documentation related
	"docs":  {"documentation", "document", "readme", "guide", "manual", "explain"},
	"write": {"create", "generate", "draft", "compose"},

	// Data related
	"data":    {"dataset", "database", "analytics", "statistics", "metrics"},
	"analyze": {"analysis", "examine", "study", "investigate", "explore"},
	"charts":  {"graphs", "visualization", "visualize", "plots", "dashboard", "report"},

	// Architecture related
	"architecture": {"design", "structure", "system", "scalable", "microservices"},
	"scale":        {"scalability", "scaling", "performance", "optimize", "load"},
	"performance":  {"optimize", "speed", "fast", "slow", "bottleneck", "efficient", "optimization"},
	"improve":      {"better", "enhance", "upgrade", "fix", "refine", "polish", "quality"},
	"quality":      {"better", "improve", "good", "clean", "nice", "readable"},

	// DevOps related
	"deploy": {"deployment", "release", "ship", "publish", "rollout"},
	"devops": {"ci", "cd", "pipeline", "cicd", "ci/cd", "infrastructure", "infra"},
	"docker": {"container", "containerize", "kubernetes", "k8s"},
}

// containsWord checks if a word exists as a complete word in text (word boundary matching)
func containsWord(text, word string) bool {
	words := strings.Fields(text)
	for _, w := range words {
		if w == word {
			return true
		}
	}
	return false
}

// expandTask expands a task with aliases for better matching using word boundaries
func expandTask(task string) string {
	taskLower := strings.ToLower(task)
	taskWords := strings.Fields(taskLower)
	wordSet := make(map[string]bool)
	for _, w := range taskWords {
		wordSet[w] = true
	}

	expanded := task

	// Sort keys for deterministic output (Go maps have random iteration order)
	keys := make([]string, 0, len(taskAliases))
	for k := range taskAliases {
		keys = append(keys, k)
	}
	// Simple sort for determinism
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		aliases := taskAliases[key]
		// Check if any alias word is in the task
		for _, alias := range aliases {
			if wordSet[alias] {
				// Add the main keyword if not already present
				if !wordSet[key] {
					expanded += " " + key
					wordSet[key] = true
				}
				break
			}
		}
		// Check if key is in task
		if wordSet[key] {
			// Add ALL aliases for better skill matching
			for _, alias := range aliases {
				if !wordSet[alias] {
					expanded += " " + alias
					wordSet[alias] = true
				}
			}
		}
	}

	return expanded
}

// useAgent finds the best agent for a task and returns its configuration
func (s *ServerV2) useAgent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	task := getArgString(req, "task")
	if task == "" {
		return mcp.NewToolResultError("task description is required"), nil
	}
	if len(task) > maxTaskLength {
		return mcp.NewToolResultError(fmt.Sprintf("task too long (max %d characters)", maxTaskLength)), nil
	}

	// Expand task with aliases for better matching
	expandedTask := expandTask(task)

	// First try semantic search if embeddings are available
	var bestAgent *models.Agent
	var matchMethod string
	var matchScore float64

	if s.embedder != nil {
		embedding, err := s.embedder.Embed(ctx, expandedTask)
		if err == nil {
			// Lower threshold to 0.25 for better recall
			similar, _ := s.db.FindSimilarAgents(ctx, embedding, 3, 0.25)
			if len(similar) > 0 {
				bestAgent, _ = s.db.GetAgentByID(ctx, similar[0].Agent.ID)
				matchScore = similar[0].Similarity
				matchMethod = fmt.Sprintf("semantic (%.0f%% match)", matchScore*100)
			}
		}
	}

	// Fallback to keyword search - try each word individually
	if bestAgent == nil {
		// First try the original task
		agents, err := s.db.SearchAgents(ctx, task)
		if err == nil && len(agents) > 0 {
			bestAgent, _ = s.db.GetAgentByID(ctx, agents[0].ID)
			matchMethod = "keyword"
		}

		// If no match, try individual words from expanded task
		if bestAgent == nil {
			words := strings.Fields(expandedTask)
			for _, word := range words {
				if len(word) < 3 { // Skip short words
					continue
				}
				agents, err := s.db.SearchAgents(ctx, word)
				if err == nil && len(agents) > 0 {
					bestAgent, _ = s.db.GetAgentByID(ctx, agents[0].ID)
					matchMethod = fmt.Sprintf("keyword (%s)", word)
					break
				}
			}
		}
	}

	// No match found - provide helpful suggestions
	if bestAgent == nil {
		suggestions := []string{
			"'review my code' for code quality feedback",
			"'security audit' for vulnerability scanning",
			"'write tests' for test creation",
			"'create documentation' for technical docs",
			"'analyze data' for data insights",
			"'design architecture' for system design",
			"'deploy to kubernetes' for DevOps help",
		}

		// Get actual available agents from database
		allAgents, _ := s.db.ListAgents(ctx, nil)
		availableNames := make([]string, 0, len(allAgents))
		for _, a := range allAgents {
			availableNames = append(availableNames, a.Name)
		}

		result, _ := json.MarshalIndent(map[string]any{
			"found":            false,
			"message":          fmt.Sprintf("No matching agent found for: %s", task),
			"suggestions":      suggestions,
			"available_agents": availableNames,
		}, "", "  ")
		return mcp.NewToolResultText(string(result)), nil
	}

	// Increment usage
	s.db.IncrementUsage(ctx, bestAgent.ID)

	// Return agent configuration for the LLM to adopt
	result, _ := json.MarshalIndent(map[string]any{
		"found":        true,
		"match_method": matchMethod,
		"agent": map[string]any{
			"name":        bestAgent.Name,
			"description": bestAgent.Description,
			"model":       bestAgent.Model,
			"skills":      bestAgent.Skills,
			"prompt":      bestAgent.Prompt,
			"tools":       bestAgent.Tools,
		},
		"instructions": "Adopt this agent's persona and use its prompt as guidance for the task. " +
			"Follow the agent's specialized approach and expertise.",
	}, "", "  ")

	return mcp.NewToolResultText(string(result)), nil
}

func main() {
	// CLI flags
	dbHost := flag.String("db-host", getEnvOrDefault("DB_HOST", "localhost"), "Database host")
	dbPort := flag.Int("db-port", getEnvOrDefaultInt("DB_PORT", 5432), "Database port")
	dbName := flag.String("db-name", getEnvOrDefault("DB_NAME", "mcp_serve"), "Database name")
	dbUser := flag.String("db-user", getEnvOrDefault("DB_USER", "mcp"), "Database user")
	dbPass := flag.String("db-pass", os.Getenv("DB_PASSWORD"), "Database password")

	embeddingType := flag.String("embedding-type", getEnvOrDefault("EMBEDDING_TYPE", "http"), "Embedding type: python or http")
	embeddingURL := flag.String("embedding-url", getEnvOrDefault("EMBEDDING_URL", "http://localhost:8081"), "Embedding service URL")

	anthropicKey := flag.String("anthropic-key", os.Getenv("ANTHROPIC_API_KEY"), "Anthropic API key")

	transport := flag.String("transport", getEnvOrDefault("MCP_TRANSPORT", "stdio"), "Transport: stdio or sse")
	port := flag.String("port", getEnvOrDefault("MCP_PORT", "8080"), "HTTP port")

	migrate := flag.Bool("migrate", getEnvOrDefaultBool("AUTO_MIGRATE", false), "Run database migrations")
	migrateOnly := flag.Bool("migrate-only", false, "Run migrations and exit")

	version := flag.Bool("version", false, "Print version")
	flag.Parse()

	if *version {
		fmt.Printf("agentmcp v%s\n", VERSION)
		os.Exit(0)
	}

	log.Printf("[INFO] Starting agentmcp v%s", VERSION)

	// Initialize database
	db, err := database.New(database.Config{
		Host:     *dbHost,
		Port:     *dbPort,
		Database: *dbName,
		User:     *dbUser,
		Password: *dbPass,
		MaxConns: 20,
	})
	if err != nil {
		log.Fatalf("[FATAL] Database connection failed: %v", err)
	}
	defer db.Close()
	log.Println("[INFO] Database connected")

	// Run migrations if requested
	if *migrate || *migrateOnly {
		log.Println("[INFO] Running database migrations...")
		runner := migrations.NewRunner(db.Pool(), sqlmigrations.Files)
		if err := runner.Run(context.Background()); err != nil {
			log.Fatalf("[FATAL] Migration failed: %v", err)
		}
		log.Println("[INFO] Migrations completed successfully")

		if *migrateOnly {
			log.Println("[INFO] Migration-only mode, exiting")
			os.Exit(0)
		}
	}

	// Initialize embeddings
	var embedder embeddings.Engine
	embedCfg := embeddings.Config{
		Type:         *embeddingType,
		HTTPEndpoint: *embeddingURL,
		Timeout:      30 * time.Second,
	}
	embedder, err = embeddings.NewEngine(embedCfg)
	if err != nil {
		log.Printf("[WARN] Embedding engine failed: %v (semantic search disabled)", err)
	} else {
		log.Println("[INFO] Embedding engine initialized")
	}

	// Initialize generator
	var gen *generator.Generator
	if *anthropicKey != "" {
		gen, err = generator.New(generator.Config{
			APIKey:    *anthropicKey,
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 4096,
			Timeout:   60 * time.Second,
		})
		if err != nil {
			log.Printf("[WARN] Generator failed: %v (agent generation disabled)", err)
		} else {
			log.Println("[INFO] Agent generator initialized")
		}
	}

	// Initialize governance
	gov := governance.New(db, governance.DefaultConfig())
	log.Println("[INFO] Governance engine initialized")

	// Create server
	srv := NewServerV2(db, embedder, gen, gov)

	// Create MCP server
	mcpServer := server.NewMCPServer("agentmcp", VERSION)

	// Register original tools
	mcpServer.AddTool(mcp.NewTool("list_agents",
		mcp.WithDescription("List all available agents. Optionally filter by tags."),
		mcp.WithString("tags", mcp.Description("Comma-separated list of tags to filter by")),
	), srv.listAgents)

	mcpServer.AddTool(mcp.NewTool("get_agent",
		mcp.WithDescription("Get complete agent definition by name."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the agent to retrieve")),
	), srv.getAgent)

	mcpServer.AddTool(mcp.NewTool("search_agents",
		mcp.WithDescription("Search agents by keyword."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query string")),
	), srv.searchAgents)

	// Register v2 tools
	mcpServer.AddTool(mcp.NewTool("find_similar_agents",
		mcp.WithDescription("Find semantically similar agents using AI embeddings."),
		mcp.WithString("description", mcp.Description("Description to find similar agents for")),
		mcp.WithString("skills", mcp.Description("Comma-separated list of skills to search for")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 5)")),
		mcp.WithNumber("threshold", mcp.Description("Similarity threshold 0-1 (default 0.7)")),
	), srv.findSimilarAgents)

	mcpServer.AddTool(mcp.NewTool("request_agent_by_skills",
		mcp.WithDescription("Request an agent with specific skills. Will find existing or generate new."),
		mcp.WithString("skills", mcp.Required(), mcp.Description("Comma-separated list of required skills")),
		mcp.WithBoolean("create_if_missing", mcp.Description("Generate new agent if none matches (default false)")),
	), srv.requestAgentBySkills)

	mcpServer.AddTool(mcp.NewTool("submit_feedback",
		mcp.WithDescription("Submit performance feedback for an agent (rating 1-5)."),
		mcp.WithString("agent_name", mcp.Required(), mcp.Description("Name of the agent")),
		mcp.WithNumber("rating", mcp.Required(), mcp.Description("Rating from 1-5")),
		mcp.WithBoolean("task_success", mcp.Description("Whether the task was successful")),
		mcp.WithString("task_type", mcp.Description("Type of task performed")),
		mcp.WithString("feedback_text", mcp.Description("Optional feedback text")),
	), srv.submitFeedback)

	mcpServer.AddTool(mcp.NewTool("get_agent_reputation",
		mcp.WithDescription("Get detailed reputation information for an agent."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the agent")),
	), srv.getAgentReputation)

	mcpServer.AddTool(mcp.NewTool("get_top_agents",
		mcp.WithDescription("Get the highest-rated agents, optionally by category."),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 10)")),
		mcp.WithString("category", mcp.Description("Optional category/skill to filter by")),
	), srv.getTopAgents)

	// Register governance tools
	mcpServer.AddTool(mcp.NewTool("report_agent",
		mcp.WithDescription("Report an agent for governance review."),
		mcp.WithString("agent_name", mcp.Required(), mcp.Description("Name of the agent to report")),
		mcp.WithString("report_type", mcp.Required(), mcp.Description("Type: harmful_output, prompt_injection, policy_violation, performance, other")),
		mcp.WithString("severity", mcp.Required(), mcp.Description("Severity: low, medium, high, critical")),
		mcp.WithString("description", mcp.Required(), mcp.Description("Detailed description of the issue")),
	), srv.reportAgent)

	mcpServer.AddTool(mcp.NewTool("review_reports",
		mcp.WithDescription("[Governance] View pending reports for review."),
	), srv.reviewReports)

	mcpServer.AddTool(mcp.NewTool("governance_action",
		mcp.WithDescription("[Governance] Execute a governance action (quarantine, ban, etc)."),
		mcp.WithString("agent_name", mcp.Required(), mcp.Description("Name of the agent")),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: quarantine, unquarantine, ban, unban, adjust_reputation")),
		mcp.WithString("reason", mcp.Required(), mcp.Description("Reason for the action")),
		mcp.WithNumber("reputation_delta", mcp.Description("Reputation adjustment amount (for adjust_reputation)")),
	), srv.governanceAction)

	mcpServer.AddTool(mcp.NewTool("governance_stats",
		mcp.WithDescription("[Governance] Get governance system statistics."),
	), srv.governanceStats)

	// Register agent registration tool
	mcpServer.AddTool(mcp.NewTool("register_agent",
		mcp.WithDescription("Register a new agent in the system."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Unique name for the agent (lowercase, hyphens ok)")),
		mcp.WithString("description", mcp.Required(), mcp.Description("Brief description of what the agent does")),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("The system prompt that defines the agent's behavior")),
		mcp.WithString("model", mcp.Description("Model to use: sonnet, opus, haiku (default: sonnet)")),
		mcp.WithString("skills", mcp.Description("Comma-separated list of skills")),
		mcp.WithString("tools", mcp.Description("Comma-separated list of tools: Read, Write, Edit, Bash, Grep, Glob")),
		mcp.WithString("tags", mcp.Description("Comma-separated list of tags for categorization")),
		mcp.WithString("version", mcp.Description("Version string (default: 1.0.0)")),
	), srv.registerAgent)

	// Register skills tools
	mcpServer.AddTool(mcp.NewTool("list_skills",
		mcp.WithDescription("List all available skills (packaged knowledge for tools like kubectl, docker, curl, etc)."),
		mcp.WithString("category", mcp.Description("Filter by category: devops, api, database, cloud, cli")),
		mcp.WithString("tags", mcp.Description("Comma-separated list of tags to filter by")),
	), srv.listSkills)

	mcpServer.AddTool(mcp.NewTool("get_skill",
		mcp.WithDescription("Get a skill's complete content including documentation and examples."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the skill to retrieve")),
	), srv.getSkill)

	mcpServer.AddTool(mcp.NewTool("search_skills",
		mcp.WithDescription("Search skills by keyword."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query string")),
	), srv.searchSkills)

	mcpServer.AddTool(mcp.NewTool("find_similar_skills",
		mcp.WithDescription("Find semantically similar skills using AI embeddings."),
		mcp.WithString("description", mcp.Required(), mcp.Description("Description of what you need help with")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 5)")),
		mcp.WithNumber("threshold", mcp.Description("Similarity threshold 0-1 (default 0.15)")),
	), srv.findSimilarSkills)

	mcpServer.AddTool(mcp.NewTool("use_skill",
		mcp.WithDescription("Find the best skill for a task and return its documentation/content."),
		mcp.WithString("task", mcp.Required(), mcp.Description("Description of what you need to do (e.g., 'use kubectl to debug pods')")),
	), srv.useSkill)

	mcpServer.AddTool(mcp.NewTool("register_skill",
		mcp.WithDescription("Register a new skill (packaged knowledge/documentation)."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Unique name for the skill (e.g., 'kubectl', 'docker-cli')")),
		mcp.WithString("description", mcp.Required(), mcp.Description("Brief description of what the skill covers")),
		mcp.WithString("content", mcp.Required(), mcp.Description("The actual documentation/knowledge content")),
		mcp.WithString("category", mcp.Description("Category: devops, api, database, cloud, cli")),
		mcp.WithString("tags", mcp.Description("Comma-separated list of tags")),
		mcp.WithString("version", mcp.Description("Version string (default: 1.0.0)")),
	), srv.registerSkill)

	// Register commands tools
	mcpServer.AddTool(mcp.NewTool("list_commands",
		mcp.WithDescription("List all available slash commands that can be synced to your project."),
		mcp.WithString("category", mcp.Description("Filter by category: code, git, test, deploy")),
		mcp.WithString("tags", mcp.Description("Comma-separated list of tags to filter by")),
	), srv.listCommands)

	mcpServer.AddTool(mcp.NewTool("get_command",
		mcp.WithDescription("Get a command's complete definition including prompt template."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the command to retrieve")),
	), srv.getCommand)

	mcpServer.AddTool(mcp.NewTool("search_commands",
		mcp.WithDescription("Search commands by keyword."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query string")),
	), srv.searchCommands)

	mcpServer.AddTool(mcp.NewTool("register_command",
		mcp.WithDescription("Register a new slash command."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Unique name for the command (e.g., 'review-pr', 'fix-tests')")),
		mcp.WithString("description", mcp.Required(), mcp.Description("Brief description of what the command does")),
		mcp.WithString("prompt", mcp.Required(), mcp.Description("The command's prompt template")),
		mcp.WithString("category", mcp.Description("Category: code, git, test, deploy")),
		mcp.WithString("tags", mcp.Description("Comma-separated list of tags")),
		mcp.WithString("version", mcp.Description("Version string (default: 1.0.0)")),
	), srv.registerCommand)

	// Register meta tools
	mcpServer.AddTool(mcp.NewTool("use_agent",
		mcp.WithDescription("Find and adopt the best agent for a task. Returns the agent's prompt and configuration to use as guidance."),
		mcp.WithString("task", mcp.Required(), mcp.Description("Description of the task you need help with")),
	), srv.useAgent)

	// Run server
	switch *transport {
	case "stdio":
		log.Println("[INFO] Starting MCP server on stdio...")
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("[FATAL] Server error: %v", err)
		}
	case "sse":
		log.Printf("[INFO] Starting MCP server on SSE port %s...", *port)
		sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL("http://localhost:"+*port))
		if err := sseServer.Start(":" + *port); err != nil {
			log.Fatalf("[FATAL] Server error: %v", err)
		}
	case "http":
		log.Printf("[INFO] Starting MCP server on Streamable HTTP port %s...", *port)
		httpServer := server.NewStreamableHTTPServer(mcpServer)
		if err := httpServer.Start(":" + *port); err != nil {
			log.Fatalf("[FATAL] Server error: %v", err)
		}
	default:
		log.Fatalf("[FATAL] Unknown transport: %s (supported: stdio, sse, http)", *transport)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		var i int
		fmt.Sscanf(v, "%d", &i)
		return i
	}
	return defaultValue
}

func getEnvOrDefaultBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1" || v == "yes"
	}
	return defaultValue
}
