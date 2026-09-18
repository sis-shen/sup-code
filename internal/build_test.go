package internal

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/supcode/supcode/internal/config"
	"github.com/supcode/supcode/internal/hooks/builtin"
	"github.com/supcode/supcode/internal/tools"
	"github.com/supcode/supcode/internal/tools/bash"
	"github.com/supcode/supcode/pkg"
)

// testConfig 构造一个不依赖用户环境配置的 Config。
func testConfig(t *testing.T) *config.Manager {
	t.Helper()
	m := config.NewManager()
	if err := m.Load(); err != nil {
		t.Fatalf("config Load(): %v", err)
	}
	// Set 优先于用户配置文件/环境变量
	require.NoError(t, m.Set(config.ConfigKeyLLMProvider, "openai"))
	require.NoError(t, m.Set(config.ConfigKeyLLMAPIKey, "sk-test"))
	require.NoError(t, m.Set(config.ConfigKeyLLMModel, "gpt-4o"))
	require.NoError(t, m.Set(config.ConfigKeyLLMMaxTokens, 4096))
	return m
}

func TestBuildAgent_NoMCPConfig_Unaffected(t *testing.T) {
	agent, err := BuildAgent(testConfig(t))
	if err != nil {
		t.Fatalf("BuildAgent: %v", err)
	}

	for _, name := range []string{"bash", "read_file", "write_file", "edit_file", "glob", "grep", "skill"} {
		if !contains(agent.ListTools(), name) {
			t.Errorf("ListTools() missing %q, got %v", name, agent.ListTools())
		}
	}

	if _, ok := agent.(io.Closer); ok {
		t.Log("no mcp configured: agent is not an io.Closer (expected)")
	}
}

func TestBuildAgent_MCP_BadServer_GracefulDegradation(t *testing.T) {
	m := testConfig(t)
	require.NoError(t, m.Set(config.ConfigKeyMCPServers, []map[string]any{
		{"name": "bad", "transport": "stdio", "command": "nonexistent-binary-xyz"},
	}))

	agent, err := BuildAgent(m)
	if err != nil {
		t.Fatalf("BuildAgent should not fail when an mcp server is unreachable: %v", err)
	}

	// 内置工具不受影响
	if !contains(agent.ListTools(), "bash") {
		t.Errorf("builtin tools lost after failed mcp connect: %v", agent.ListTools())
	}
}

func TestBuildAgent_MCP_ClosesOnShutdown(t *testing.T) {
	m := testConfig(t)
	require.NoError(t, m.Set(config.ConfigKeyMCPServers, []map[string]any{
		{"name": "bad", "transport": "stdio", "command": "nonexistent-binary-xyz"},
	}))

	agent, err := BuildAgent(m)
	if err != nil {
		t.Fatalf("BuildAgent: %v", err)
	}

	closer, ok := agent.(io.Closer)
	if !ok {
		t.Fatal("agent with mcp configured should implement io.Closer")
	}
	if err := closer.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
	// 幂等：再次 Close 不应 panic
	_ = closer.Close()
}

func TestBuildAgent_SkillTool_ListsRepoBuiltinSkills(t *testing.T) {
	// 从包目录切到仓库根，使 skill.GetSkillDirs() 的相对 builtin 目录 "skills" 指向仓库自带技能
	t.Chdir("..")

	agent, err := BuildAgent(testConfig(t))
	if err != nil {
		t.Fatalf("BuildAgent: %v", err)
	}

	res, err := agent.ExecuteTool(context.Background(), "skill", json.RawMessage(`{"action":"list"}`))
	if err != nil {
		t.Fatalf("ExecuteTool(skill list): %v", err)
	}
	if !res.Success {
		t.Fatalf("skill list failed: %s", res.Error)
	}
	for _, want := range []string{"code-review", "test-generator"} {
		if !strings.Contains(string(res.Data), want) {
			t.Errorf("skill list missing %q: %s", want, res.Data)
		}
	}
}

