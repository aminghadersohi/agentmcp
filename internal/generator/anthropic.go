// Package generator provides AI-powered agent generation
package generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aminghadersohi/agentmcp/internal/models"
	"gopkg.in/yaml.v3"
)

const (
	anthropicAPIURL = "https://api.anthropic.com/v1/messages"
	defaultModel    = "claude-sonnet-4-20250514"
	maxTokens       = 4096
)

// Config holds generator configuration
type Config struct {
	APIKey    string
	Model     string
	MaxTokens int
	Timeout   time.Duration
}

// DefaultConfig returns default generator configuration
func DefaultConfig() Config {
	return Config{
		Model:     defaultModel,
		MaxTokens: maxTokens,
		Timeout:   60 * time.Second,
	}
}

// Generator generates new agents using AI
type Generator struct {
	apiKey    string
	model     string
	maxTokens int
	client    *http.Client
}

// New creates a new agent generator
func New(cfg Config) (*Generator, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	return &Generator{
		apiKey:    cfg.APIKey,
		model:     cfg.Model,
		maxTokens: cfg.MaxTokens,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

// anthropicRequest is the request body for Anthropic API
type anthropicRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// GenerateFromSkills generates an agent definition from a list of skills
func (g *Generator) GenerateFromSkills(ctx context.Context, skills []string) (*models.Agent, error) {
	prompt := buildGenerationPrompt(skills)

	resp, err := g.callAPI(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	agent, err := parseAgentYAML(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated agent: %w", err)
	}

	// Mark as generated and set skills
	agent.IsGenerated = true
	agent.Skills = normalizeSkills(skills)
	agent.ReputationScore = 50.0 // Start with neutral reputation
	agent.Status = models.StatusActive

	return agent, nil
}

// buildGenerationPrompt creates the prompt for agent generation
func buildGenerationPrompt(skills []string) string {
	return fmt.Sprintf(`You are an expert at creating AI agent definitions. Create an agent with the following skills:

Skills required: %s

Generate a complete agent definition in YAML format with these exact fields:
- name: A descriptive, lowercase-kebab-case name (e.g., "react-typescript-expert")
- version: "1.0.0"
- description: Clear 1-2 sentence description of the agent's capabilities
- model: "sonnet" (use "opus" only if the skills require advanced reasoning)
- tools: Appropriate MCP tools from this list: [Read, Write, Grep, Glob, Edit, Bash]
- metadata: Include author as "generated", tags array matching the skills, and created timestamp
- prompt: A detailed system prompt that:
  1. Defines the agent's expertise clearly
  2. Lists core competencies
  3. Includes working principles
  4. Provides problem-solving approach
  5. Sets appropriate boundaries

Important:
- The name must be unique and descriptive
- The prompt should be comprehensive but focused
- Only include tools that are relevant to the skills
- Tags should include all the input skills plus related concepts

Respond ONLY with the YAML, no explanation or markdown code fences.

Example format:
---
name: example-agent
version: 1.0.0
description: Example agent description
model: sonnet
tools:
  - Read
  - Write
metadata:
  author: generated
  tags:
    - skill1
    - skill2
  created: 2025-01-01T00:00:00Z
prompt: |
  You are an expert...
`, strings.Join(skills, ", "))
}

// callAPI calls the Anthropic API
func (g *Generator) callAPI(ctx context.Context, prompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     g.model,
		MaxTokens: g.maxTokens,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", g.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return apiResp.Content[0].Text, nil
}

// parseAgentYAML parses the YAML response into an Agent struct
func parseAgentYAML(yamlContent string) (*models.Agent, error) {
	// Clean up the response (remove markdown fences if present)
	yamlContent = strings.TrimSpace(yamlContent)
	yamlContent = strings.TrimPrefix(yamlContent, "```yaml")
	yamlContent = strings.TrimPrefix(yamlContent, "```")
	yamlContent = strings.TrimSuffix(yamlContent, "```")
	yamlContent = strings.TrimSpace(yamlContent)

	// Parse YAML into intermediate struct
	var raw struct {
		Name        string         `yaml:"name"`
		Version     string         `yaml:"version"`
		Description string         `yaml:"description"`
		Model       string         `yaml:"model"`
		Tools       []string       `yaml:"tools"`
		Metadata    map[string]any `yaml:"metadata"`
		Prompt      string         `yaml:"prompt"`
	}

	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	// Validate required fields
	if raw.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}
	if raw.Prompt == "" {
		return nil, fmt.Errorf("agent prompt is required")
	}

	// Set defaults
	if raw.Version == "" {
		raw.Version = "1.0.0"
	}
	if raw.Model == "" {
		raw.Model = "sonnet"
	}

	return &models.Agent{
		Name:        raw.Name,
		Version:     raw.Version,
		Description: raw.Description,
		Model:       raw.Model,
		Tools:       raw.Tools,
		Metadata:    raw.Metadata,
		Prompt:      raw.Prompt,
	}, nil
}

// normalizeSkills normalizes skill strings
func normalizeSkills(skills []string) []string {
	normalized := make([]string, len(skills))
	for i, s := range skills {
		normalized[i] = strings.ToLower(strings.TrimSpace(s))
	}
	return normalized
}

// ValidateAgentDefinition validates an agent definition
func ValidateAgentDefinition(agent *models.Agent) error {
	if agent.Name == "" {
		return fmt.Errorf("name is required")
	}
	if agent.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if len(agent.Name) > 255 {
		return fmt.Errorf("name too long (max 255 characters)")
	}
	if len(agent.Prompt) > 50000 {
		return fmt.Errorf("prompt too long (max 50000 characters)")
	}

	// Validate model
	validModels := map[string]bool{"sonnet": true, "opus": true, "haiku": true}
	if agent.Model != "" && !validModels[agent.Model] {
		return fmt.Errorf("invalid model: %s", agent.Model)
	}

	// Validate tools
	validTools := map[string]bool{
		"Read": true, "Write": true, "Grep": true, "Glob": true,
		"Edit": true, "Bash": true, "WebFetch": true, "WebSearch": true,
	}
	for _, tool := range agent.Tools {
		if !validTools[tool] {
			return fmt.Errorf("invalid tool: %s", tool)
		}
	}

	return nil
}
