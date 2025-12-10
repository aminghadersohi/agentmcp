package database

import (
	"testing"
)

func TestEscapeLikePattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special chars",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "percent sign",
			input:    "100%",
			expected: "100\\%",
		},
		{
			name:     "underscore",
			input:    "hello_world",
			expected: "hello\\_world",
		},
		{
			name:     "backslash",
			input:    "path\\to\\file",
			expected: "path\\\\to\\\\file",
		},
		{
			name:     "all special chars",
			input:    "50%_test\\path",
			expected: "50\\%\\_test\\\\path",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "multiple percent",
			input:    "%%",
			expected: "\\%\\%",
		},
		{
			name:     "sql injection attempt",
			input:    "'; DROP TABLE agents; --",
			expected: "'; DROP TABLE agents; --",
		},
		{
			name:     "wildcard pattern",
			input:    "%admin%",
			expected: "\\%admin\\%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeLikePattern(tt.input)
			if result != tt.expected {
				t.Errorf("escapeLikePattern(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeLikePatternIdempotent(t *testing.T) {
	// Escaping already escaped string should double-escape
	input := "test\\%"
	first := escapeLikePattern(input)
	second := escapeLikePattern(first)

	// After first escape: test\\\\\\%
	// After second escape: test\\\\\\\\\\\\\\%
	if first == second {
		t.Error("escapeLikePattern should not be idempotent (double escape)")
	}
}

func BenchmarkEscapeLikePattern(b *testing.B) {
	input := "50%_test\\path with spaces"
	for i := 0; i < b.N; i++ {
		escapeLikePattern(input)
	}
}

func BenchmarkEscapeLikePatternNoSpecialChars(b *testing.B) {
	input := "normal search query without special characters"
	for i := 0; i < b.N; i++ {
		escapeLikePattern(input)
	}
}
