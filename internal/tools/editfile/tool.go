package editfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/supcode/supcode/pkg"
)

// EditFileParams is the parsed parameters for EditFile.
type EditFileParams struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

var jsonSchema = json.RawMessage(`{
	"type": "object",
	"required": ["path", "old_string", "new_string"],
	"additionalProperties": false,
	"properties": {
		"path": {
			"type": "string",
			"description": "The path of the file to edit"
		},
		"old_string": {
			"type": "string",
			"description": "The exact string to replace (must appear exactly once)"
		},
		"new_string": {
			"type": "string",
			"description": "The string to replace with"
		}
	}
}`)

// Tool implements pkg.Tool for editing file contents via exact string replacement.
type Tool struct{}

func (t *Tool) Name() string { return "edit_file" }

func (t *Tool) Description() string {
	return "Edit a file by finding an exact string match and replacing it. old_string must appear exactly once in the file."
}

func (t *Tool) Schema() pkg.ToolSchema {
	return pkg.ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  jsonSchema,
	}
}

func (t *Tool) Execute(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
	var fp EditFileParams
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
	if fp.OldString == "" {
		return pkg.ToolResult{
			Success: false,
			Error:   "old_string is required",
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

	content := string(data)

	// Count occurrences
	count := strings.Count(content, fp.OldString)
	if count == 0 {
		return pkg.ToolResult{
			Success: false,
			Error:   "old_string not found in file",
		}, nil
	}
	if count > 1 {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("old_string appears %d times in file; expected exactly 1 match", count),
		}, nil
	}

	newContent := strings.Replace(content, fp.OldString, fp.NewString, 1)

	if err := os.WriteFile(fp.Path, []byte(newContent), 0644); err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("failed to write file: %v", err),
		}, nil
	}

	return pkg.ToolResult{
		Success: true,
		Data:    json.RawMessage(`{"status":"replaced"}`),
	}, nil
}
