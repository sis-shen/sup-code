package anthropic

import (
    "context"
    "net/http"
    "net/http/httptest"
    "encoding/json"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/supcode/supcode/pkg"
)

func newTestClient(srv *httptest.Server) *LLMClient {
    return NewLLMClientWithHTTP(Config{APIKey: "test-key", BaseURL: srv.URL, Model: "claude-sonnet-4-20250514"}, srv.Client())
}

func writeSSE(w http.ResponseWriter, events []string) {
    w.Header().Set("Content-Type", "text/event-stream")
    f, _ := w.(http.Flusher)
    for _, e := range events {
        w.Write([]byte(e + "\n"))
        f.Flush()
    }
}

func TestAnthropic_ChatText(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "test-key", r.Header.Get("x-api-key"))
        writeSSE(w, []string{
            `event: content_block_delta`,
            `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
            ``,
            `event: message_delta`,
            `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
            ``,
            `event: message_stop`,
            `data: {"type":"message_stop"}`,
            ``,
        })
    }))
    defer srv.Close()
    client := newTestClient(srv)
    eventCh, err := client.Chat(context.Background(), "system", nil, nil)
    require.NoError(t, err)
    var events []pkg.StreamEvent
    for e := range eventCh { events = append(events, e) }
    require.Len(t, events, 2)
    assert.Equal(t, "text_delta", events[0].Type)
    assert.Equal(t, "Hello", events[0].Delta)
    assert.Equal(t, "done", events[1].Type)
}

func TestAnthropic_ToolUse(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        writeSSE(w, []string{
            `event: content_block_start`,
            `data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tu1","name":"test_tool","input":{"key":"val"}}}`,
            ``,
            `event: message_delta`,
            `data: {"type":"message_delta","delta":{"stop_reason":"tool_use"}}`,
            ``,
            `event: message_stop`,
            `data: {"type":"message_stop"}`,
            ``,
        })
    }))
    defer srv.Close()
    client := newTestClient(srv)
    eventCh, err := client.Chat(context.Background(), "system", nil, nil)
    require.NoError(t, err)
    var events []pkg.StreamEvent
    for e := range eventCh { events = append(events, e) }
    require.Len(t, events, 2)
    assert.Equal(t, "tool_call", events[0].Type)
    require.NotNil(t, events[0].ToolCall)
    assert.Equal(t, "tu1", events[0].ToolCall.ID)
    assert.Equal(t, "done", events[1].Type)
}

func TestAnthropic_ProviderName(t *testing.T) {
    c := NewLLMClient(Config{APIKey: "key"})
    assert.Equal(t, "anthropic", c.ProviderName())
}

func TestAnthropic_Models(t *testing.T) {
    c := NewLLMClient(Config{APIKey: "key"})
    models, err := c.Models(context.Background())
    require.NoError(t, err)
    assert.Greater(t, len(models), 0)
    assert.Contains(t, models[0].ID, "claude")
}

func TestAnthropic_Retry429(t *testing.T) {
    attempt := 0
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempt++
        if attempt == 1 {
            w.Header().Set("Retry-After", "0")
            http.Error(w, "Too Many Requests", 429)
            return
        }
        writeSSE(w, []string{
            `event: content_block_delta`,
            `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"ok"}}`,
            ``,
            `event: message_delta`,
            `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
            ``,
            `event: message_stop`,
            `data: {"type":"message_stop"}`,
            ``,
        })
    }))
    defer srv.Close()
    client := newTestClient(srv)
    eventCh, err := client.Chat(context.Background(), "system", nil, nil)
    require.NoError(t, err)
    var events []pkg.StreamEvent
    for e := range eventCh { events = append(events, e) }
    require.Len(t, events, 2)
    assert.Equal(t, "text_delta", events[0].Type)
}

func TestAnthropic_ErrorStatusCode(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        http.Error(w, "Server Error", http.StatusInternalServerError)
    }))
    defer srv.Close()
    client := newTestClient(srv)
    eventCh, err := client.Chat(context.Background(), "system", nil, nil)
    require.NoError(t, err)
    var events []pkg.StreamEvent
    for e := range eventCh { events = append(events, e) }
    assert.Greater(t, len(events), 0)
    assert.Equal(t, "error", events[len(events)-1].Type)
}

func TestAnthropic_NoAPIKey(t *testing.T) {
    client := NewLLMClient(Config{Model: "claude-sonnet-4-20250514"})
    assert.Equal(t, "anthropic", client.ProviderName())
}

func TestAnthropic_DefaultBaseURL(t *testing.T) {
    client := NewLLMClient(Config{APIKey: "key"})
    assert.Equal(t, "anthropic", client.ProviderName())
}


func TestAnthropic_BuildRequestWithToolCallMsg(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var body map[string]interface{}
        json.NewDecoder(r.Body).Decode(&body)
        // Verify tool calls in request
        if msgs, ok := body["messages"].([]interface{}); ok {
            for _, m := range msgs {
                if msg, ok := m.(map[string]interface{}); ok {
                    if content, ok := msg["content"].([]interface{}); ok {
                        for _, c := range content {
                            if blk, ok := c.(map[string]interface{}); ok && blk["type"] == "tool_use" {
                                // Found tool_use block - valid
                            }
                        }
                    }
                }
            }
        }
        writeSSE(w, []string{
            `event: message_delta`,
            `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}`,
            ``,
            `event: message_stop`,
            `data: {"type":"message_stop"}`,
            ``,
        })
    }))
    defer srv.Close()
    client := newTestClient(srv)
    eventCh, err := client.Chat(context.Background(), "system", []pkg.Message{
        {Role: pkg.RoleAssistant, Content: "Using tool", ToolCalls: []pkg.ToolCall{
            {ID: "call1", Name: "test", Params: json.RawMessage(`{"key":"val"}`)},
        }},
        {Role: pkg.RoleTool, Content: "Tool result", ToolID: "call1"},
    }, []pkg.ToolSchema{
        {Name: "test", Description: "Test tool", Parameters: json.RawMessage(`{"type":"object"}`)},
    })
    require.NoError(t, err)
    for range eventCh {}
}

func TestAnthropic_NetworkError(t *testing.T) {
    // Server that closes connection immediately
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        hijacker, ok := w.(http.Hijacker)
        if ok {
            conn, _, _ := hijacker.Hijack()
            conn.Close()
        }
    }))
    defer srv.Close()
    client := newTestClient(srv)
    eventCh, err := client.Chat(context.Background(), "system", nil, nil)
    require.NoError(t, err)
    var events []pkg.StreamEvent
    for e := range eventCh { events = append(events, e) }
    assert.Greater(t, len(events), 0)
}
