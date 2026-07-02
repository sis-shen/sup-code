package deepseek

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/supcode/supcode/pkg"
)

func newTestClient(srv *httptest.Server) *LLMClient {
    return NewLLMClientWithHTTP(Config{APIKey: "sk-test", BaseURL: srv.URL, Model: "deepseek-chat", MaxTokens: 4096}, srv.Client())
}

func TestDeepSeek_Chat(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/event-stream")
        f, _ := w.(http.Flusher)
        w.Write([]byte("data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n"))
        f.Flush()
        w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n"))
        f.Flush()
        w.Write([]byte("data: [DONE]\n"))
        f.Flush()
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

func TestDeepSeek_EmptyResponse(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/event-stream")
        f, _ := w.(http.Flusher)
        w.Write([]byte("data: \n"))
        f.Flush()
        w.Write([]byte("data: [DONE]\n"))
        f.Flush()
    }))
    defer srv.Close()
    client := newTestClient(srv)
    eventCh, err := client.Chat(context.Background(), "system", nil, nil)
    require.NoError(t, err)
    var events []pkg.StreamEvent
    for e := range eventCh { events = append(events, e) }
    require.Len(t, events, 1)
    assert.Equal(t, "done", events[0].Type)
}

func TestDeepSeek_ProviderName(t *testing.T) {
    c := NewLLMClient(Config{APIKey: "key"})
    assert.Equal(t, "deepseek", c.ProviderName())
}

func TestDeepSeek_Models(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"data":[{"id":"deepseek-chat"},{"id":"deepseek-reasoner"}]}`))
    }))
    defer srv.Close()
    client := newTestClient(srv)
    models, err := client.Models(context.Background())
    require.NoError(t, err)
    assert.Greater(t, len(models), 0)
}

func TestDeepSeek_Retry429(t *testing.T) {
    attempt := 0
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        attempt++
        if attempt == 1 { http.Error(w, "Too Many", 429); return }
        w.Header().Set("Content-Type", "text/event-stream")
        f, _ := w.(http.Flusher)
        w.Write([]byte("data: [DONE]\n")); f.Flush()
    }))
    defer srv.Close()
    client := newTestClient(srv)
    eventCh, err := client.Chat(context.Background(), "system", nil, nil)
    require.NoError(t, err)
    var events []pkg.StreamEvent
    for e := range eventCh { events = append(events, e) }
    require.Len(t, events, 1)
    assert.Equal(t, "done", events[0].Type)
}
