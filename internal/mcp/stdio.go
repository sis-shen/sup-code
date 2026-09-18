package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/supcode/supcode/pkg"
)

// stdioTransport implements Transport via a child process stdin/stdout.
type stdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	mu      sync.Mutex
	reqID   atomic.Int32
	pending map[int]chan jsonRPCResponse
	closed  bool

	retries    int
	maxRetries int
	config     pkg.MCPServerConfig
}

// newStdioTransport starts a new stdio transport.
// maxRetries: maximum number of auto-restart attempts (0 = no restart).
func newStdioTransport(ctx context.Context, config pkg.MCPServerConfig, maxRetries int) (*stdioTransport, error) {
	t := &stdioTransport{
		pending:    make(map[int]chan jsonRPCResponse),
		maxRetries: maxRetries,
		config:     config,
	}

	if err := t.startProcess(ctx); err != nil {
		return nil, err
	}

	go t.readLoop()
	return t, nil
}

func (t *stdioTransport) startProcess(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, t.config.Command, t.config.Args...)
	if t.config.Env != nil {
		env := make([]string, 0, len(t.config.Env))
		for k, v := range t.config.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = append(cmd.Env, env...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start process: %w", err)
	}

	t.mu.Lock()
	t.cmd = cmd
	t.stdin = stdin
	t.stdout = stdout
	t.stderr = stderr
	t.mu.Unlock()
	return nil
}

func (t *stdioTransport) readLoop() {
	scanner := bufio.NewScanner(t.stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var resp jsonRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
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

	// Process exited or connection closed
	t.mu.Lock()
	closed := t.closed
	t.mu.Unlock()

	if !closed {
		// Attempt restart
		t.tryRestart()
	}
}

func (t *stdioTransport) tryRestart() {
	t.mu.Lock()
	if t.retries >= t.maxRetries {
		t.mu.Unlock()
		// Notify all pending requests of failure
		t.failPending(fmt.Errorf("process crashed after %d retries", t.maxRetries))
		return
	}
	t.retries++
	t.mu.Unlock()

	ctx := context.Background()
	if err := t.startProcess(ctx); err != nil {
		t.failPending(fmt.Errorf("restart failed: %w", err))
		return
	}

	go t.readLoop()
}

func (t *stdioTransport) failPending(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, ch := range t.pending {
		ch <- jsonRPCResponse{ID: id, Error: &jsonRPCError{Code: -32000, Message: err.Error()}}
		delete(t.pending, id)
	}
}

func (t *stdioTransport) Send(ctx context.Context, req jsonRPCRequest) (jsonRPCResponse, error) {
	req.JSONRPC = jsonRPCVersion
	if req.ID == 0 {
		req.ID = int(t.reqID.Add(1))
	}

	data, err := json.Marshal(req)
	if err != nil {
		return jsonRPCResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	respCh := make(chan jsonRPCResponse, 1)
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return jsonRPCResponse{}, fmt.Errorf("transport closed")
	}
	t.pending[req.ID] = respCh
	stdin := t.stdin
	t.mu.Unlock()

	if stdin == nil {
		return jsonRPCResponse{}, fmt.Errorf("transport not started")
	}
	if _, err := stdin.Write(append(data, '\n')); err != nil {
		return jsonRPCResponse{}, fmt.Errorf("write stdin: %w", err)
	}

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

func (t *stdioTransport) Close() error {
	t.mu.Lock()
	t.closed = true
	cmd := t.cmd
	t.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}

// compile-time check
var _ Transport = (*stdioTransport)(nil)
