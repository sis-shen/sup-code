package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/supcode/supcode/pkg"
)

// mockMCPServer simulates a real MCP server over SSE for testing.
type mockMCPServer struct {
	t            *testing.T
	server       *httptest.Server
	pending      chan []byte
	tools        []pkg.ToolSchema
	protocolVer  string
	mu           sync.Mutex
	receivedReqs []jsonRPCRequest
}

// NewMockMCPServer creates a mock MCP server.
// tools: tools to return from tools/list
// protocolVersion: version string to return from initialize
func NewMockMCPServer(t *testing.T, tools []pkg.ToolSchema, protocolVersion string) *mockMCPServer {
	m := &mockMCPServer{
		t:            t,
		pending:      make(chan []byte, 100),
		tools:        tools,
		protocolVer:  protocolVersion,
		receivedReqs: make([]jsonRPCRequest, 0),
	}

	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("mock MCP server: %s %s", r.Method, r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			m.handleSSE(w, r)
		case r.Method == http.MethodPost && (r.URL.Path == "/message" || r.URL.Path == "/"):
			m.handleMessage(w, r)
		default:
			http.NotFound(w, r)
		}
	}))

	t.Cleanup(m.server.Close)
	return m
}

// URL returns the base URL of the mock server.
func (m *mockMCPServer) URL() string { return m.server.URL }

// MessageURL returns the POST endpoint URL.
func (m *mockMCPServer) MessageURL() string { return m.server.URL + "/message" }

// ReceivedRequests returns all JSON-RPC requests received by the server.
func (m *mockMCPServer) ReceivedRequests() []jsonRPCRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]jsonRPCRequest, len(m.receivedReqs))
	copy(cp, m.receivedReqs)
	return cp
}

func (m *mockMCPServer) handleSSE(w http.ResponseWriter, _ *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Send endpoint event
	_, _ = fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", m.MessageURL())
	flusher.Flush()

	// Send responses as they come in
	for resp := range m.pending {
		_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(resp))
		flusher.Flush()
	}
}

func (m *mockMCPServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	defer func() { _ = r.Body.Close() }()

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	m.mu.Lock()
	m.receivedReqs = append(m.receivedReqs, req)
	m.mu.Unlock()

	// Log the request
	m.t.Logf("mock MCP received: method=%s id=%d", req.Method, req.ID)

	var resp jsonRPCResponse
	resp.JSONRPC = jsonRPCVersion
	resp.ID = req.ID

	switch req.Method {
	case "initialize":
		resp.Result = mustMarshalJSON(map[string]interface{}{
			"protocolVersion": m.protocolVer,
		})
	case "tools/list":
		toolsList := m.tools
		if toolsList == nil {
			toolsList = []pkg.ToolSchema{}
		}
		resp.Result = mustMarshalJSON(map[string]interface{}{
			"tools": toolsList,
		})
	case "tools/call":
		content := []map[string]interface{}{
			{"type": "text", "text": "mock result"},
		}
		resp.Result = mustMarshalJSON(map[string]interface{}{
			"content": content,
			"isError": false,
		})
	case "notifications/initialized":
		// Notifications have no response
		w.WriteHeader(http.StatusAccepted)
		return
	default:
		resp.Error = &jsonRPCError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}

	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	m.pending <- data
	w.WriteHeader(http.StatusAccepted)
}

// Stop closes the SSE channel and waits for cleanup.
func (m *mockMCPServer) Stop() {
	close(m.pending)
}

func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
