package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSETransport_New(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport, err := newSSETransport(context.Background(), server.URL)
	require.NoError(t, err)
	require.NotNil(t, transport)
	defer func() { _ = transport.Close() }()
}

func TestSSETransport_SendAfterClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport, err := newSSETransport(context.Background(), server.URL)
	require.NoError(t, err)

	require.NoError(t, transport.Close())
	_, err = transport.Send(context.Background(), jsonRPCRequest{Method: "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestSSETransport_HandlesInvalidURL(t *testing.T) {
	_, err := newSSETransport(context.Background(), "http://127.0.0.1:1")
	require.Error(t, err)
}

func TestSSETransport_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := newSSETransport(context.Background(), server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestSSEParseEvent(t *testing.T) {
	s := &sseTransport{
		pending: make(map[int]chan jsonRPCResponse),
	}
	s.handleSSEEvent("message", `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05"}}`)
}

func TestSSEEventParser(t *testing.T) {
	s := &sseTransport{
		pending: make(map[int]chan jsonRPCResponse),
	}

	respCh := make(chan jsonRPCResponse, 1)
	s.pending[5] = respCh
	s.handleSSEEvent("message", `{"jsonrpc":"2.0","id":5,"result":{"tools":[]}}`)

	resp := <-respCh
	assert.NotNil(t, resp.Result)
	assert.Contains(t, string(resp.Result), "tools")
}

var _ = ""
