package llm

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/supcode/supcode/pkg"
)

type mockFailedClient struct{ name string }

func (m *mockFailedClient) Chat(ctx context.Context, sp string, msgs []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
    ch := make(chan pkg.StreamEvent, 1)
    ch <- pkg.StreamEvent{Type: "error", Error: "mock failure"}
    close(ch)
    return ch, nil
}
func (m *mockFailedClient) Models(ctx context.Context) ([]pkg.ModelInfo, error) { return nil, nil }
func (m *mockFailedClient) ProviderName() string { return m.name }

type mockSuccessClient struct{ name string }

func (m *mockSuccessClient) Chat(ctx context.Context, sp string, msgs []pkg.Message, tools []pkg.ToolSchema) (<-chan pkg.StreamEvent, error) {
    ch := make(chan pkg.StreamEvent, 2)
    ch <- pkg.StreamEvent{Type: "text_delta", Delta: "ok"}
    ch <- pkg.StreamEvent{Type: "done"}
    close(ch)
    return ch, nil
}
func (m *mockSuccessClient) Models(ctx context.Context) ([]pkg.ModelInfo, error) { return nil, nil }
func (m *mockSuccessClient) ProviderName() string { return m.name }

func TestFallback_ProviderName(t *testing.T) {
    fc := &FallbackClient{providers: []providerEntry{{name: "openai"}}}
    assert.Equal(t, "openai", fc.ProviderName())
}

func TestFallback_EmptyProviders(t *testing.T) {
    fc := &FallbackClient{}
    assert.Equal(t, "none", fc.ProviderName())
    _, err := fc.Models(context.Background())
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "no providers")
}

func TestFallback_Chat_Failover(t *testing.T) {
    fc := &FallbackClient{
        providers: []providerEntry{
            {name: "openai", client: &mockFailedClient{name: "openai"}},
            {name: "ollama", client: &mockSuccessClient{name: "ollama"}},
        },
    }
    eventCh, err := fc.Chat(context.Background(), "test", nil, nil)
    assert.NoError(t, err)
    assert.NotNil(t, eventCh)
    var events []pkg.StreamEvent
    for e := range eventCh { events = append(events, e) }
    assert.Greater(t, len(events), 0)
    assert.Equal(t, "text_delta", events[0].Type)
}