func TestBuildAgent_AuditHookRegisteredByDefault(t *testing.T) {
	agent, err := BuildAgent(testConfig(t))
	if err != nil {
		t.Fatalf("BuildAgent: %v", err)
	}
	if !contains(agent.ListHooks(), "audit_log") {
		t.Errorf("audit_log hook should be registered by default, got %v", agent.ListHooks())
	}
}

func TestBuildAgent_GitCommitHookDisabledByDefault(t *testing.T) {
	agent, err := BuildAgent(testConfig(t))
	if err != nil {
		t.Fatalf("BuildAgent: %v", err)
	}
	if contains(agent.ListHooks(), "git_auto_commit") {
		t.Errorf("git_auto_commit hook must be disabled by default, got %v", agent.ListHooks())
	}
}

func TestBuildAgent_GitCommitHookEnabledViaConfig(t *testing.T) {
	m := testConfig(t)
	require.NoError(t, m.Set(config.ConfigKeyHooksGitCommitEnabled, true))

	agent, err := BuildAgent(m)
	if err != nil {
		t.Fatalf("BuildAgent: %v", err)
	}
	if !contains(agent.ListHooks(), "git_auto_commit") {
		t.Errorf("git_auto_commit hook should be registered when enabled, got %v", agent.ListHooks())
	}
}

func TestBuildAgent_AuditHookDisabledViaConfig(t *testing.T) {
	m := testConfig(t)
	require.NoError(t, m.Set(config.ConfigKeyHooksAuditEnabled, false))

	agent, err := BuildAgent(m)
	if err != nil {
		t.Fatalf("BuildAgent: %v", err)
	}
	if contains(agent.ListHooks(), "audit_log") {
		t.Errorf("audit_log hook should be absent when disabled, got %v", agent.ListHooks())
	}
}

// recordingPermEngine 记录所有 LogAction 调用，用于端到端验证审计 Hook 链路。
type recordingPermEngine struct {
	mu      sync.Mutex
	actions []pkg.Action
}

func (r *recordingPermEngine) Check(ctx context.Context, action pkg.Action) (pkg.Decision, error) {
	return pkg.DecisionAllow, nil
}
func (r *recordingPermEngine) AddRule(rule pkg.PermissionRule) error { return nil }
func (r *recordingPermEngine) RemoveRule(ruleID string) error        { return nil }
func (r *recordingPermEngine) ListRules() []pkg.PermissionRule       { return nil }
func (r *recordingPermEngine) LogAction(ctx context.Context, action pkg.Action, decision pkg.Decision, result string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions = append(r.actions, action)
	return nil
}
func (r *recordingPermEngine) recorded() []pkg.Action {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]pkg.Action, len(r.actions))
	copy(cp, r.actions)
	return cp
}

var _ pkg.PermissionEngine = (*recordingPermEngine)(nil)

// TestAuditHook_EndToEnd_RecordsToolCall 不依赖真实 LLM/网络，
// 验证链路：工具执行 -> Registry.Execute -> AuditHook.AfterTool -> PermissionEngine.LogAction。
func TestAuditHook_EndToEnd_RecordsToolCall(t *testing.T) {
	permEng := &recordingPermEngine{}
	reg := tools.NewRegistry(permEng)
	if err := reg.RegisterHook(builtin.NewAuditHook(permEng)); err != nil {
		t.Fatalf("RegisterHook: %v", err)
	}
	if err := reg.Register(&bash.Tool{}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if _, err := reg.Execute(context.Background(), "bash", json.RawMessage(`{"command":"echo hi"}`)); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// AuditHook 异步写日志，轮询等待
	deadline := time.Now().Add(2 * time.Second)
	for {
		actions := permEng.recorded()
		if len(actions) > 0 {
			if actions[0].ToolName != "bash" {
				t.Fatalf("recorded tool = %q, want bash", actions[0].ToolName)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("AuditHook.LogAction was not called within timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
