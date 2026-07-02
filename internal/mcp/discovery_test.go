package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/supcode/supcode/pkg"
)

func TestParseServerConfig_Basic(t *testing.T) {
	data := json.RawMessage(`{"name":"test","command":"echo","transport":"stdio"}`)
	cfg, err := ParseServerConfig(data)
	require.NoError(t, err)
	assert.Equal(t, "test", cfg.Name)
	assert.Equal(t, "echo", cfg.Command)
	assert.Equal(t, "stdio", cfg.Transport)
}

func TestParseServerConfig_EmptyName(t *testing.T) {
	data := json.RawMessage(`{"command":"echo"}`)
	_, err := ParseServerConfig(data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestParseServerConfig_DefaultTransport(t *testing.T) {
	data := json.RawMessage(`{"name":"test","command":"echo"}`)
	cfg, err := ParseServerConfig(data)
	require.NoError(t, err)
	assert.Equal(t, "stdio", cfg.Transport, "default transport should be stdio")
}

func TestParseServerConfig_WithEnv(t *testing.T) {
	data := json.RawMessage(`{"name":"test","command":"node","args":["server.js"],"env":{"KEY":"value"}}`)
	cfg, err := ParseServerConfig(data)
	require.NoError(t, err)
	assert.Equal(t, "node", cfg.Command)
	assert.Equal(t, []string{"server.js"}, cfg.Args)
	assert.Equal(t, "value", cfg.Env["KEY"])
}

func TestParseServerConfig_InvalidJSON(t *testing.T) {
	_, err := ParseServerConfig(json.RawMessage(`{"name":}`))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestParseServerConfig_SSETransport(t *testing.T) {
	data := json.RawMessage(`{"name":"test","command":"http://localhost:8080/sse","transport":"sse"}`)
	cfg, err := ParseServerConfig(data)
	require.NoError(t, err)
	assert.Equal(t, "sse", cfg.Transport)
}

func TestDiscoverFromDirectory(t *testing.T) {
	// Phase 2 MVP: returns empty
	cfgs, err := DiscoverFromDirectory("/nonexistent")
	require.NoError(t, err)
	assert.Empty(t, cfgs)
}

// suppress unused import
var _ = pkg.MCPServerConfig{}
