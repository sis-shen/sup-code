package bash

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/supcode/supcode/pkg"
)

const (
	maxOutputSize  = 100 * 1024 // 100KB
	defaultTimeout = 30 * time.Second
)

// BashParams is the parsed parameters for Bash.
type BashParams struct {
	Command string  `json:"command"`
	Timeout *int    `json:"timeout,omitempty"` // in seconds
	WorkDir *string `json:"workdir,omitempty"`
}

var jsonSchema = json.RawMessage(`{
	"type": "object",
	"required": ["command"],
	"additionalProperties": false,
	"properties": {
		"command": {
			"type": "string",
			"description": "The shell command to execute"
		},
		"timeout": {
			"type": "integer",
			"description": "Timeout in seconds (default: 30)",
			"minimum": 1,
			"maximum": 300
		},
		"workdir": {
			"type": "string",
			"description": "Working directory for the command"
		}
	}
}`)

// Tool implements pkg.Tool for executing bash commands.
type Tool struct{}

func (t *Tool) Name() string { return "bash" }

func (t *Tool) Description() string {
	return fmt.Sprintf("Execute a shell command (%s) with timeout and output truncation.", getShell())
}

func (t *Tool) Schema() pkg.ToolSchema {
	return pkg.ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  jsonSchema,
	}
}

// set by tests for mockability
var execCommand = exec.CommandContext

func (t *Tool) Execute(ctx context.Context, params json.RawMessage) (pkg.ToolResult, error) {
	var fp BashParams
	if err := json.Unmarshal(params, &fp); err != nil {
		return pkg.ToolResult{
			Success: false,
			Error:   fmt.Sprintf("invalid parameters: %v", err),
		}, nil
	}

	if fp.Command == "" {
		return pkg.ToolResult{
			Success: false,
			Error:   "command is required",
		}, nil
	}

	timeout := defaultTimeout
	if fp.Timeout != nil && *fp.Timeout > 0 {
		timeout = time.Duration(*fp.Timeout) * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return pkg.ToolResult{Success: false, Error: "canceled"}, ctx.Err()
	default:
	}

	cmd := execCommand(execCtx, getShell(), getShellFlag(), fp.Command)
	if fp.WorkDir != nil && *fp.WorkDir != "" {
		cmd.Dir = *fp.WorkDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	out := stdout.Bytes()
	errOut := stderr.Bytes()

	truncated := false
	if len(out) > maxOutputSize {
		out = out[:maxOutputSize]
		truncated = true
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command didn't start (executable not found, etc.)
			return pkg.ToolResult{
				Success: false,
				Error:   fmt.Sprintf("command execution failed: %v", err),
			}, nil
		}
	}

	result := map[string]interface{}{
		"stdout":    string(out),
		"stderr":    string(errOut),
		"exit_code": exitCode,
	}
	if truncated {
		result["truncated"] = true
	}

	data, _ := json.Marshal(result)
	return pkg.ToolResult{
		Success: exitCode == 0,
		Data:    json.RawMessage(data),
	}, nil
}

func getShell() string {
	if runtime.GOOS == "windows" {
		return "powershell.exe"
	}
	return "bash"
}

func getShellFlag() string {
	if runtime.GOOS == "windows" {
		return "-Command"
	}
	return "-c"
}
