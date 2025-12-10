// Package embeddings provides text embedding generation for semantic search
package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/pgvector/pgvector-go"
)

const (
	// DefaultDimension is the embedding dimension for all-MiniLM-L6-v2
	DefaultDimension = 384
)

// Engine defines the interface for generating embeddings
type Engine interface {
	// Embed generates an embedding for the given text
	Embed(ctx context.Context, text string) (pgvector.Vector, error)
	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([]pgvector.Vector, error)
	// Dimension returns the embedding dimension
	Dimension() int
}

// Config holds embedding engine configuration
type Config struct {
	// Type: "python" or "http"
	Type string
	// Model name for sentence-transformers
	Model string
	// HTTPEndpoint for HTTP-based embedding service
	HTTPEndpoint string
	// PythonPath for Python executable
	PythonPath string
	// Timeout for embedding operations
	Timeout time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Type:       "python",
		Model:      "all-MiniLM-L6-v2",
		PythonPath: "python3",
		Timeout:    30 * time.Second,
	}
}

// NewEngine creates an embedding engine based on config
func NewEngine(cfg Config) (Engine, error) {
	switch cfg.Type {
	case "python":
		return NewPythonEngine(cfg)
	case "http":
		return NewHTTPEngine(cfg)
	default:
		return nil, fmt.Errorf("unknown embedding engine type: %s", cfg.Type)
	}
}

// ============ Python-based Engine ============

// PythonEngine uses sentence-transformers via Python subprocess
type PythonEngine struct {
	model      string
	pythonPath string
	timeout    time.Duration
}

// NewPythonEngine creates a new Python-based embedding engine
func NewPythonEngine(cfg Config) (*PythonEngine, error) {
	return &PythonEngine{
		model:      cfg.Model,
		pythonPath: cfg.PythonPath,
		timeout:    cfg.Timeout,
	}, nil
}

// Embed generates an embedding for a single text
func (e *PythonEngine) Embed(ctx context.Context, text string) (pgvector.Vector, error) {
	vectors, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return pgvector.Vector{}, err
	}
	return vectors[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *PythonEngine) EmbedBatch(ctx context.Context, texts []string) ([]pgvector.Vector, error) {
	// Create Python script for embedding
	script := fmt.Sprintf(`
import sys
import json
from sentence_transformers import SentenceTransformer

model = SentenceTransformer('%s')
texts = json.loads(sys.argv[1])
embeddings = model.encode(texts)
print(json.dumps(embeddings.tolist()))
`, e.model)

	textsJSON, err := json.Marshal(texts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal texts: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, e.pythonPath, "-c", script, string(textsJSON))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("embedding failed: %s: %w", stderr.String(), err)
	}

	var embeddings [][]float32
	if err := json.Unmarshal(stdout.Bytes(), &embeddings); err != nil {
		return nil, fmt.Errorf("failed to parse embeddings: %w", err)
	}

	vectors := make([]pgvector.Vector, len(embeddings))
	for i, emb := range embeddings {
		vectors[i] = pgvector.NewVector(emb)
	}

	return vectors, nil
}

// Dimension returns the embedding dimension
func (e *PythonEngine) Dimension() int {
	return DefaultDimension
}

// ============ HTTP-based Engine ============

// HTTPEngine uses an HTTP endpoint for embeddings
type HTTPEngine struct {
	endpoint string
	client   *http.Client
}

// NewHTTPEngine creates a new HTTP-based embedding engine
func NewHTTPEngine(cfg Config) (*HTTPEngine, error) {
	if cfg.HTTPEndpoint == "" {
		return nil, fmt.Errorf("HTTP endpoint is required")
	}

	return &HTTPEngine{
		endpoint: strings.TrimSuffix(cfg.HTTPEndpoint, "/"),
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

type embedRequest struct {
	Texts []string `json:"texts"`
}

type embedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// Embed generates an embedding for a single text
func (e *HTTPEngine) Embed(ctx context.Context, text string) (pgvector.Vector, error) {
	vectors, err := e.EmbedBatch(ctx, []string{text})
	if err != nil {
		return pgvector.Vector{}, err
	}
	return vectors[0], nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *HTTPEngine) EmbedBatch(ctx context.Context, texts []string) ([]pgvector.Vector, error) {
	reqBody, err := json.Marshal(embedRequest{Texts: texts})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.endpoint+"/embed", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding service returned status %d", resp.StatusCode)
	}

	var result embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	vectors := make([]pgvector.Vector, len(result.Embeddings))
	for i, emb := range result.Embeddings {
		vectors[i] = pgvector.NewVector(emb)
	}

	return vectors, nil
}

// Dimension returns the embedding dimension
func (e *HTTPEngine) Dimension() int {
	return DefaultDimension
}

// ============ Utility Functions ============

// CreateAgentEmbedding creates an embedding from agent data
func CreateAgentEmbedding(engine Engine, ctx context.Context, name, description string, skills []string) (pgvector.Vector, error) {
	// Combine agent info into a single text for embedding
	text := fmt.Sprintf("%s. %s. Skills: %s",
		name,
		description,
		strings.Join(skills, ", "),
	)
	return engine.Embed(ctx, text)
}

// CosineSimilarity calculates cosine similarity between two vectors
func CosineSimilarity(a, b pgvector.Vector) float64 {
	aSlice := a.Slice()
	bSlice := b.Slice()

	if len(aSlice) != len(bSlice) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range aSlice {
		dotProduct += float64(aSlice[i]) * float64(bSlice[i])
		normA += float64(aSlice[i]) * float64(aSlice[i])
		normB += float64(bSlice[i]) * float64(bSlice[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
