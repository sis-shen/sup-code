package llm

import (
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFactory_OpenAI(t *testing.T) {
    client, err := NewLLMClient(Config{
        Provider: "openai", APIKey: "sk-test", Model: "gpt-4o", MaxTokens: 4096,
    })
    require.NoError(t, err)
    assert.Equal(t, "openai", client.ProviderName())
}

func TestFactory_Anthropic(t *testing.T) {
    client, err := NewLLMClient(Config{
        Provider: "anthropic", APIKey: "sk-test", Model: "claude-sonnet-4-20250514",
    })
    require.NoError(t, err)
    assert.Equal(t, "anthropic", client.ProviderName())
}

func TestFactory_DeepSeek(t *testing.T) {
    client, err := NewLLMClient(Config{
        Provider: "deepseek", APIKey: "sk-test", Model: "deepseek-chat", MaxTokens: 4096,
    })
    require.NoError(t, err)
    assert.Equal(t, "deepseek", client.ProviderName())
}

func TestFactory_Ollama(t *testing.T) {
    client, err := NewLLMClient(Config{
        Provider: "ollama", Model: "llama3",
    })
    require.NoError(t, err)
    assert.Equal(t, "ollama", client.ProviderName())
}

func TestFactory_UnknownProvider(t *testing.T) {
    _, err := NewLLMClient(Config{Provider: "unknown"})
    require.Error(t, err)
    assert.Contains(t, err.Error(), "unknown")
}

func TestAvailableProviders(t *testing.T) {
    providers := AvailableProviders()
    assert.Contains(t, providers, "openai")
    assert.Contains(t, providers, "anthropic")
    assert.Contains(t, providers, "deepseek")
    assert.Contains(t, providers, "ollama")
    assert.Len(t, providers, 4)
}
