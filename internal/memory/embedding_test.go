package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supcode/supcode/pkg"
)

func TestEmbeddingProvider_Constructor(t *testing.T) {
	p := NewEmbeddingProvider("test-key")
	if p.apiKey != "test-key" {
		t.Errorf("apiKey = %q, want %q", p.apiKey, "test-key")
	}
	if p.model != "text-embedding-3-small" {
		t.Errorf("default model = %q, want %q", p.model, "text-embedding-3-small")
	}

	p2 := NewEmbeddingProviderWithModel("key2", "text-embedding-ada-002", "http://localhost/test")
	if p2.model != "text-embedding-ada-002" {
		t.Errorf("model = %q, want %q", p2.model, "text-embedding-ada-002")
	}
	if p2.baseURL != "http://localhost/test" {
		t.Errorf("baseURL = %q, want %q", p2.baseURL, "http://localhost/test")
	}

	p3 := NewEmbeddingProviderWithModel("key3", "default-model", "")
	if p3.baseURL != "https://api.openai.com/v1/embeddings" {
		t.Errorf("default baseURL = %q", p3.baseURL)
	}
}

func TestEmbeddingProvider_BatchGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing auth header")
		}

		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"embedding": []float64{0.1, 0.2, 0.3}, "index": 0},
				{"embedding": []float64{0.4, 0.5, 0.6}, "index": 1},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewEmbeddingProviderWithModel("test-key", "test-model", server.URL)
	ctx := context.Background()

	vectors, err := p.BatchGenerate(ctx, []string{"hello", "world"})
	if err != nil {
		t.Fatalf("BatchGenerate: %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if len(vectors[0]) != 3 {
		t.Errorf("expected vector of length 3, got %d", len(vectors[0]))
	}
	if vectors[0][0] != 0.1 || vectors[0][2] != 0.3 {
		t.Errorf("first vector = %v, want [0.1, 0.2, 0.3]", vectors[0])
	}
}

func TestEmbeddingProvider_Generate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embedRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Input) != 1 || req.Input[0] != "test text" {
			t.Errorf("unexpected input: %v", req.Input)
		}
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"embedding": []float64{0.5, 0.5}, "index": 0},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewEmbeddingProviderWithModel("key", "m", server.URL)
	ctx := context.Background()

	vec, err := p.Generate(ctx, "test text")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(vec) != 2 {
		t.Errorf("expected length 2, got %d", len(vec))
	}
}

func TestEmbeddingProvider_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "invalid api key"},
		})
	}))
	defer server.Close()

	p := NewEmbeddingProviderWithModel("bad-key", "m", server.URL)
	ctx := context.Background()

	_, err := p.BatchGenerate(ctx, []string{"test"})
	if err == nil {
		t.Errorf("expected error from bad API response")
	}
}

func TestEmbeddingProvider_APIErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{"message": "rate limit exceeded"},
		})
	}))
	defer server.Close()

	p := NewEmbeddingProviderWithModel("key", "m", server.URL)
	ctx := context.Background()

	_, err := p.BatchGenerate(ctx, []string{"test"})
	if err == nil {
		t.Errorf("expected error from API error response")
	}
	if err != nil && err.Error() != "openai error: rate limit exceeded" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestEmbeddingProvider_BatchLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req embedRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Input) > 20 {
			t.Errorf("batch size > 20: %d", len(req.Input))
		}
		resp := map[string]interface{}{"data": []map[string]interface{}{}}
		for i, input := range req.Input {
			resp["data"] = append(resp["data"].([]map[string]interface{}),
				map[string]interface{}{"embedding": []float64{float64(i)}, "index": i})
			_ = input
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewEmbeddingProviderWithModel("key", "m", server.URL)
	ctx := context.Background()

	texts := make([]string, 30)
	vectors, _ := p.BatchGenerate(ctx, texts)
	if len(vectors) != 20 {
		t.Errorf("expected 20 results (batch limited to 20), got %d", len(vectors))
	}
}

func TestEmbeddingProvider_ZeroValueInterface(t *testing.T) {
	var _ pkg.EmbeddingProvider = (*EmbeddingProvider)(nil)
}
