package skilltool

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/supcode/supcode/internal/skill"
	"github.com/supcode/supcode/pkg"
)

// makeLoader 在临时目录构造一个包含单个技能的 SkillLoader。
func makeLoader(t *testing.T, version, description, instructions string) *skill.SkillLoader {
	t.Helper()
	const name = "code-review"
	dir := t.TempDir()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Join(skillDir, "prompts"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"name":        name,
		"version":     version,
		"description": description,
		"tools":       []string{"bash", "read_file"},
	}
	data, _ := json.Marshal(manifest)
	if err := os.WriteFile(filepath.Join(skillDir, "skill.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "prompts", "system.md"), []byte(instructions), 0644); err != nil {
		t.Fatal(err)
	}
	return skill.NewSkillLoader(dir, "", "")
}

func TestTool_Schema_And_Name(t *testing.T) {
	tool := New(makeLoader(t, "1.0.0", "desc", "do it"))
	if tool.Name() != "skill" {
		t.Errorf("Name() = %q, want %q", tool.Name(), "skill")
	}
	if !json.Valid(tool.Schema().Parameters) {
		t.Errorf("Schema().Parameters is not valid JSON: %s", tool.Schema().Parameters)
	}
	if tool.Schema().Name != "skill" {
		t.Errorf("Schema().Name = %q, want skill", tool.Schema().Name)
	}
}

func TestTool_List_ReturnsDiscoveredSkills(t *testing.T) {
	tool := New(makeLoader(t, "1.0.0", "review code", "instructions"))
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"list"}`))
	if err != nil {
		t.Fatalf("Execute(list): %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}
	if !strings.Contains(string(res.Data), "code-review") {
		t.Errorf("list result missing skill name: %s", res.Data)
	}
}

func TestTool_Load_ReturnsInstructions(t *testing.T) {
	const instructions = "## Steps\n1. read the diff\n2. comment"
	tool := New(makeLoader(t, "1.2.3", "review code", instructions))

	res, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"load","name":"code-review"}`))
	if err != nil {
		t.Fatalf("Execute(load): %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got error: %s", res.Error)
	}

	var payload struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Tools        []string `json:"recommended_tools"`
		Instructions string   `json:"instructions"`
	}
	if err := json.Unmarshal(res.Data, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v (data=%s)", err, res.Data)
	}
	if payload.Name != "code-review" || payload.Version != "1.2.3" {
		t.Errorf("payload = %+v, want code-review/1.2.3", payload)
	}
	if payload.Instructions != instructions {
		t.Errorf("instructions = %q, want %q", payload.Instructions, instructions)
	}
	if len(payload.Tools) != 2 {
		t.Errorf("recommended_tools = %v, want 2 items", payload.Tools)
	}
}

func TestTool_Load_MissingName_Errors(t *testing.T) {
	tool := New(makeLoader(t, "1.0.0", "d", "i"))
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"load"}`))
	if err == nil {
		t.Fatal("expected error when name missing")
	}
	if res.Success {
		t.Error("expected Success=false")
	}
}

func TestTool_Load_UnknownSkill_Errors(t *testing.T) {
	tool := New(makeLoader(t, "1.0.0", "d", "i"))
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"load","name":"nope"}`))
	if err == nil {
		t.Fatal("expected error for unknown skill")
	}
	if res.Success {
		t.Error("expected Success=false")
	}
}

func TestTool_UnknownAction_Errors(t *testing.T) {
	tool := New(makeLoader(t, "1.0.0", "d", "i"))
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"frobnicate"}`))
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	if res.Success {
		t.Error("expected Success=false")
	}
}

func TestTool_InvalidJSON_Errors(t *testing.T) {
	tool := New(makeLoader(t, "1.0.0", "d", "i"))
	res, err := tool.Execute(context.Background(), json.RawMessage(`{not json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if res.Success {
		t.Error("expected Success=false")
	}
}

func TestTool_Execute_RespectsContextCancellation(t *testing.T) {
	tool := New(makeLoader(t, "1.0.0", "d", "i"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := tool.Execute(ctx, json.RawMessage(`{"action":"list"}`))
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if res.Success {
		t.Error("expected Success=false on cancellation")
	}
}

var _ pkg.Tool = (*Tool)(nil)
