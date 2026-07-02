package llm

import (
    "context"
    "fmt"
    "log"

    "github.com/supcode/supcode/pkg"
)

// ProviderPriority defines fallback priority: openai -> anthropic -> deepseek -> ollama
var ProviderPriority = []string{"openai", "anthropic", "deepseek", "ollama"}

// FallbackConfig holds configuration for each provider in the chain.
type FallbackConfig struct {
    APIKey    string
    BaseURL   string
    Model     string
    MaxTokens int
}

// FallbackClient wraps multiple LLM clients and provides failover logic.
type FallbackClient struct {
    providers []providerEntry
}

type providerEntry struct {
    name   string
    client pkg.LLMClient
}

// NewFallbackClient creates a FallbackClient with configured providers.
// It creates clients for all providers that have API keys configured,
// ordered by ProviderPriority.
func NewFallbackClient(apiKey string, baseURLs map[string]string, models map[string]string, maxTokens int) (*FallbackClient, error) {
    fc := &FallbackClient{}
    for _, name := range ProviderPriority {
        cfg := Config{
            Provider:  name,
            APIKey:    apiKey,
            BaseURL:   baseURLs[name],
            Model:     models[name],
            MaxTokens: maxTokens,
        }
        client, err := NewLLMClient(cfg)
        if err != nil {
            log.Printf("fallback: failed to create %s client: %v", name, err)
            continue
        }
        fc.providers = append(fc.providers, providerEntry{name: name, client: client})
    }
    if len(fc.providers) == 0 {
        return nil, fmt.Errorf("no LLM providers configured")
    }
    return fc, nil
}

// Chat implements pkg.LLMClient.Chat with failover.
// It tries each provider in priority order, falling through on error.
func (fc *FallbackClient) Chat(ctx context.Context, systemPrompt string, messages []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
    var lastErr error
    for _, entry := range fc.providers {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }

        eventCh, err := entry.client.Chat(ctx, systemPrompt, messages, tools)
        if err != nil {
            log.Printf("fallback: %s failed: %v, trying next", entry.name, err)
            lastErr = err
            continue
        }

        // Check first event - if it's an error, try next provider
        firstEvent, ok := <-eventCh
        if !ok {
            log.Printf("fallback: %s returned closed channel, trying next", entry.name)
            continue
        }

        // Create a new channel with this event + remaining events
        resultCh := make(chan pkg.StreamEvent, 64)
        resultCh <- firstEvent
        go func() {
            defer close(resultCh)
            for e := range eventCh {
                resultCh <- e
            }
        }()

        // If first event is an error, try next provider
        if firstEvent.Type == "error" {
            log.Printf("fallback: %s returned error: %s, trying next", entry.name, firstEvent.Error)
            lastErr = fmt.Errorf("%s: %s", entry.name, firstEvent.Error)
            continue
        }

        return resultCh, nil
    }

    if lastErr != nil {
        return nil, fmt.Errorf("all providers failed: %w", lastErr)
    }
    return nil, fmt.Errorf("all providers failed")
}

// Models implements pkg.LLMClient.Models (uses the primary provider).
func (fc *FallbackClient) Models(ctx context.Context) ([]pkg.ModelInfo, error) {
    if len(fc.providers) == 0 {
        return nil, fmt.Errorf("no providers configured")
    }
    return fc.providers[0].client.Models(ctx)
}

// ProviderName returns the primary provider name.
func (fc *FallbackClient) ProviderName() string {
    if len(fc.providers) == 0 { return "none" }
    return fc.providers[0].name
}

// compile-time interface check
var _ pkg.LLMClient = (*FallbackClient)(nil)
