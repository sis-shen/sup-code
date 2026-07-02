package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/supcode/supcode/pkg"
)

// MCPServerConfigFile is the file format for MCP server configurations.
type MCPServerConfigFile struct {
	Servers []pkg.MCPServerConfig `json:"servers"`
}

// DiscoverFromDirectory is a placeholder for Phase 3 service discovery.
// Phase 2 MVP: return empty list without scanning.
// Phase 3: scan ~/.supcode/mcp/*.json and parse MCPServerConfig files.
func DiscoverFromDirectory(dir string) ([]pkg.MCPServerConfig, error) {
	// Phase 2 MVP: no-op, return empty
	return nil, nil
}

// ParseServerConfig parses a json.RawMessage into a MCPServerConfig.
func ParseServerConfig(data json.RawMessage) (pkg.MCPServerConfig, error) {
	var cfg pkg.MCPServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse server config: %w", err)
	}
	if cfg.Name == "" {
		return cfg, fmt.Errorf("server name is required")
	}
	if cfg.Transport == "" {
		cfg.Transport = "stdio"
	}
	return cfg, nil
}

// compile-time check for unused imports
var _ = fmt.Sprintf
