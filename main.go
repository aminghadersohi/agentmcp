// main.go - MCP Agent Server
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"
)

const VERSION = "1.0.0"

// Agent represents an agent definition
type Agent struct {
	Name        string         `yaml:"name" json:"name"`
	Version     string         `yaml:"version" json:"version"`
	Description string         `yaml:"description" json:"description"`
	Model       string         `yaml:"model" json:"model"`
	Tools       []string       `yaml:"tools" json:"tools"`
	Metadata    map[string]any `yaml:"metadata" json:"metadata"`
	Prompt      string         `yaml:"prompt" json:"prompt"`
}

// AgentServer serves agent definitions via MCP
type AgentServer struct {
	agentsDir string
	cache     map[string]*Agent
	watcher   *fsnotify.Watcher
	apiKey    string
}

// NewAgentServer creates a new agent server
func NewAgentServer(agentsDir string, apiKey string) *AgentServer {
	return &AgentServer{
		agentsDir: agentsDir,
		cache:     make(map[string]*Agent),
		apiKey:    apiKey,
	}
}

// LoadAgents scans directory and loads all agent YAML files
func (s *AgentServer) LoadAgents() error {
	log.Printf("[INFO] Loading agents from: %s", s.agentsDir)

	// Create agents directory if it doesn't exist
	if err := os.MkdirAll(s.agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(s.agentsDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to glob agent files: %w", err)
	}

	// Also check for .yml extension
	ymlFiles, err := filepath.Glob(filepath.Join(s.agentsDir, "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to glob yml files: %w", err)
	}
	files = append(files, ymlFiles...)

	if len(files) == 0 {
		log.Printf("[WARN] No agent files found in %s", s.agentsDir)
		return nil
	}

	loadedCount := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Printf("[ERROR] Failed to read %s: %v", file, err)
			continue
		}

		var agent Agent
		if err := yaml.Unmarshal(data, &agent); err != nil {
			log.Printf("[ERROR] Failed to parse %s: %v", file, err)
			continue
		}

		// Validate required fields
		if agent.Name == "" {
			log.Printf("[WARN] Agent in %s has no name, skipping", file)
			continue
		}

		s.cache[agent.Name] = &agent
		log.Printf("[INFO] Loaded agent: %s v%s", agent.Name, agent.Version)
		loadedCount++
	}

	log.Printf("[INFO] Successfully loaded %d agents", loadedCount)
	return nil
}

// WatchAgents sets up file watcher for hot reload
func (s *AgentServer) WatchAgents() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	s.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					if strings.HasSuffix(event.Name, ".yaml") || strings.HasSuffix(event.Name, ".yml") {
						log.Printf("[INFO] Detected change in %s, reloading agents...", event.Name)
						if err := s.LoadAgents(); err != nil {
							log.Printf("[ERROR] Failed to reload agents: %v", err)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[ERROR] Watcher error: %v", err)
			}
		}
	}()

	if err := watcher.Add(s.agentsDir); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	log.Printf("[INFO] File watcher enabled for %s", s.agentsDir)
	return nil
}

// Close cleans up resources
func (s *AgentServer) Close() error {
	if s.watcher != nil {
		return s.watcher.Close()
	}
	return nil
}

// MCP Tool Handlers

// listAgents returns a list of all available agents
func (s *AgentServer) listAgents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("[INFO] listAgents called")

	var args struct {
		Tags []string `json:"tags"`
	}

	if request.Params.Arguments != nil {
		if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
		}
	}

	var agents []map[string]any

	for _, agent := range s.cache {
		// Filter by tags if specified
		if len(args.Tags) > 0 {
			agentTags := []string{}
			if tags, ok := agent.Metadata["tags"].([]any); ok {
				for _, t := range tags {
					if tag, ok := t.(string); ok {
						agentTags = append(agentTags, tag)
					}
				}
			}

			// Check if agent has all required tags
			hasAllTags := true
			for _, reqTag := range args.Tags {
				found := false
				for _, agentTag := range agentTags {
					if agentTag == reqTag {
						found = true
						break
					}
				}
				if !found {
					hasAllTags = false
					break
				}
			}

			if !hasAllTags {
				continue
			}
		}

		agents = append(agents, map[string]any{
			"name":        agent.Name,
			"version":     agent.Version,
			"description": agent.Description,
			"tags":        agent.Metadata["tags"],
		})
	}

	result := map[string]any{
		"agents": agents,
		"count":  len(agents),
	}

	return mcp.NewToolResultText(fmt.Sprintf("%v", result)), nil
}

// getAgent returns the full definition of a specific agent
func (s *AgentServer) getAgent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	log.Printf("[INFO] getAgent called for: %s", args.Name)

	if args.Name == "" {
		return mcp.NewToolResultError("name parameter is required"), nil
	}

	agent, exists := s.cache[args.Name]
	if !exists {
		return mcp.NewToolResultError(fmt.Sprintf("agent not found: %s", args.Name)), nil
	}

	// Convert agent to JSON string for better formatting
	agentJSON, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to serialize agent: %v", err)), nil
	}

	return mcp.NewToolResultText(string(agentJSON)), nil
}

