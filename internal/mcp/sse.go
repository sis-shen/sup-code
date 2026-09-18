package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

// sseTransport implements Transport via Server-Sent Events.
type sseTransport struct {
	client  *http.Client
	baseURL string

	mu      sync.Mutex
	reqID   atomic.Int32
	pending map[int]chan jsonRPCResponse
	closed  bool

	sessionID string
	endpoint  string // POST endpoint for sending messages
}

// newSSETransport connects to an MCP server via SSE.
func newSSETransport(ctx context.Context, baseURL string) (*sseTransport, error) {
	t := &sseTransport{
		client:  http.DefaultClient,
		baseURL: baseURL,
		pending: make(map[int]chan jsonRPCResponse),
	}

	if err := t.connectSSE(ctx); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *sseTransport) connectSSE(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	if t.sessionID != "" {
		req.Header.Set("Last-Event-ID", t.sessionID)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return fmt.Errorf("SSE connect: HTTP %d", resp.StatusCode)
	}

	go t.readSSE(resp.Body)
	return nil
}

func (t *sseTransport) readSSE(body io.ReadCloser) {
	defer func() { _ = body.Close() }()
	scanner := bufio.NewScanner(body)

	var eventType string
	var dataBuf bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line = end of event
			if dataBuf.Len() > 0 {
				t.handleSSEEvent(eventType, dataBuf.String())
			}
			eventType = ""
			dataBuf.Reset()
			continue
		}

		if len(line) > 0 && line[0] == ':' {
			// Comment line
			continue
		}

		if colonIdx := bytes.IndexByte([]byte(line), ':'); colonIdx >= 0 {
			field := line[:colonIdx]
			value := ""
			if colonIdx+1 < len(line) {
				value = line[colonIdx+1:]
				if len(value) > 0 && value[0] == ' ' {
					value = value[1:]
				}
			}
			switch field {
			case "event":
				eventType = value
			case "data":
				if dataBuf.Len() > 0 {
					dataBuf.WriteByte('\n')
				}
				dataBuf.WriteString(value)
			case "id":
				t.sessionID = value
			}
		}
	}
}

func (t *sseTransport) handleSSEEvent(eventType string, data string) {
	if eventType != "message" {
		return
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return
	}

	t.mu.Lock()
	ch, ok := t.pending[resp.ID]
	if ok {
		delete(t.pending, resp.ID)
	}
	t.mu.Unlock()

	if ok && ch != nil {
		ch <- resp
	}
}

func (t *sseTransport) Send(ctx context.Context, req jsonRPCRequest) (jsonRPCResponse, error) {
	req.JSONRPC = jsonRPCVersion
	if req.ID == 0 {
		req.ID = int(t.reqID.Add(1))
	}

	data, err := json.Marshal(req)
	if err != nil {
		return jsonRPCResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	// POST to the endpoint (or base URL if no dedicated endpoint)
	postURL := t.baseURL
	if t.endpoint != "" {
		postURL = t.endpoint
	}

	respCh := make(chan jsonRPCResponse, 1)
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return jsonRPCResponse{}, fmt.Errorf("transport closed")
	}
	t.pending[req.ID] = respCh
	t.mu.Unlock()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, bytes.NewReader(data))
	if err != nil {
		return jsonRPCResponse{}, fmt.Errorf("create POST request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return jsonRPCResponse{}, fmt.Errorf("POST request: %w", err)
	}
	_ = httpResp.Body.Close()

	select {
	case resp := <-respCh:
		if resp.Error != nil {
			return resp, resp.Error
		}
		return resp, nil
	case <-ctx.Done():
		return jsonRPCResponse{}, ctx.Err()
	}
}

func (t *sseTransport) Close() error {
	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()
	return nil
}

// compile-time check
var _ Transport = (*sseTransport)(nil)
