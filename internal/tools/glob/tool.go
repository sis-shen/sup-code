package glob

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/supcode/supcode/pkg"
)

const defaultMaxResults = 10000

// GlobParams is the parsed parameters for Glob.
type GlobParams struct {
	Pattern    string `json:"pattern"`
	BasePath   string `json:"base_path,omitempty"`
	MaxResults *int   `json:"max_results,omitempty"`
}

var jsonSchema = json.RawMessage(`{
	"type": "object",
	"required": ["pattern"],
	"additionalProperties": false,
	"properties": {
		"pattern": {
			"type": "string",
			"description": "Glob pattern to match (supports doublestar **)"
		},
		"base_path": {
			"type": "string",
			"description": "Base directory to search from (default: current working directory)"
		},
		"max_results": {
			"type": "integer",
			"description": "Maximum number of results (default: 10000)",
			"minimum": 1,
			"maximum": 100000
		}
	}
}`)

// Tool implements pkg.Tool for file globbing with doublestar support.
type Tool struct{}

func (t *Tool) Name() string { return "glob" }

func (t *Tool) Description() string {
	return "Find files matching a glob pattern with doublestar (**) recursion support."
}

func (t *Tool) Schema() pkg.ToolSchema {
	return pkg.ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  jsonSchema,
	}
}

func (t *Tool) Execute(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
	var gp GlobParams
	if err := json.Unmarshal(params, &gp); err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	if gp.Pattern == "" {
		return pkg.ToolResult{
			Success: false,
			Error:   "pattern is required",
		}, nil
	}

	maxResults := defaultMaxResults
	if gp.MaxResults != nil && *gp.MaxResults > 0 {
		maxResults = *gp.MaxResults
	}

	basePath := gp.BasePath
	if basePath == "" {
		basePath = "."
	}

	select {
	case <-ctx.Done():
		return pkg.ToolResult{Success: false, Error: "canceled"}, ctx.Err()
	default:
	}

	matches, err := doublestarGlob(ctx, basePath, gp.Pattern, maxResults)
	if err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("glob failed: %v", err),
		}, nil
	}

	result := map[string]interface{}{
		"matches": matches,
		"count":   len(matches),
	}
	data, _ := json.Marshal(result)
	return pkg.ToolResult{
		Success: true,
		Data:    json.RawMessage(data),
	}, nil
}

// doublestarGlob performs glob matching with ** recursion support.
func doublestarGlob(ctx context.Context, basePath, pattern string, maxResults int) ([]string, error) {
	// If pattern contains **, we need to walk the filesystem manually
	if strings.Contains(pattern, "**") {
		return doublestarWalk(ctx, basePath, pattern, maxResults)
	}

	// Use standard filepath.Glob for simple patterns
	fullPattern := filepath.Join(basePath, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, err
	}
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}
	return matches, nil
}

// doublestarWalk does recursive file walking with ** support.
func doublestarWalk(ctx context.Context, basePath, pattern string, maxResults int) ([]string, error) {
	parts := splitPattern(pattern)
	var results []string

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if matchPattern(parts, path) {
			results = append(results, path)
			if len(results) >= maxResults {
				return filepath.SkipAll
			}
		}
		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, err
	}
	return results, nil
}

// splitPattern splits a pattern by / for segmented matching.
func splitPattern(pattern string) []string {
	parts := strings.Split(pattern, "/")
	// Filter out empty strings from leading/trailing slashes
	var filtered []string
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// matchPattern checks if a path matches the pattern parts.
// This is a simplified implementation supporting * and ** wildcards.
func matchPattern(parts []string, path string) bool {
	pathParts := strings.Split(filepath.ToSlash(path), "/")
	return segmentsMatch(parts, pathParts, 0, 0)
}

func segmentsMatch(pattern, text []string, pi, ti int) bool {
	// Consumed all pattern parts
	if pi >= len(pattern) {
		// If pattern is exhausted, text must also be exhausted
		return ti >= len(text)
	}

	// If text is exhausted, only ** can match (as zero segments)
	if ti >= len(text) {
		return allDoubleStar(pi, pattern)
	}

	switch pattern[pi] {
	case "**":
		// ** matches zero or more path segments
		// Try matching zero segments
		if segmentsMatch(pattern, text, pi+1, ti) {
			return true
		}
		// Try matching one or more segments
		if segmentsMatch(pattern, text, pi, ti+1) {
			return true
		}
		return false
	default:
		matched, _ := filepath.Match(pattern[pi], text[ti])
		if matched {
			return segmentsMatch(pattern, text, pi+1, ti+1)
		}
		return false
	}
}

func allDoubleStar(pi int, pattern []string) bool {
	for _, p := range pattern[pi:] {
		if p != "**" {
			return false
		}
	}
	return true
}
