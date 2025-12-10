package governance

import (
	"math"
	"testing"

	"github.com/aminghadersohi/agentmcp/internal/models"
)

func TestLogBase10(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"log10(1)", 1, 0},
		{"log10(10)", 10, 1},
		{"log10(100)", 100, 2},
		{"log10(1000)", 1000, 3},
		{"log10(0)", 0, 0},
		{"log10(-1)", -1, 0},
		{"log10(50)", 50, math.Log10(50)},
		{"log10(0.5)", 0.5, math.Log10(0.5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logBase10(tt.input)
			// Allow small floating point differences
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("logBase10(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLogBase10MatchesStdLib(t *testing.T) {
	// Test that our implementation matches math.Log10 for positive values
	testValues := []float64{1, 2, 5, 10, 50, 100, 500, 1000, 10000}

	for _, v := range testValues {
		our := logBase10(v)
		std := math.Log10(v)
		if math.Abs(our-std) > 0.0001 {
			t.Errorf("logBase10(%v) = %v, math.Log10 = %v", v, our, std)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.AutoQuarantineThreshold <= 0 {
		t.Error("AutoQuarantineThreshold should be positive")
	}

	if cfg.ReputationBanThreshold < 0 {
		t.Error("ReputationBanThreshold should be non-negative")
	}

	if !cfg.Enabled {
		t.Error("Governance should be enabled by default")
	}
}

func TestCalculateReputation(t *testing.T) {
	tests := []struct {
		name        string
		agent       *models.Agent
		successRate float64
		minExpected float64
		maxExpected float64
	}{
		{
			name: "new agent no feedback",
			agent: &models.Agent{
				FeedbackCount: 0,
				AvgRating:     0,
				UsageCount:    0,
			},
			successRate: 0,
			minExpected: 45,
			maxExpected: 55,
		},
		{
			name: "agent with perfect rating",
			agent: &models.Agent{
				FeedbackCount: 10,
				AvgRating:     5.0,
				UsageCount:    100,
			},
			successRate: 1.0,
			minExpected: 100,
			maxExpected: 100,
		},
		{
			name: "agent with poor rating",
			agent: &models.Agent{
				FeedbackCount: 10,
				AvgRating:     1.0,
				UsageCount:    5,
			},
			successRate: 0.2,
			minExpected: 50,
			maxExpected: 75,
		},
		{
			name: "agent with high usage",
			agent: &models.Agent{
				FeedbackCount: 5,
				AvgRating:     4.0,
				UsageCount:    10000,
			},
			successRate: 0.8,
			minExpected: 80,
			maxExpected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateReputation(tt.agent, tt.successRate)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("CalculateReputation() = %v, want between %v and %v",
					result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestCalculateReputationBounds(t *testing.T) {
	// Test that reputation is always between 0 and 100
	testCases := []struct {
		feedbackCount int
		avgRating     float64
		usageCount    int
		successRate   float64
	}{
		{0, 0, 0, 0},
		{1000, 5.0, 1000000, 1.0},
		{1, 1.0, 1, 0.0},
		{100, 3.0, 500, 0.5},
	}

	for _, tc := range testCases {
		agent := &models.Agent{
			FeedbackCount: tc.feedbackCount,
			AvgRating:     tc.avgRating,
			UsageCount:    tc.usageCount,
		}
		result := CalculateReputation(agent, tc.successRate)

		if result < 0 {
			t.Errorf("Reputation should not be negative, got %v", result)
		}
		if result > 100 {
			t.Errorf("Reputation should not exceed 100, got %v", result)
		}
	}
}

func TestCanPoliceAct(t *testing.T) {
	tests := []struct {
		action   models.GovernanceActionType
		expected bool
	}{
		{models.ActionQuarantine, true},
		{models.ActionWarn, true},
		{models.ActionUnquarantine, false},
		{models.ActionBan, false},
		{models.ActionPromote, false},
		{models.ActionDemote, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			result := CanPoliceAct(tt.action)
			if result != tt.expected {
				t.Errorf("CanPoliceAct(%v) = %v, want %v", tt.action, result, tt.expected)
			}
		})
	}
}

func TestCanJudgeAct(t *testing.T) {
	tests := []struct {
		action   models.GovernanceActionType
		expected bool
	}{
		{models.ActionQuarantine, true},
		{models.ActionUnquarantine, true},
		{models.ActionWarn, true},
		{models.ActionPromote, true},
		{models.ActionDemote, true},
		{models.ActionBan, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			result := CanJudgeAct(tt.action)
			if result != tt.expected {
				t.Errorf("CanJudgeAct(%v) = %v, want %v", tt.action, result, tt.expected)
			}
		})
	}
}

func TestCanExecutionerAct(t *testing.T) {
	tests := []struct {
		action   models.GovernanceActionType
		expected bool
	}{
		{models.ActionBan, true},
		{models.ActionQuarantine, false},
		{models.ActionUnquarantine, false},
		{models.ActionWarn, false},
		{models.ActionPromote, false},
		{models.ActionDemote, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			result := CanExecutionerAct(tt.action)
			if result != tt.expected {
				t.Errorf("CanExecutionerAct(%v) = %v, want %v", tt.action, result, tt.expected)
			}
		})
	}
}

func BenchmarkCalculateReputation(b *testing.B) {
	agent := &models.Agent{
		FeedbackCount: 50,
		AvgRating:     4.2,
		UsageCount:    1000,
	}
	for i := 0; i < b.N; i++ {
		CalculateReputation(agent, 0.85)
	}
}

func BenchmarkLogBase10(b *testing.B) {
	for i := 0; i < b.N; i++ {
		logBase10(float64(i%10000 + 1))
	}
}
