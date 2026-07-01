package readfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/supcode/supcode/pkg"
)

// ReadFileParams is the parsed parameters for ReadFile.
type ReadFileParams struct {
	Path     string `json:"path"`
	StartLine *int  `json:"start_line,omitempty"`
	EndLine   *int  `json:"end_line,omitempty"`
}

var jsonSchema = json.RawMessage(`{
	"type": "object",
	"required": ["path"],
	"additionalProperties": false,
	"properties": {
		"path": {
			"type": "string",
			"description": "The path of the file to read"
		},
		"start_line": {
			"type": "integer",
			"description": "Optional 1-based start line for partial read",
			"minimum": 1
		},
		"end_line": {
			"type": "integer",
			"description": "Optional 1-based end line for partial read (inclusive)",
			"minimum": 1
		}
	}
}`)

// Tool implements pkg.Tool for reading file contents.
type Tool struct{}

func (t *Tool) Name() string { return "read_file" }

func (t *Tool) Description() string {
	return "Read the contents of a file, with optional line range filtering."
}

func (t *Tool) Schema() pkg.ToolSchema {
	return pkg.ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  jsonSchema,
	}
}

func (t *Tool) Execute(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
	var fp ReadFileParams
	if err := json.Unmarshal(params, &fp); err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	if fp.Path == "" {
		return pkg.ToolResult{
			Success: false,
			Error:   "path is required",
		}, nil
	}

	select {
	case <-ctx.Done():
		return pkg.ToolResult{Success: false, Error: "canceled"}, ctx.Err()
	default:
	}

	data, err := os.ReadFile(fp.Path)
	if err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to read file: %v", err),
		}, nil
	}

	if fp.StartLine != nil || fp.EndLine != nil {
		lines := strings.Split(string(data), "\n")
		start := 0
		if fp.StartLine != nil && *fp.StartLine > 0 {
			start = *fp.StartLine - 1
		}
		end := len(lines)
		if fp.EndLine != nil && *fp.EndLine <= len(lines) {
			end = *fp.EndLine
		}
	if start > end && fp.StartLine != nil && fp.EndLine != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("start_line %d is after end_line %d", *fp.StartLine, *fp.EndLine),
		}, nil
	}
	if start >= len(lines) {
			return pkg.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("start_line %d exceeds file length (%d lines)", *fp.StartLine, len(lines)),
			}, nil
		}
	result := strings.Join(lines[start:end], "\n")
		data = []byte(result)
	}

	return pkg.ToolResult{
		Success: true,
		Data:    json.RawMessage(fmt.Sprintf(`{"content":%s}`, mustMarshal(string(data)))),
	}, nil
}

func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
