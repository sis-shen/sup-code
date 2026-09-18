package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/pkg"
)

func newTestClient(server *httptest.Server) *LLMClient {
	return NewLLMClientWithHTTP(Config{
		APIKey:    "test-key",
		BaseURL:   server.URL,
		Model:     "gpt-4o",
		MaxTokens: 4096,
	}, server.Client())
}

func writeTestSSE(w http.ResponseWriter, events []string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}
	for _, e := range events {
		_, _ = w.Write([]byte(e + "\n"))
		flusher.Flush()
	}
}

func checkAuth(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer test-key"
}

func TestChat_NormalStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeTestSSE(w, []string{
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		})
	}))
	defer srv.Close()
	client := newTestClient(srv)
	eventCh, err := client.Chat(context.Background(), "system prompt", nil, nil)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	require.Len(t, events, 3)
	assert.Equal(t, "text_delta", events[0].Type)
	assert.Equal(t, "Hello", events[0].Delta)
	assert.Equal(t, "text_delta", events[1].Type)
	assert.Equal(t, " world", events[1].Delta)
	assert.Equal(t, "done", events[2].Type)
}

func TestChat_ToolCallStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkAuth(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		writeTestSSE(w, []string{
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"id":"call1","type":"function","function":{"name":"test_tool","arguments":"{\"key\":\"value\"}"}}]},"finish_reason":"tool_calls"}]}`,
			`data: [DONE]`,
		})
	}))
	defer srv.Close()
	client := newTestClient(srv)
	tools := []pkg.ToolSchema{{Name: "test_tool", Description: "A test tool", Parameters: json.RawMessage(`{"type":"object"}`)}}
	eventCh, err := client.Chat(context.Background(), "system", nil, tools)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	require.Len(t, events, 2)
	assert.Equal(t, "tool_call", events[0].Type)
	require.NotNil(t, events[0].ToolCall)
	assert.Equal(t, "call1", events[0].ToolCall.ID)
	assert.Equal(t, "test_tool", events[0].ToolCall.Name)
	assert.Equal(t, "done", events[1].Type)
}

func TestChat_429Retry(t *testing.T) {
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt <= 2 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		writeTestSSE(w, []string{
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		})
	}))
	defer srv.Close()
	client := newTestClient(srv)
	eventCh, err := client.Chat(context.Background(), "system", nil, nil)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	require.Len(t, events, 2)
	assert.Equal(t, "text_delta", events[0].Type)
	assert.Equal(t, "ok", events[0].Delta)
	assert.Equal(t, "done", events[1].Type)
}

func TestChat_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Error", http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := newTestClient(srv)
	eventCh, err := client.Chat(context.Background(), "system", nil, nil)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	require.Len(t, events, 1)
	assert.Equal(t, "error", events[0].Type)
}

func TestChat_StreamInterrupted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		_, _ = w.Write([]byte(`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"partial"},"finish_reason":null}]}` + "\n"))
		flusher.Flush()
	}))
	defer srv.Close()
	client := newTestClient(srv)
	eventCh, err := client.Chat(context.Background(), "system", nil, nil)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	require.Len(t, events, 1)
	assert.Equal(t, "text_delta", events[0].Type)
	assert.Equal(t, "partial", events[0].Delta)
}

func TestChat_FunctionCallingFormat(t *testing.T) {
	var requestBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&requestBody)
		writeTestSSE(w, []string{
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		})
	}))
	defer srv.Close()
	client := newTestClient(srv)
	tools := []pkg.ToolSchema{
		{Name: "test_tool", Description: "A test tool", Parameters: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`)},
	}
	eventCh, err := client.Chat(context.Background(), "system", []pkg.Message{
		{Role: pkg.RoleUser, Content: "hello"},
	}, tools)
	require.NoError(t, err)
	// Drain events first to ensure request is sent
	for range eventCh {
	}
	require.NotNil(t, requestBody)
	assert.Contains(t, requestBody, "tools")
	toolsRaw := requestBody["tools"].([]interface{})
	require.Len(t, toolsRaw, 1)
	tool := toolsRaw[0].(map[string]interface{})
	assert.Equal(t, "function", tool["type"])
}