// searchAgents searches for agents by keyword
func (s *AgentServer) searchAgents(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Query string `json:"query"`
	}

	if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid arguments: %v", err)), nil
	}

	log.Printf("[INFO] searchAgents called with query: %s", args.Query)

	if args.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	query := strings.ToLower(args.Query)
	var results []map[string]any

	for _, agent := range s.cache {
		// Search in name, description, and tags
		matches := false

		if strings.Contains(strings.ToLower(agent.Name), query) {
			matches = true
		}

		if strings.Contains(strings.ToLower(agent.Description), query) {
			matches = true
		}

		// Search in tags
		if tags, ok := agent.Metadata["tags"].([]any); ok {
			for _, t := range tags {
				if tag, ok := t.(string); ok {
					if strings.Contains(strings.ToLower(tag), query) {
						matches = true
						break
					}
				}
			}
		}

		if matches {
			results = append(results, map[string]any{
				"name":        agent.Name,
				"version":     agent.Version,
				"description": agent.Description,
				"tags":        agent.Metadata["tags"],
			})
		}
	}

	result := map[string]any{
		"results": results,
		"count":   len(results),
		"query":   args.Query,
	}

	return mcp.NewToolResultText(fmt.Sprintf("%v", result)), nil
}

func main() {
	// CLI flags
	agentsDir := flag.String("agents", getEnvOrDefault("MCP_AGENTS_DIR", "./agents"), "Path to agents directory")
	transport := flag.String("transport", getEnvOrDefault("MCP_TRANSPORT", "stdio"), "Transport: stdio or sse")
	port := flag.String("port", getEnvOrDefault("MCP_PORT", "8080"), "HTTP port (if using sse transport)")
	apiKey := flag.String("api-key", os.Getenv("MCP_API_KEY"), "Optional API key for authentication")
	watch := flag.Bool("watch", getEnvOrDefault("MCP_WATCH", "false") == "true", "Enable file watching for hot reload")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("mcp-serve v%s\n", VERSION)
		os.Exit(0)
	}

	log.Printf("[INFO] Starting mcp-serve v%s", VERSION)
	log.Printf("[INFO] Agents directory: %s", *agentsDir)
	log.Printf("[INFO] Transport: %s", *transport)

	// Initialize agent server
	agentServer := NewAgentServer(*agentsDir, *apiKey)
	defer agentServer.Close()

	// Load agents
	if err := agentServer.LoadAgents(); err != nil {
		log.Fatalf("[FATAL] Failed to load agents: %v", err)
	}

	if len(agentServer.cache) == 0 {
		log.Printf("[WARN] No agents loaded. Add .yaml files to %s", *agentsDir)
	}

	// Setup file watcher if enabled
	if *watch {
		if err := agentServer.WatchAgents(); err != nil {
			log.Printf("[WARN] Failed to setup file watcher: %v", err)
		}
	}

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"mcp-serve",
		VERSION,
		server.WithToolCapabilities(true),
	)

	// Register tools
	listAgentsTool := mcp.NewTool("list_agents",
		mcp.WithDescription("List all available agent definitions. Optionally filter by tags."),
		mcp.WithObject(
			mcp.WithProperty("tags",
				mcp.NewObjectProperty(
					mcp.WithDescription("Filter by tags (optional)"),
					mcp.WithItems("string"),
				),
			),
		),
	)

	getAgentTool := mcp.NewTool("get_agent",
		mcp.WithDescription("Get complete agent definition by name. Returns the full agent specification including prompt, tools, and metadata."),
		mcp.WithObject(
			mcp.WithProperty("name",
				mcp.NewObjectProperty(
					mcp.WithDescription("Agent name"),
					mcp.WithRequired(),
				),
			),
		),
	)

	searchAgentsTool := mcp.NewTool("search_agents",
		mcp.WithDescription("Search agents by keyword in name, description, or tags. Returns matching agents."),
		mcp.WithObject(
			mcp.WithProperty("query",
				mcp.NewObjectProperty(
					mcp.WithDescription("Search query"),
					mcp.WithRequired(),
				),
			),
		),
	)

	mcpServer.AddTool(listAgentsTool, agentServer.listAgents)
	mcpServer.AddTool(getAgentTool, agentServer.getAgent)
	mcpServer.AddTool(searchAgentsTool, agentServer.searchAgents)

	// Run server with selected transport
	switch *transport {
	case "stdio":
		log.Println("[INFO] Starting MCP server on stdio...")
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("[FATAL] Server error: %v", err)
		}
	case "sse":
		log.Printf("[INFO] Starting MCP server on HTTP port %s...", *port)
		if err := server.ServeSSE(mcpServer, *port); err != nil {
			log.Fatalf("[FATAL] Server error: %v", err)
		}
	default:
		log.Fatalf("[FATAL] Unknown transport: %s (use 'stdio' or 'sse')", *transport)
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
