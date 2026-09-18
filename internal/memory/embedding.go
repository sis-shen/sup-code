package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supcode/supcode/pkg"
)

// EmbeddingProvider implements pkg.EmbeddingProvider via the OpenAI Embeddings API.
type EmbeddingProvider struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string
}

// NewEmbeddingProvider creates a new OpenAI embedding provider.
func NewEmbeddingProvider(apiKey string) *EmbeddingProvider {
	return &EmbeddingProvider{
		apiKey:  apiKey,
		model:   "text-embedding-3-small",
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://api.openai.com/v1/embeddings",
	}
}

// NewEmbeddingProviderWithModel creates an embedding provider with a custom model.
func NewEmbeddingProviderWithModel(apiKey, model, baseURL string) *EmbeddingProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1/embeddings"
	}
	return &EmbeddingProvider{
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: baseURL,
	}
}

type embedRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type embedResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Generate creates an embedding vector for a single text.
func (p *EmbeddingProvider) Generate(ctx context.Context, text string) ([]float32, error) {
	vectors, err := p.BatchGenerate(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("embedding: empty response")
	}
	return vectors[0], nil
}

// SetClient overrides the HTTP client (useful for testing with httptest).
func (p *EmbeddingProvider) SetClient(c *http.Client) {
	p.client = c
}

// BatchGenerate creates embedding vectors for multiple texts (max 20 per batch).
func (p *EmbeddingProvider) BatchGenerate(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) > 20 {
		texts = texts[:20]
	}

	body := embedRequest{
		Input: texts,
		Model: p.model,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("embedding marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("embedding request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding api: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("embedding read: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding api status %d: %s", resp.StatusCode, string(raw))
	}

	var result embedResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("embedding decode: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("openai error: %s", result.Error.Message)
	}

	vectors := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(vectors) {
			vec := make([]float32, len(d.Embedding))
			for i, v := range d.Embedding {
				vec[i] = float32(v)
			}
			vectors[d.Index] = vec
		}
	}

	return vectors, nil
}

var _ pkg.EmbeddingProvider = (*EmbeddingProvider)(nil)
