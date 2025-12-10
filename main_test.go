//go:build !v2

// main_test.go - Unit tests for AgentMCP Server (v1)
package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// setupTestAgents creates temporary agent files for testing
func setupTestAgents(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "agentmcp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create test agent 1
	agent1 := `---
name: test-agent-1
version: 1.0.0
description: First test agent
model: sonnet
tools:
  - Read
  - Write
metadata:
  author: Test Author
  tags:
    - test
    - frontend
prompt: You are a test agent for testing purposes.
`

	// Create test agent 2
	agent2 := `---
name: test-agent-2
version: 2.0.0
description: Second test agent for backend
model: opus
tools:
  - Bash
  - Grep
metadata:
  author: Another Author
  tags:
    - test
    - backend
prompt: You are another test agent.
`

	if err := os.WriteFile(filepath.Join(tmpDir, "agent1.yaml"), []byte(agent1), 0644); err != nil {
		t.Fatalf("Failed to write agent1: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "agent2.yaml"), []byte(agent2), 0644); err != nil {
		t.Fatalf("Failed to write agent2: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestLoadAgents(t *testing.T) {
	tmpDir, cleanup := setupTestAgents(t)
	defer cleanup()

	srv := NewAgentServer(tmpDir, "")
	if err := srv.LoadAgents(); err != nil {
		t.Fatalf("LoadAgents failed: %v", err)
	}

	if len(srv.cache) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(srv.cache))
	}

	agent1, exists := srv.cache["test-agent-1"]
	if !exists {
		t.Error("test-agent-1 not found in cache")
	}

	if agent1.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", agent1.Version)
	}

	if agent1.Model != "sonnet" {
		t.Errorf("Expected model sonnet, got %s", agent1.Model)
	}

	if len(agent1.Tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(agent1.Tools))
	}
}

func TestListAgents(t *testing.T) {
	tmpDir, cleanup := setupTestAgents(t)
	defer cleanup()

	srv := NewAgentServer(tmpDir, "")
	if err := srv.LoadAgents(); err != nil {
		t.Fatalf("LoadAgents failed: %v", err)
	}

	// Test without filters
	req := mcp.CallToolRequest{
		Params: mcp.CallToolRequestParams{
			Arguments: nil,
		},
	}

	result, err := srv.listAgents(context.Background(), req)
	if err != nil {
		t.Fatalf("listAgents failed: %v", err)
	}

	if len(result.Content) == 0 {
		t.Error("Expected non-empty result")
	}

	// Test with tag filter
	args, _ := json.Marshal(map[string]any{
		"tags": []string{"frontend"},
	})
	req.Params.Arguments = args

	result, err = srv.listAgents(context.Background(), req)
	if err != nil {
		t.Fatalf("listAgents with tags failed: %v", err)
	}

	if len(result.Content) == 0 {
		t.Error("Expected non-empty result with frontend tag")
	}
}

func TestGetAgent(t *testing.T) {
	tmpDir, cleanup := setupTestAgents(t)
	defer cleanup()

	srv := NewAgentServer(tmpDir, "")
	if err := srv.LoadAgents(); err != nil {
		t.Fatalf("LoadAgents failed: %v", err)
	}

	// Test getting existing agent
	args, _ := json.Marshal(map[string]any{
		"name": "test-agent-1",
	})
	req := mcp.CallToolRequest{
		Params: mcp.CallToolRequestParams{
			Arguments: args,
		},
	}

	result, err := srv.getAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("getAgent failed: %v", err)
	}

	if result.IsError != nil && *result.IsError {
		t.Error("Expected successful result, got error")
	}

	// Test getting non-existent agent
	args, _ = json.Marshal(map[string]any{
		"name": "non-existent-agent",
	})
	req.Params.Arguments = args

	result, err = srv.getAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("getAgent failed: %v", err)
	}

	if result.IsError == nil || !*result.IsError {
		t.Error("Expected error for non-existent agent")
	}

	// Test with missing name parameter
	args, _ = json.Marshal(map[string]any{})
	req.Params.Arguments = args

	result, err = srv.getAgent(context.Background(), req)
	if err != nil {
		t.Fatalf("getAgent failed: %v", err)
	}

	if result.IsError == nil || !*result.IsError {
		t.Error("Expected error for missing name parameter")
	}
}

func TestSearchAgents(t *testing.T) {
	tmpDir, cleanup := setupTestAgents(t)
	defer cleanup()

	srv := NewAgentServer(tmpDir, "")
	if err := srv.LoadAgents(); err != nil {
		t.Fatalf("LoadAgents failed: %v", err)
	}

	tests := []struct {
		name          string
		query         string
		expectResults bool
	}{
		{"Search by name", "test-agent-1", true},
		{"Search by description", "backend", true},
		{"Search by tag", "frontend", true},
		{"Search no results", "nonexistent", false},
		{"Search partial match", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, _ := json.Marshal(map[string]any{
				"query": tt.query,
			})
			req := mcp.CallToolRequest{
				Params: mcp.CallToolRequestParams{
					Arguments: args,
				},
			}

			result, err := srv.searchAgents(context.Background(), req)
			if err != nil {
				t.Fatalf("searchAgents failed: %v", err)
			}

			hasResults := len(result.Content) > 0
			if hasResults != tt.expectResults {
				t.Errorf("Expected results=%v, got=%v for query '%s'", tt.expectResults, hasResults, tt.query)
			}
		})
	}

	// Test with empty query
	args, _ := json.Marshal(map[string]any{
		"query": "",
	})
	req := mcp.CallToolRequest{
		Params: mcp.CallToolRequestParams{
			Arguments: args,
		},
	}

	result, err := srv.searchAgents(context.Background(), req)
	if err != nil {
		t.Fatalf("searchAgents failed: %v", err)
	}

	if result.IsError == nil || !*result.IsError {
		t.Error("Expected error for empty query")
	}
}

func TestInvalidAgentYAML(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentmcp-test-invalid-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create invalid YAML
	invalidYAML := `this is not valid yaml: [[[`
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to write invalid yaml: %v", err)
	}

	srv := NewAgentServer(tmpDir, "")
	// Should not fail, just skip invalid files
	if err := srv.LoadAgents(); err != nil {
		t.Fatalf("LoadAgents should handle invalid YAML gracefully: %v", err)
	}

	if len(srv.cache) != 0 {
		t.Errorf("Expected 0 agents from invalid YAML, got %d", len(srv.cache))
	}
}

func TestAgentWithoutName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentmcp-test-noname-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create agent without name
	noNameAgent := `---
version: 1.0.0
description: Agent without name
model: sonnet
prompt: This agent has no name.
`
	if err := os.WriteFile(filepath.Join(tmpDir, "noname.yaml"), []byte(noNameAgent), 0644); err != nil {
		t.Fatalf("Failed to write no-name agent: %v", err)
	}

	srv := NewAgentServer(tmpDir, "")
	if err := srv.LoadAgents(); err != nil {
		t.Fatalf("LoadAgents failed: %v", err)
	}

	if len(srv.cache) != 0 {
		t.Errorf("Expected 0 agents (agent without name should be skipped), got %d", len(srv.cache))
	}
}

func TestEmptyAgentsDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "agentmcp-test-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srv := NewAgentServer(tmpDir, "")
	if err := srv.LoadAgents(); err != nil {
		t.Fatalf("LoadAgents should handle empty directory: %v", err)
	}

	if len(srv.cache) != 0 {
		t.Errorf("Expected 0 agents in empty directory, got %d", len(srv.cache))
	}
}