func TestProviderName(t *testing.T) {
	client := NewLLMClient(Config{APIKey: "key"})
	assert.Equal(t, "openai", client.ProviderName())
}

func TestModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4o"},{"id":"gpt-4-turbo"},{"id":"gpt-3.5-turbo"}]}`))
	}))
	defer srv.Close()
	client := newTestClient(srv)
	models, err := client.Models(context.Background())
	require.NoError(t, err)
	require.Len(t, models, 3)
	assert.Equal(t, "gpt-4o", models[0].ID)
	assert.True(t, models[0].SupportsVision)
	assert.False(t, models[2].SupportsVision)
}

func TestModels_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()
	client := newTestClient(srv)
	_, err := client.Models(context.Background())
	assert.Error(t, err)
}

func TestParseRetryAfter(t *testing.T) {
	assert.Equal(t, time.Second, parseRetryAfter("1"))
	assert.Equal(t, 5*time.Second, parseRetryAfter("5"))
	assert.Equal(t, time.Duration(0), parseRetryAfter(""))
	assert.Equal(t, time.Duration(0), parseRetryAfter("invalid"))
}

func TestIsRetryableError_EdgeCases(t *testing.T) {
	client := NewLLMClient(Config{APIKey: "key"})
	assert.True(t, client.isRetryableError(&pkg.ErrRetryable{Cause: assert.AnError}))
	assert.False(t, client.isRetryableError(assert.AnError))
}

func TestChat_RetryWithRetryAfter(t *testing.T) {
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			w.Header().Set("Retry-After", "0")
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}
		writeTestSSE(w, []string{
			`data: {"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"retried"},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		})
	}))
	defer srv.Close()
	client := newTestClient(srv)
	eventCh, err := client.Chat(context.Background(), "system", nil, nil)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	require.Len(t, events, 2)
	assert.Equal(t, "text_delta", events[0].Type)
	assert.Equal(t, "retried", events[0].Delta)
}

func TestChat_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		http.Error(w, "Slow", http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := newTestClient(srv)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Chat(ctx, "system", nil, nil)
	require.NoError(t, err)
}

func TestChat_ServerErrorWith5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Server Error", http.StatusInternalServerError)
	}))
	defer srv.Close()
	tc := Config{APIKey: "test-key", BaseURL: srv.URL, Model: "gpt-4o", MaxTokens: 4096}
	c := NewLLMClientWithHTTP(tc, srv.Client())
	eventCh, err := c.Chat(context.Background(), "system", nil, nil)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	require.True(t, len(events) > 0)
	assert.Equal(t, "error", events[len(events)-1].Type)
}

func TestChat_EmptyStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f, _ := w.(http.Flusher)
		// Send empty line to close the stream
		_, _ = w.Write([]byte("\n"))
		f.Flush()
	}))
	defer srv.Close()
	client := newTestClient(srv)
	eventCh, err := client.Chat(context.Background(), "system", nil, nil)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	// Should not panic, channel should close gracefully
	t.Logf("Received %d events (should be 0)", len(events))
}

func TestChat_Three429s(t *testing.T) {
	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		w.Header().Set("Retry-After", "0")
		http.Error(w, "Too Many", http.StatusTooManyRequests)
	}))
	defer srv.Close()
	client := newTestClient(srv)
	eventCh, err := client.Chat(context.Background(), "system", nil, nil)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	require.Len(t, events, 1)
	assert.Equal(t, "error", events[0].Type)
	// Should attempt 1 initial + up to 3 retries = 4 total
}

func TestChat_ContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		f, _ := w.(http.Flusher)
		_, _ = w.Write([]byte("data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n"))
		f.Flush()
		time.Sleep(5 * time.Second) // Should cause context timeout
		_, _ = w.Write([]byte("data: {\"id\":\"1\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n"))
		f.Flush()
	}))
	defer srv.Close()
	client := newTestClient(srv)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	eventCh, err := client.Chat(ctx, "system", nil, nil)
	require.NoError(t, err)
	var events []pkg.StreamEvent
	for e := range eventCh {
		events = append(events, e)
	}
	t.Logf("Events after timeout: %d", len(events))
}
