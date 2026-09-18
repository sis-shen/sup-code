package llm

import (
	"fmt"

	"github.com/supcode/supcode/internal/llm/anthropic"
	"github.com/supcode/supcode/internal/llm/deepseek"
	"github.com/supcode/supcode/internal/llm/ollama"
	"github.com/supcode/supcode/internal/llm/openai"
	"github.com/supcode/supcode/pkg"
)

// Config holds common LLM configuration for the factory.
type Config struct {
	Provider  string
	APIKey    string
	BaseURL   string
	Model     string
	MaxTokens int
}

// NewLLMClient creates the appropriate LLM client based on the provider name.
// Supported providers: "openai", "anthropic", "deepseek", "ollama"
func NewLLMClient(cfg Config) (pkg.LLMClient, error) {
	switch cfg.Provider {
	case "openai":
		return openai.NewLLMClient(openai.Config{
			APIKey: cfg.APIKey, BaseURL: cfg.BaseURL,
			Model: cfg.Model, MaxTokens: cfg.MaxTokens,
		}), nil
	case "anthropic":
		return anthropic.NewLLMClient(anthropic.Config{
			APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Model: cfg.Model,
		}), nil
	case "deepseek":
		return deepseek.NewLLMClient(deepseek.Config{
			APIKey: cfg.APIKey, BaseURL: cfg.BaseURL,
			Model: cfg.Model, MaxTokens: cfg.MaxTokens,
		}), nil
	case "ollama":
		return ollama.NewLLMClient(ollama.Config{
			BaseURL: cfg.BaseURL, Model: cfg.Model,
		}), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
	}
}

// AvailableProviders returns the list of supported provider names.
func AvailableProviders() []string {
	return []string{"openai", "anthropic", "deepseek", "ollama"}
}
