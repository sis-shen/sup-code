package ollama

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
    return NewLLMClientWithHTTP(Config{BaseURL: srv.URL, Model: "llama3"}, srv.Client())
}

func TestOllama_Chat(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "", r.Header.Get("Authorization"))
        w.Header().Set("Content-Type", "text/event-stream")
        f, _ := w.(http.Flusher)
        w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n")); f.Flush()
        w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n")); f.Flush()
        w.Write([]byte("data: [DONE]\n")); f.Flush()
    }))
    defer srv.Close()
    client := newTestClient(srv)
    eventCh, err := client.Chat(context.Background(), "system", nil, nil)
    require.NoError(t, err)
    var events []pkg.StreamEvent
    for e := range eventCh { events = append(events, e) }
    require.Len(t, events, 2)
    assert.Equal(t, "text_delta", events[0].Type)
    assert.Equal(t, "done", events[1].Type)
}

func TestOllama_ProviderName(t *testing.T) {
    c := NewLLMClient(Config{})
    assert.Equal(t, "ollama", c.ProviderName())
}

func TestOllama_Models(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"models":[{"name":"llama3:latest"},{"name":"mistral:latest"}]}`))
    }))
    defer srv.Close()
    client := newTestClient(srv)
    models, err := client.Models(context.Background())
    require.NoError(t, err)
    assert.Greater(t, len(models), 0)
}
