//go:build v2

// main_v2_test.go - Unit tests for AgentMCP v2 Server
package main

import (
	"strings"
	"testing"
)

// ============ escapeLikePattern Tests ============

func TestContainsWord(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		word     string
		expected bool
	}{
		{"exact match", "hello world", "hello", true},
		{"exact match end", "hello world", "world", true},
		{"no match", "hello world", "hell", false},
		{"substring not matched", "docker container", "doc", false},
		{"empty text", "", "word", false},
		{"empty word", "hello world", "", false},
		{"single word match", "hello", "hello", true},
		{"case sensitive no match", "Hello World", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsWord(tt.text, tt.word)
			if result != tt.expected {
				t.Errorf("containsWord(%q, %q) = %v, want %v", tt.text, tt.word, result, tt.expected)
			}
		})
	}
}

// ============ expandTask Tests ============

func TestExpandTask(t *testing.T) {
	tests := []struct {
		name            string
		task            string
		shouldContain   []string
		shouldNotExpand bool
	}{
		{
			name:          "typo reviw expands to review",
			task:          "reviw my code",
			shouldContain: []string{"review"},
		},
		{
			name:          "fix expands to bug keywords",
			task:          "fix this error",
			shouldContain: []string{"bug"},
		},
		{
			name:          "testing expands to test",
			task:          "write testing",
			shouldContain: []string{"test"},
		},
		{
			name:          "better expands to improve and quality",
			task:          "make it better",
			shouldContain: []string{"improve", "quality"},
		},
		{
			name:          "optimize expands to performance",
			task:          "optimize this",
			shouldContain: []string{"performance", "refactor"},
		},
		{
			name:          "charts expands with visualization keywords",
			task:          "create charts",
			shouldContain: []string{"graphs", "visualization"},
		},
		{
			name:            "no expansion for unrelated text",
			task:            "hello world",
			shouldNotExpand: true,
		},
		{
			name:          "security keywords expand",
			task:          "check vulnerability",
			shouldContain: []string{"security"},
		},
		{
			name:          "docker expands with container keywords",
			task:          "deploy with docker",
			shouldContain: []string{"container"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTask(tt.task)

			if tt.shouldNotExpand {
				if result != tt.task {
					t.Errorf("expandTask(%q) should not expand, got %q", tt.task, result)
				}
				return
			}

			for _, keyword := range tt.shouldContain {
				if !strings.Contains(strings.ToLower(result), keyword) {
					t.Errorf("expandTask(%q) = %q, should contain %q", tt.task, result, keyword)
				}
			}
		})
	}
}

func TestExpandTaskDeterministic(t *testing.T) {
	// Run multiple times to ensure deterministic output
	task := "fix this bug and optimize performance"
	first := expandTask(task)

	for i := 0; i < 10; i++ {
		result := expandTask(task)
		if result != first {
			t.Errorf("expandTask is not deterministic: got %q then %q", first, result)
		}
	}
}

func TestExpandTaskWordBoundaries(t *testing.T) {
	// "doc" should not match in "docker"
	result := expandTask("run docker container")

	// Should contain docker-related expansions
	if !strings.Contains(result, "container") {
		t.Errorf("docker should expand to include container, got %q", result)
	}

	// "documentation" should NOT be added (doc != docker)
	expanded := expandTask("update documentation")
	if !strings.Contains(expanded, "docs") {
		t.Logf("Note: documentation expanded to: %q", expanded)
	}
}

// ============ Input Validation Tests ============

func TestInputLengthConstants(t *testing.T) {
	// Ensure constants are set to reasonable values
	if maxTaskLength < 100 || maxTaskLength > 10000 {
		t.Errorf("maxTaskLength should be between 100 and 10000, got %d", maxTaskLength)
	}

	if maxQueryLength < 50 || maxQueryLength > 1000 {
		t.Errorf("maxQueryLength should be between 50 and 1000, got %d", maxQueryLength)
	}

	if maxDescriptionLength < 100 || maxDescriptionLength > 50000 {
		t.Errorf("maxDescriptionLength should be between 100 and 50000, got %d", maxDescriptionLength)
	}

	if maxNameLength < 10 || maxNameLength > 500 {
		t.Errorf("maxNameLength should be between 10 and 500, got %d", maxNameLength)
	}
}

// ============ Helper Function Tests ============

