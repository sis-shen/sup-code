package writefile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/supcode/supcode/pkg"
)

// WriteFileParams is the parsed parameters for WriteFile.
type WriteFileParams struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

var jsonSchema = json.RawMessage(`{
	"type": "object",
	"required": ["path", "content"],
	"additionalProperties": false,
	"properties": {
		"path": {
			"type": "string",
			"description": "The path of the file to write"
		},
		"content": {
			"type": "string",
			"description": "The content to write to the file"
		}
	}
}`)

// Tool implements pkg.Tool for writing file contents.
type Tool struct{}

func (t *Tool) Name() string { return "write_file" }

func (t *Tool) Description() string {
	return "Write content to a file, creating parent directories if needed."
}

func (t *Tool) Schema() pkg.ToolSchema {
	return pkg.ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  jsonSchema,
	}
}

func (t *Tool) Execute(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
	var fp WriteFileParams
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

	// Create parent directories
	dir := filepath.Dir(fp.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return pkg.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("failed to create directories: %v", err),
			}, nil
		}
	}

	if err := os.WriteFile(fp.Path, []byte(fp.Content), 0644); err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, nil
	}

	return pkg.ToolResult{
		Success: true,
		Data:    json.RawMessage(fmt.Sprintf(`{"path":"%s","size":%d}`, fp.Path, len(fp.Content))),
	}, nil
}
