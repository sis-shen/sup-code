package skilltool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supcode/supcode/internal/skill"
	"github.com/supcode/supcode/pkg"
)

const toolName = "skill"

var schemaParams = json.RawMessage(`{
  "type": "object",
  "properties": {
    "action": {"type": "string", "enum": ["list", "load"], "description": "list: 列出所有可用技能；load: 加载指定技能的专家指导"},
    "name":   {"type": "string", "description": "action=load 时必填，技能名称"}
  },
  "required": ["action"]
}`)

// Tool 把技能库适配为标准 pkg.Tool，使其和内置工具、MCP 工具走同一个 Registry。
type Tool struct {
	loader *skill.SkillLoader
}

// New 创建一个 Skill 工具。
func New(loader *skill.SkillLoader) *Tool {
	return &Tool{loader: loader}
}

// Name 返回工具名。
func (t *Tool) Name() string { return toolName }

// Description 返回工具描述。
func (t *Tool) Description() string {
	return "发现并加载可复用的专家技能指导（skill）。action=list 查看当前可用技能列表；" +
		"action=load 配合 name 参数加载指定技能的详细操作指南，加载后请遵循其中的指导完成任务。"
}

// Schema 返回参数 JSON Schema。
func (t *Tool) Schema() pkg.ToolSchema {
	return pkg.ToolSchema{Name: toolName, Description: t.Description(), Parameters: schemaParams}
}

type params struct {
	Action string `json:"action"`
	Name   string `json:"name"`
}

// Execute 执行 skill 工具。
func (t *Tool) Execute(ctx context.Context, raw json.RawMessage) (pkg.ToolResult, error) {
	select {
	case <-ctx.Done():
		return pkg.ToolResult{Success: false, Error: ctx.Err().Error()}, ctx.Err()
	default:
	}

	var p params
	if err := json.Unmarshal(raw, &p); err != nil {
		return pkg.ToolResult{Success: false, Error: fmt.Sprintf("invalid params: %v", err)}, err
	}

	switch p.Action {
	case "list":
		infos, err := t.loader.Discover()
		if err != nil {
			return pkg.ToolResult{Success: false, Error: err.Error()}, err
		}
		data, _ := json.Marshal(infos)
		return pkg.ToolResult{Success: true, Data: data}, nil

	case "load":
		if p.Name == "" {
			err := fmt.Errorf("name is required when action=load")
			return pkg.ToolResult{Success: false, Error: err.Error()}, err
		}
		s, err := t.loader.Load(p.Name)
		if err != nil {
			return pkg.ToolResult{Success: false, Error: err.Error()}, err
		}
		payload := struct {
			Name         string   `json:"name"`
			Version      string   `json:"version"`
			Description  string   `json:"description"`
			Tools        []string `json:"recommended_tools,omitempty"`
			Instructions string   `json:"instructions"`
		}{
			Name:         s.Manifest.Name,
			Version:      s.Manifest.Version,
			Description:  s.Manifest.Description,
			Tools:        s.Manifest.Tools,
			Instructions: s.SystemPrompt,
		}
		data, _ := json.Marshal(payload)
		return pkg.ToolResult{Success: true, Data: data}, nil

	default:
		err := fmt.Errorf("unknown action: %s (expected list|load)", p.Action)
		return pkg.ToolResult{Success: false, Error: err.Error()}, err
	}
}

var _ pkg.Tool = (*Tool)(nil)