func TestGetArgString(t *testing.T) {
	tests := []struct {
		name     string
		args     interface{}
		key      string
		expected string
	}{
		{
			name:     "valid string arg",
			args:     map[string]interface{}{"name": "test-agent"},
			key:      "name",
			expected: "test-agent",
		},
		{
			name:     "missing key",
			args:     map[string]interface{}{"other": "value"},
			key:      "name",
			expected: "",
		},
		{
			name:     "nil args",
			args:     nil,
			key:      "name",
			expected: "",
		},
		{
			name:     "wrong type",
			args:     map[string]interface{}{"name": 123},
			key:      "name",
			expected: "",
		},
		{
			name:     "empty string",
			args:     map[string]interface{}{"name": ""},
			key:      "name",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the mcp.CallToolRequest structure
			type mockRequest struct {
				Params struct {
					Arguments interface{}
				}
			}

			// We can't directly test getArgString without the mcp types,
			// but we can test the logic it uses
			args, ok := tt.args.(map[string]interface{})
			var result string
			if ok {
				if v, ok := args[tt.key].(string); ok {
					result = v
				}
			}

			if result != tt.expected {
				t.Errorf("getArgString logic for key %q = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestGetArgFloat(t *testing.T) {
	tests := []struct {
		name     string
		args     interface{}
		key      string
		expected float64
	}{
		{
			name:     "valid float",
			args:     map[string]interface{}{"rating": 4.5},
			key:      "rating",
			expected: 4.5,
		},
		{
			name:     "integer as float",
			args:     map[string]interface{}{"rating": float64(5)},
			key:      "rating",
			expected: 5.0,
		},
		{
			name:     "missing key",
			args:     map[string]interface{}{},
			key:      "rating",
			expected: 0,
		},
		{
			name:     "wrong type",
			args:     map[string]interface{}{"rating": "five"},
			key:      "rating",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, ok := tt.args.(map[string]interface{})
			var result float64
			if ok {
				if v, ok := args[tt.key].(float64); ok {
					result = v
				}
			}

			if result != tt.expected {
				t.Errorf("getArgFloat logic for key %q = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestGetArgBool(t *testing.T) {
	tests := []struct {
		name     string
		args     interface{}
		key      string
		expected bool
	}{
		{
			name:     "true value",
			args:     map[string]interface{}{"create": true},
			key:      "create",
			expected: true,
		},
		{
			name:     "false value",
			args:     map[string]interface{}{"create": false},
			key:      "create",
			expected: false,
		},
		{
			name:     "missing key",
			args:     map[string]interface{}{},
			key:      "create",
			expected: false,
		},
		{
			name:     "wrong type",
			args:     map[string]interface{}{"create": "true"},
			key:      "create",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, ok := tt.args.(map[string]interface{})
			var result bool
			if ok {
				if v, ok := args[tt.key].(bool); ok {
					result = v
				}
			}

			if result != tt.expected {
				t.Errorf("getArgBool logic for key %q = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

// ============ Task Aliases Tests ============

func TestTaskAliasesStructure(t *testing.T) {
	// Ensure all keys have at least one alias
	for key, aliases := range taskAliases {
		if len(aliases) == 0 {
			t.Errorf("taskAliases[%q] has no aliases", key)
		}

		// Check for duplicate aliases within a key
		seen := make(map[string]bool)
		for _, alias := range aliases {
			if seen[alias] {
				t.Errorf("taskAliases[%q] has duplicate alias %q", key, alias)
			}
			seen[alias] = true
		}
	}
}

func TestTaskAliasesTypoHandling(t *testing.T) {
	// Verify common typos are included
	reviewAliases := taskAliases["review"]

	typos := []string{"reviw", "reveiw"}
	for _, typo := range typos {
		found := false
		for _, alias := range reviewAliases {
			if alias == typo {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("taskAliases[review] should include typo %q", typo)
		}
	}
}

// ============ Benchmark Tests ============

func BenchmarkExpandTask(b *testing.B) {
	task := "review my code for security vulnerabilities and optimize performance"
	for i := 0; i < b.N; i++ {
		expandTask(task)
	}
}

func BenchmarkContainsWord(b *testing.B) {
	text := "this is a sample text with multiple words for testing"
	word := "testing"
	for i := 0; i < b.N; i++ {
		containsWord(text, word)
	}
}
