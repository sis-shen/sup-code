# Task: MCP / Skill 真正接入 Tool Pool，清理冗余 Hook 实现并接入主逻辑链路

> 本文档由代码审查（见对话记录）产出的问题清单转化而来。执行者：AI 编码 Agent 本身。
> 验收方式：**不依赖人工点验**，每个阶段完成后由执行者自行运行 `go build` / `go vet` / `go test`（单元 + 集成），
> 命令输出作为验收证据附在提交说明或 `tasks/acceptance/` 报告中。只有 checklist 全部打勾才能进入下一阶段。

---

## 0. 背景（审查结论摘要）

| 模块 | 现状 |
|---|---|
| Agent Loop (`internal/agent/loop.go`) | ✅ 已实现，Plan-and-Execute，`toolRegistry.ListSchemas()` / `Execute()` 是唯一的工具查询与执行入口 |
| Tool Pool (`pkg.Tool` + `internal/tools/registry.go`) | ✅ 抽象健全，`Register/Execute` 统一 |
| 内置 Tool | ✅ 6 个，已在 `internal/build.go` 注册 |
| MCP (`internal/mcp/bridge.go`) | ⚠️ **适配器代码已实现**（`MCPToolAdapter implements pkg.Tool`），但 `internal/build.go` 从未创建 `mcp.Client` / 调用 `Bridge.RegisterAllTools`，生产环境零 MCP 工具 |
| Skill (`internal/skill/*`) | ⚠️ **加载/拼接逻辑已实现**，但没有实现为 `pkg.Tool`，也未注册进 Tool Pool，Agent 完全不可见 |
| Hook | ⚠️ **两套并行实现**：`internal/tools/registry.go` 内联的 Before/After 循环（**这套在生产中真正生效**，因为 `Agent.Run -> toolRegistry.Execute` 会走它）与 `internal/hooks/engine.go` 的 `HookEngine`（功能等价但从未被任何生产代码引用，纯冗余）；且内置钩子 `internal/hooks/builtin/{audit,git_commit}.go` 从未被注册进 Registry，主链路上**没有任何钩子在运行** |

**核心结论**：Tool Pool 抽象本身没问题，缺口全部在“组装根”`internal/build.go` 没有把 MCP、Skill、Hook 接进去。因此本任务**不需要改动 `internal/agent/loop.go` 的核心循环**（它已经是"从 Registry 拿 Schema、按名执行"的通用逻辑），只需要：
1. 把 MCP 工具适配器注册进 `build.go` 里那个和内置工具**同一个** `*tools.Registry` 实例；
2. 把 Skill 包装成一个 `pkg.Tool`，同样注册进这个 Registry；
3. 删除 `internal/hooks/engine.go` 这套冗余实现，把它唯一有价值的"每钩子超时保护"逻辑合并进 `internal/tools/registry.go` 的钩子链；
4. 在 `build.go` 里默认注册内置钩子（审计日志、Git 自动提交），使其真正参与 `Agent.Run` 的每一次工具调用。

## 1. 范围声明（Out of Scope）

以下问题在本次审查中被发现，但**不在本任务范围内**，不要顺手修改，避免范围蔓延：
- Agent Loop 的流式输出未打通到 TUI（`pkg.InteractionService.StreamResponse` 未接线）
- Agent Loop 不支持并行工具调用（`ToolSelector` 只取第一个 `tool_call`）
- TUI 的 Ctrl+C 未接入 `ctx` 取消
- `internal/subagent` 并行子 Agent 编排

如果在实现过程中发现必须联动修改以上内容才能让本任务通过测试，先在对应阶段的验收报告里注明原因，再决定是否小范围联动。

## 2. 通用执行约定

- 工作目录：`D:\codes\sup-code`
- 每个阶段开始前先跑一次基线，阶段结束后重新跑一次，对比差异：
  ```powershell
  go build ./...
  go vet ./...
  go test ./... 2>&1 | Select-String -Pattern "FAIL|ok "
  ```
- **已知与本任务无关的基线失败**（Phase 0 已确认，环境变量 `SUPCODE_LLM_API_KEY` 在本机被全局设置导致）：
  - `internal` 包 `TestNewSupCode_MissingAPIKey`
  - `internal/cli` 包 `TestConfigGetExistingKey`
  - `internal/config` 包 `TestDefaultValues`、`TestMissingAPIKey`
  - `tests/integration` 包 `TestAgentLoop_ReadFile`（依赖真实/本地 mock LLM server，网络环境相关）
  这些失败在执行 `go test` 前可通过临时清空该环境变量复现验证：
  ```powershell
  $env:SUPCODE_LLM_API_KEY = $null; go test ./internal/... 2>&1 | Select-String "FAIL|ok "
  ```
  验收时只需保证：**这些用例的失败原因不变差**（即不能因为本任务的改动引入新的失败原因），新增测试必须全部通过。
- 涉及接口变更（`pkg/interfaces.go`）时，先搜索全仓库所有实现者，确认改动不遗漏任何实现（见下方每阶段"影响面检查"）。
- 每阶段建议单独 commit，commit message 前缀使用 `toolpool(phaseN): ...`，但除非用户要求，不做 push / PR。

---

## Phase 0 — 基线锁定

### 目标
固化当前 build/test 基线，作为后续阶段回归对比的对照组。

### 步骤
1. `go build ./...` — 记录是否成功。
2. `go test ./... 2>&1` — 记录全部包的 pass/fail 列表。
3. 将结果保存为 `tasks/acceptance/toolpool-baseline.md`（简单列表即可，不需要长篇分析）。

### 验收 Checklist
- [x] `go build ./...` 无报错（已于本次审查中验证通过）
- [x] `go test ./...` 输出已记录，已知失败项与第 2 节列表一致，无额外失败
- [x] `tasks/acceptance/toolpool-baseline.md` 已写入

---

## Phase 1 — MCP 真正接入 Tool Pool

### 目标
`internal/build.go` 组装出的 Agent，在配置中声明了 MCP server 时，能够真实连接该 server，并把它暴露的工具通过 `mcp.Bridge` 注册进**与内置工具相同的** `*tools.Registry` 实例，使 `Agent.Run` 无需任何改动即可调度 MCP 工具。

### 涉及文件
| 文件 | 改动类型 |
|---|---|
| `pkg/interfaces.go` | 修改：`Config` 接口新增 `UnmarshalKey` |
| `internal/config/config.go` | 修改：实现 `UnmarshalKey`（委托给 viper） |
| `internal/mcp/bootstrap.go` | 新增：批量连接 + 结果收集的辅助函数 |
| `internal/agent/loop.go` | 修改：`Agent` 增加 `ListTools() []string` 只读introspection方法（供测试与未来 `/tools` 命令使用，不改变 `Run` 逻辑） |
| `internal/build.go` | 修改：读取 `mcp.servers` 配置、创建 `mcp.Client`、调用 `Bridge.RegisterAllTools`、返回值包装为 `io.Closer` |
| `cmd/supcode/main.go` | 修改：在 Mode B / Mode C 构建 agent 后，若其实现 `io.Closer` 则 `defer Close()` |
| `internal/config/defaults.go` | 确认 `ConfigKeyMCPServers = "mcp.servers"` 已存在，无需新增 |
| `internal/mcp/pool_integration_test.go` | 新增：真实 `tools.Registry` + mock MCP server 的端到端单测 |
| `internal/config/config_test.go` | 新增：`UnmarshalKey` 测试 |
| `internal/build_test.go` | 新增：`BuildAgent` 对 MCP 配置的接线测试（含优雅降级） |

### 详细步骤

**1) 扩展 Config 接口**（`pkg/interfaces.go`，`Config` 接口内）：
```go
// UnmarshalKey 将指定 key 下的配置解析到 out（用于列表/结构体类配置，如 mcp.servers）
UnmarshalKey(key string, out any) error
```
在 `internal/config/config.go` 的 `Manager` 上实现：
```go
func (m *Manager) UnmarshalKey(key string, out any) error {
    return m.v.UnmarshalKey(key, out)
}
```
**影响面检查**：全仓库搜索 `pkg.Config = `/`_ pkg.Config`，确认只有 `internal/config.Manager` 一个实现者（已在审查中核实），无需修改 mock。

**2) 新增 `internal/mcp/bootstrap.go`**：
```go
package mcp

import (
    "context"
    "github.com/supcode/supcode/pkg"
)

// ConnectResult 记录单个 MCP server 的连接结果。
type ConnectResult struct {
    ServerName string
    Err        error
}

// ConnectConfigured 依次连接给定的 MCP server 配置列表。
// 单个 server 连接失败不会中断其余 server 的连接（优雅降级）。
func ConnectConfigured(ctx context.Context, client pkg.MCPClient, servers []pkg.MCPServerConfig) []ConnectResult {
    results := make([]ConnectResult, 0, len(servers))
    for _, s := range servers {
        err := client.Connect(ctx, s)
        results = append(results, ConnectResult{ServerName: s.Name, Err: err})
    }
    return results
}
```

**3) `internal/agent/loop.go` 增加只读 introspection**：
```go
// ListTools 返回当前工具池中所有已注册工具名（用于诊断/测试，不参与主循环逻辑）。
func (a *Agent) ListTools() []string {
    return a.toolRegistry.List()
}
```

**4) `internal/build.go` 接线**（在注册完内置工具之后、构建 `agent.NewAgent` 之前插入）：
```go
var mcpServers []pkg.MCPServerConfig
_ = cfg.UnmarshalKey(config.ConfigKeyMCPServers, &mcpServers) // key 不存在时保持空列表，非致命错误

var mcpClient pkg.MCPClient
if len(mcpServers) > 0 {
    c := mcp.NewClient()
    for _, r := range mcp.ConnectConfigured(context.Background(), c, mcpServers) {
        if r.Err != nil {
            slog.Warn("mcp server connect failed, skipping its tools", "server", r.ServerName, "error", r.Err)
        }
    }
    if err := mcp.NewBridge(c, reg).RegisterAllTools(context.Background()); err != nil {
        slog.Warn("some mcp tools failed to register", "error", err)
    }
    mcpClient = c
}
```
`BuildAgent` 返回前，若 `mcpClient != nil`，用一个轻量包装类型返回，使调用方可以 `Close()` 掉 MCP 连接：
```go
type agentWithCleanup struct {
    pkg.Agent
    mcpClient pkg.MCPClient
}

func (a *agentWithCleanup) Close() error {
    if a.mcpClient != nil {
        return a.mcpClient.Close()
    }
    return nil
}
```
`BuildAgent` 末尾：
```go
if mcpClient != nil {
    return &agentWithCleanup{Agent: a, mcpClient: mcpClient}, nil
}
return a, nil
```
**注意**：`BuildAgent(cfg pkg.Config)` 目前不接收 `context.Context`，MCP 连接只能用 `context.Background()`；这是可接受的启动期行为，不在本任务中改造 `BuildAgent` 的签名（避免影响 `cmd/supcode/main.go` 之外的调用方）。

**5) `cmd/supcode/main.go` 增加优雅关闭**（`runSingleShot` 与 Mode C 两处 `BuildAgent` 调用之后）：
```go
agent, err := internal.BuildAgent(cfg)
...
if closer, ok := agent.(io.Closer); ok {
    defer closer.Close()
}
```

**6) MCP server 配置读取方式**：用户在 `~/.supcode/config.yaml` 或项目 `.supcode/config.yaml` 中声明：
```yaml
mcp:
  servers:
    - name: filesystem
      transport: stdio
      command: npx
      args: ["-y", "@modelcontextprotocol/server-filesystem", "./mcp-test"]
```
`viper.UnmarshalKey("mcp.servers", &servers)` 天然支持这种嵌套列表反序列化，无需额外解析代码。

### 新增/修改测试

1. `internal/config/config_test.go` 新增 `TestManager_UnmarshalKey_MCPServers`：写入内存 viper 值（`m.Set("mcp.servers", []map[string]any{...})`），调用 `UnmarshalKey`，断言反序列化出的 `[]pkg.MCPServerConfig` 字段正确。
2. `internal/mcp/pool_integration_test.go`（新文件，`package mcp`，可直接使用已有的 `NewMockMCPServer` 测试工具）：
   ```go
   func TestBridge_UnifiedToolPool_WithRealRegistry(t *testing.T) {
       mockSrv := NewMockMCPServer(t, []pkg.ToolSchema{
           {Name: "mcp_echo", Description: "echo via mcp", Parameters: json.RawMessage(`{"type":"object"}`)},
       }, protocolVersion)
       defer mockSrv.Stop()

       reg := tools.NewRegistry(nil)
       require.NoError(t, reg.Register(&bash.Tool{})) // 内置工具

       client := NewClient()
       require.NoError(t, client.Connect(context.Background(),
           pkg.MCPServerConfig{Name: "mock", Transport: "sse", Command: mockSrv.URL()}))
       defer client.Close()

       require.NoError(t, NewBridge(client, reg).RegisterAllTools(context.Background()))

       names := reg.List()
       assert.Contains(t, names, "bash")
       assert.Contains(t, names, "mcp_echo") // 证明 MCP 工具与内置工具共存同一 Registry

       result, err := reg.Execute(context.Background(), "mcp_echo", json.RawMessage(`{}`))
       require.NoError(t, err)
       assert.True(t, result.Success) // 证明走 Registry.Execute 统一入口也能调用到 MCP 工具
   }
   ```
   这是本阶段**最关键的验收证据**：证明 MCP 工具通过与内置工具完全相同的 `Register`/`Execute` 路径工作，没有独立的第二套调度逻辑。
3. `internal/build_test.go`（新文件，`package internal`）：
   - `TestBuildAgent_NoMCPConfig_Unaffected`：不配置 `mcp.servers`，`BuildAgent` 正常返回，`ListTools()`（通过类型断言拿到 `*agent.Agent`，注意 `agentWithCleanup` 场景下需要先判断类型）只包含 6 个内置工具。
   - `TestBuildAgent_MCP_BadServer_GracefulDegradation`：配置一个必然连接失败的 server（如 `command: nonexistent-binary-xyz`），断言 `BuildAgent` 不返回 error、不 panic，且不会把失败 server 的工具混入。
   - `TestBuildAgent_MCP_ClosesOnShutdown`：配置一个可连接的 mock stdio/sse server，`BuildAgent` 返回值类型断言为 `io.Closer` 成功，调用 `Close()` 无 panic。

### 验收 Checklist
- [x] `go build ./...` 通过
- [x] `go vet ./...` 无新增告警
- [x] `go test ./internal/config/... ./internal/mcp/... ./internal/... -run . -v` 全部新增用例通过（`TestManager_UnmarshalKey_*`、`TestBridge_UnifiedToolPool_WithRealRegistry`、`TestConnectConfigured_GracefulDegradation`、`TestBuildAgent_*`）
- [x] `go test ./...` 整体不新增失败（对照 Phase 0 基线，仅剩 4 个已知基线失败）
- [x] 手动验证一次真实链路（可选但建议）：以 `TestBridge_UnifiedToolPool_WithRealRegistry`（真实 `tools.Registry` + mock MCP SSE server + `tools/call`）作为等价端到端覆盖
- [x] `internal/build.go` 中 MCP 相关代码有清晰注释说明"失败降级、不阻塞内置工具"策略
- [x] `pkg.Config` 接口新增方法后，`internal/wire_test.go` 中的编译期接口检查仍然通过（证明未遗漏实现）

---

## Phase 2 — Skill 真正接入 Tool Pool

### 目标
把 Skill 包装为标准 `pkg.Tool`，注册进与内置工具、MCP 工具**同一个** Registry。Agent 通过 function calling 自主决定何时调用 `skill` 工具去发现/加载技能说明，工具返回的内容会像其他工具结果一样被 `Agent.Run`（`internal/agent/loop.go` 现有逻辑，`pkg.RoleTool` 消息）自动写回对话历史 —— **不需要改动 Agent Loop 或 ContextManager**。

> 设计取舍说明：不采用"把所有 skill 的 SystemPrompt 无条件拼进 system prompt"的旧 `injector.go` 方案作为默认路径，因为那样不受 LLM 控制、会无限膨胀上下文。改为"技能即工具"：LLM 按需调用 `skill` 工具的 `list`/`load` 动作，这与 Tool Pool 的统一调度模型完全一致，也是本任务标题"真正接入 tool pool"的字面要求。`internal/skill/injector.go` 保留在代码库中作为未来可选的"常驻技能"能力，本阶段不删除也不默认启用。

### 涉及文件
| 文件 | 改动类型 |
|---|---|
| `internal/tools/skilltool/tool.go` | 新增：实现 `pkg.Tool` 的 Skill 工具 |
| `internal/tools/skilltool/tool_test.go` | 新增：单元测试 |
| `internal/build.go` | 修改：构造 `skill.NewSkillLoader(skill.GetSkillDirs())`，注册 `skilltool.New(loader)` |
| `internal/build_test.go` | 修改：新增断言 `ListTools()` 包含 `"skill"` |

### 详细步骤

**1) 新建 `internal/tools/skilltool/tool.go`**：
```go
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

func New(loader *skill.SkillLoader) *Tool {
    return &Tool{loader: loader}
}

func (t *Tool) Name() string { return toolName }

func (t *Tool) Description() string {
    return "发现并加载可复用的专家技能指导（skill）。action=list 查看当前可用技能列表；" +
        "action=load 配合 name 参数加载指定技能的详细操作指南，加载后请遵循其中的指导完成任务。"
}

func (t *Tool) Schema() pkg.ToolSchema {
    return pkg.ToolSchema{Name: toolName, Description: t.Description(), Parameters: schemaParams}
}

type params struct {
    Action string `json:"action"`
    Name   string `json:"name"`
}

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
            Name: s.Manifest.Name, Version: s.Manifest.Version,
            Description: s.Manifest.Description, Tools: s.Manifest.Tools,
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
```

**2) `internal/build.go` 接线**：
```go
import (
    "github.com/supcode/supcode/internal/skill"
    "github.com/supcode/supcode/internal/tools/skilltool"
)
...
builtinDir, userDir, projectDir := skill.GetSkillDirs()
skillLoader := skill.NewSkillLoader(builtinDir, userDir, projectDir)
if err := reg.Register(skilltool.New(skillLoader)); err != nil {
    return nil, err
}
```
放在注册 6 个内置工具的同一个循环之后（MCP 注册之前或之后均可，顺序不影响功能）。

**影响面检查**：`internal/skill` 包名与新建的 `internal/tools/skilltool` 包名不同，`build.go` 中同时 import 二者不会有命名冲突（原 `internal/skill` 包名为 `skill`，新包名为 `skilltool`）。

### 新增/修改测试

1. `internal/tools/skilltool/tool_test.go`：
   - `TestTool_List_ReturnsDiscoveredSkills`：用临时目录构造 1-2 个假 `skill.json`，`loader := skill.NewSkillLoader(tmpDir, "", "")`，执行 `action=list`，断言 `Data` 中包含技能名。
   - `TestTool_Load_ReturnsInstructions`：同上，`action=load` + `name`，断言返回的 `instructions` 字段等于 `prompts/system.md` 内容。
   - `TestTool_Load_MissingName_Errors`
   - `TestTool_UnknownAction_Errors`
   - `TestTool_Load_UnknownSkill_Errors`
   - `TestTool_Schema_And_Name`：断言 `Name() == "skill"`，`Schema().Parameters` 是合法 JSON。
   - `TestTool_Execute_RespectsContextCancellation`
2. `internal/build_test.go` 追加：
   - `TestBuildAgent_RegistersSkillTool`：`BuildAgent` 后 `ListTools()` 包含 `"skill"`。
   - `TestBuildAgent_SkillTool_ListsRepoBuiltinSkills`：由于仓库自带 `skills/code-review`、`skills/test-generator`，直接通过类型断言拿到 registry（或给 `agent.Agent` 增加一个仅测试可见的 `ExecuteTool` 转发方法，见下）验证 `skill` 工具 `action=list` 能发现这两个内置技能。

   为了让这个集成测试能真正"执行"一次工具（而不仅仅是看到工具名），在 `internal/agent/loop.go` 追加一个测试/诊断用的直通方法（与 `ListTools` 放在一起）：
   ```go
   // ExecuteTool 直接通过工具池执行一个工具，绕过 Plan/Select 流程。
   // 用于诊断命令与集成测试；生产 Agent.Run 不使用此方法。
   func (a *Agent) ExecuteTool(ctx context.Context, name string, params json.RawMessage) (pkg.ToolResult, error) {
       return a.toolRegistry.Execute(ctx, name, params)
   }
   ```
   这个方法同时也是给 Phase 1 MCP 集成测试复用的通用手段，如果 Phase 1 已经加了类似方法就不要重复添加。

### 验收 Checklist
- [x] `go build ./...` 通过
- [x] `go test ./internal/tools/skilltool/... -v` 全部通过
- [x] `go test ./internal/... -run TestBuildAgent -v` 全部通过（含 Phase 1 + Phase 2 新增用例）
- [x] `go test ./...` 整体不新增失败
- [x] 手动执行一次单发查询验证端到端可用：以 `TestBuildAgent_SkillTool_ListsRepoBuiltinSkills`（真实 `BuildAgent` + 仓库自带 `skills/` 目录 + `skill` 工具 `action=list`）作为等价覆盖
- [x] 确认未修改 `internal/agent/loop.go` 的 `Run`/`executeStep` 核心逻辑（只新增了只读 introspection 方法）

> 附带修复：发现仓库自带的 `skills/*/skill.json` 均带 UTF-8 BOM，导致 `ParseManifest` 报
> `invalid character 'ï'`，Skill 功能此前实际不可用。已在 `internal/skill/manifest.go` 的
> `ParseManifest` 中容忍 BOM，并新增 `TestParseManifestWithUTF8BOM`。

---

## Phase 3 — 去除冗余 Hook 实现，强化 Registry 内的钩子链

### 目标
消除 `internal/hooks/engine.go`（`HookEngine` + `HookWrapper`）与 `internal/tools/registry.go` 内联钩子循环之间的功能重复，只保留**一套**实现——即生产环境唯一在跑的 `Registry.Execute` 内联链，同时把 `HookEngine` 里唯一有价值、但 `Registry` 目前没有的特性（**每个钩子独立超时 + ctx 取消检测**）合并进去。

### 涉及文件
| 文件 | 改动类型 |
|---|---|
| `internal/hooks/engine.go` | **删除** |
| `internal/hooks/engine_test.go` | **删除** |
| `internal/tools/registry.go` | 修改：钩子链加入超时保护 + ctx 检测 |
| `internal/tools/registry_test.go` | 修改/新增：覆盖超时与取消场景 |
| `internal/hooks/builtin/*.go` | 不变（仍然是独立的 `pkg.ToolHook` 实现，供 Phase 4 注册使用） |

### 详细步骤

**1) 确认删除安全性**：`internal/hooks/engine.go` 中的符号（`HookEngine`, `NewHookEngine`, `HookWrapper`, `NewHookWrapper`）仅被同目录下的 `engine_test.go` 引用（已在审查中用全仓库搜索确认），`internal/hooks/builtin/*.go` 不依赖它们。删除后运行 `go build ./...` 验证零编译错误。

**2) 直接删除文件**：
```powershell
Remove-Item internal\hooks\engine.go, internal\hooks\engine_test.go
```
删除后如果 `internal/hooks` 目录只剩 `builtin/` 子目录、没有直接 .go 文件，是允许的（Go 允许只有子包的空父目录）。

**3) 强化 `internal/tools/registry.go` 的钩子链**（替换 `Execute` 方法中第 131-152 行的循环逻辑）：
```go
const defaultHookTimeout = 30 * time.Second

...

currentParams := params
for _, hook := range hooks {
    select {
    case <-ctx.Done():
        return pkg.ToolResult{Success: false, Error: ctx.Err().Error()}, ctx.Err()
    default:
    }

    hookCtx, cancel := context.WithTimeout(ctx, defaultHookTimeout)
    newParams, err := hook.BeforeTool(hookCtx, name, currentParams)
    cancel()
    if err != nil {
        return pkg.ToolResult{Success: false, Error: fmt.Sprintf("hook %s rejected: %v", hook.Name(), err)}, err
    }
    if newParams != nil {
        currentParams = newParams
    }
}

result, err := tool.Execute(ctx, currentParams)

runAfterHooks := func() {
    for _, hook := range hooks {
        select {
        case <-ctx.Done():
            return
        default:
        }
        hookCtx, cancel := context.WithTimeout(ctx, defaultHookTimeout)
        if afterErr := hook.AfterTool(hookCtx, name, currentParams, result); afterErr != nil {
            log.Printf("[tools] AfterTool hook %s error: %v", hook.Name(), afterErr)
        }
        cancel()
    }
}

if err != nil {
    runAfterHooks()
    return result, err
}
runAfterHooks()
return result, nil
```
（需要新增 `import ("context"; "log"; "time")`，`context` 已存在。）

**关键行为变化**：
- `AfterTool` 的错误从"直接吞掉"改为 `log.Printf` 记录（与旧 `HookEngine.ExecuteAfter` 行为对齐，之前 `registry.go` 是完全静默 `hook.AfterTool(...)` 不检查返回值——见原代码第 145/151 行，这是一个**隐藏 bug**，本阶段顺带修掉）。
- 新增每钩子 30 秒超时与 ctx 取消检测，防止一个挂死的 Hook（例如 `git_commit` 卡在网络 IO）拖死整个 Agent 循环。

### 新增/修改测试

`internal/tools/registry_test.go` 新增：
1. `TestExecute_AfterToolHookError_IsLoggedNotSwallowedSilently`：注册一个 `AfterTool` 返回 error 的假钩子，断言 `Execute` 整体仍返回工具本身的成功结果（错误不传播到调用方，但可以通过重定向 `log` 输出断言被记录，或者只断言不 panic/不影响返回值，取决于实现复杂度，二选一即可）。
2. `TestExecute_BeforeToolHook_TimesOut`：注册一个 `BeforeTool` 里 `time.Sleep` 超过测试用临时缩短的超时时间（可以把 `defaultHookTimeout` 抽成 `var`，测试里用 `//go:build` 或直接构造一个可注入超时的 `Registry` 字段，例如 `NewRegistryWithHookTimeout(permEng, timeout)` 供测试使用，生产 `NewRegistry` 用默认值），断言最终返回超时错误而不是无限阻塞测试进程。
3. `TestExecute_ContextCancelled_DuringHookChain`：`ctx` 提前 cancel，断言钩子链提前中止且不执行工具本体。
4. 保留并确保现有 `registry_test.go` 用例全部仍然通过（回归）。

### 验收 Checklist
- [x] `internal/hooks/engine.go` 与 `engine_test.go` 已删除
- [x] `go build ./...` 通过（证明没有遗留引用）
- [x] `go vet ./...` 无告警
- [x] `go test ./internal/tools/... -timeout 60s -v` 全部通过，新增的超时/取消用例通过且不产生挂起
- [x] `go test ./internal/hooks/... -v`：目录仍然编译通过（只剩 `builtin` 子包的测试）
- [x] `go test ./...` 整体不新增失败（Phase 5 全量回归复核）
- [x] 全仓库搜索确认无残留引用（`hooks\.HookEngine|hooks\.NewHookEngine|hooks\.HookWrapper` 结果为空）

---

## Phase 4 — 将 Hook 接入主逻辑链路（默认注册内置钩子）

### 目标
让 `internal/hooks/builtin` 下现成的 `AuditHook`、`GitCommitHook` 在 `BuildAgent` 组装时被真正 `RegisterHook` 进 Registry，使其在**每一次** `Agent.Run -> toolRegistry.Execute` 调用中真实生效，而不是只存在于自己的单测里。默认策略要保守（审计日志默认开，Git 自动提交默认关，避免误伤用户仓库）。

### 涉及文件
| 文件 | 改动类型 |
|---|---|
| `internal/config/defaults.go` | 修改：新增两个配置键 |
| `internal/build.go` | 修改：按配置注册内置钩子 |
| `internal/build_test.go` | 修改：新增钩子注册相关断言 |
| `docs/`（如有用户手册） | 可选：补充配置说明，非强制 |

### 详细步骤

**1) `internal/config/defaults.go` 新增配置键**：
```go
ConfigKeyHooksAuditEnabled     = "hooks.audit_log.enabled"
ConfigKeyHooksGitCommitEnabled = "hooks.git_auto_commit.enabled"
```
`DefaultConfig` map 中追加：
```go
ConfigKeyHooksAuditEnabled:     true,
ConfigKeyHooksGitCommitEnabled: false,
```

**2) `internal/build.go` 接线**（在 `reg` 构造完、内置工具/MCP/Skill 都注册完之后）：
```go
import "github.com/supcode/supcode/internal/hooks/builtin"
...
if cfg.GetBool(config.ConfigKeyHooksAuditEnabled) {
    if err := reg.RegisterHook(builtin.NewAuditHook(permEng)); err != nil {
        return nil, fmt.Errorf("register audit hook: %w", err)
    }
}
if cfg.GetBool(config.ConfigKeyHooksGitCommitEnabled) {
    if err := reg.RegisterHook(builtin.NewGitCommitHook(true)); err != nil {
        return nil, fmt.Errorf("register git commit hook: %w", err)
    }
}
```

**3) 校验 `AuditHook` 依赖的 `PermissionEngine.LogAction`**：确认 `internal/permission` 包的实现有可用的 `LogAction`（审查中未展开，需要在实现本阶段时读一遍 `internal/permission/*.go` 确认签名匹配、无 panic 风险，比如日志文件路径不存在时的兜底）。

### 新增/修改测试

1. `internal/build_test.go` 追加：
   - `TestBuildAgent_AuditHookRegisteredByDefault`：默认配置下构建 agent，通过一个测试专用的 introspection（例如给 `agent.Agent` 增加 `HasHook(name string) bool`，内部调用 `a.toolRegistry`（需要 Registry 暴露 `ListHookNames()`，一并在 `internal/tools/registry.go` 和 `pkg.ToolRegistry` 接口新增，工作量小）验证 `"audit_log"` 已注册。
   - `TestBuildAgent_GitCommitHookDisabledByDefault`：默认配置下 `"git_auto_commit"` 不在钩子列表中。
   - `TestBuildAgent_GitCommitHookEnabledViaConfig`：设置 `cfg.Set(config.ConfigKeyHooksGitCommitEnabled, true)` 后重新 `BuildAgent`，断言钩子已注册。
2. **端到端集成测试**（`tests/integration/` 目录，参考该目录现有的 `TestAgentLoop_ReadFile` 写法）：
   - `TestAgentLoop_AuditHookRecordsToolCalls`：跑一次真实/mock LLM 驱动的 `Agent.Run`（复用现有集成测试的 mock LLM server 基础设施），执行完后断言 `PermissionEngine` 的审计日志中出现了对应的 `Action` 记录。如果现有集成测试基础设施因环境限制超时（Phase 0 记录的已知问题），允许把该用例标注为依赖同样的基础设施，并在报告中注明"与已知基线问题同因，不阻塞验收"，但**必须**提供一个不依赖真实 LLM 网络调用的等价单测替代（例如直接构造 `tools.Registry` + `RegisterHook` + `Execute`，绕开 Planner/Selector，验证钩子确实被调用），保证核心逻辑有测试覆盖。

因为 `pkg.ToolRegistry` 接口新增 `ListHookNames()` 属于接口变更，需要**影响面检查**：全仓库搜索所有实现 `pkg.ToolRegistry` 的类型（已知只有 `internal/tools.Registry` 和 `internal/mcp/bridge_test.go` 里的 `mockRegistry`），两处都要同步加上该方法。

### 验收 Checklist
- [x] `pkg/interfaces.go` 的 `ToolRegistry` 接口新增 `ListHookNames() []string`，`internal/tools/registry.go`、`internal/mcp/bridge_test.go`、`internal/agent/mock_llm_test.go`、`internal/subagent/mock_orchestrator_test.go`、`internal/tools/mock_registry_test.go` 均已同步实现
- [x] `go build ./...` 通过
- [x] `go test ./internal/... -run TestBuildAgent -v` 全部通过
- [x] `go test ./internal/hooks/... ./internal/permission/... -v` 全部通过
- [x] `TestAuditHook_EndToEnd_RecordsToolCall` 不依赖真实网络/LLM，证明 "工具执行 -> Registry.Execute -> AuditHook.AfterTool -> PermissionEngine.LogAction" 链路是通的
- [x] `go test ./...` 整体不新增失败（Phase 5 全量回归复核）
- [x] 默认配置下 `git_auto_commit` 钩子保持关闭（`TestBuildAgent_GitCommitHookDisabledByDefault`），审计默认开启（`TestBuildAgent_AuditHookRegisteredByDefault`）

> 附加实现：为便于验收，`pkg.Agent` 接口新增 `ListTools()` / `ListHooks()` / `ExecuteTool()` 三个诊断方法
> （唯一实现者 `agent.Agent` 及包装类型 `agentWithCleanup` 均已满足）。

---

## Phase 5 — 全量回归与收尾

### 目标
确认四个阶段合起来之后，整个系统仍然自洽，并把最终状态归档。

### 步骤与 Checklist
- [x] `go build ./...` 全量通过
- [x] `go vet ./...` 全量无告警
- [x] `go test ./... 2>&1 | Tee-Object -FilePath tasks/acceptance/toolpool-final.log`，与 Phase 0 基线逐包对比：
  - [x] 之前通过的包仍然通过（`internal/hooks` 因删除唯一测试文件不再出现，属预期）
  - [x] 已知基线失败（4 个用例）失败原因不变
  - [x] 新增测试全部通过
- [x] 用一次 mock 端到端调用验证三种工具来源共存于同一 Registry：`TestBridge_UnifiedToolPool_WithRealRegistry`（内置 + MCP）+ `TestBuildAgent_SkillTool_ListsRepoBuiltinSkills`（Skill）以真实 `Registry` 覆盖；无真实 LLM Key 环境下不以 `supcode "..."` 直连验证
- [x] 搜索确认仓库文档不存在"MCP/Skill 尚未接入"等过时描述；`docs/产品使用说明书.md` 第 11 节的 `mcp.servers` 配置格式与新实现一致，无需修改（`supcode mcp list` 子命令缺失属既有文档不一致，已记录为范围外）
- [x] 在 `tasks/acceptance/` 下新增 `report-toolpool.md`，总结改动文件、测试结果与范围外遗留问题
- [x] 本 `task.md` 中 Phase 1~5 的所有 checkbox 均已勾选

---

## 附：本任务结束后的架构对照（目标状态）

```
Agent.Run (未改动核心逻辑)
   ├─ toolRegistry.ListSchemas()  ──▶ 现在包含：内置6个 + skill + (若配置) MCP 动态发现的工具
   └─ toolRegistry.Execute(name, params)
            │
            ▼
   tools.Registry（唯一钩子链实现，去重后）
     Execute = 权限检查 → BeforeHook(30s超时) → tool.Execute → AfterHook(30s超时，错误会记录日志)
            │
   ┌────────┼─────────────────┬───────────────────────┐
   ▼        ▼                 ▼                       ▼
 内置工具   skill 工具         MCP 工具(MCPToolAdapter)   Hook: audit_log(默认开) / git_auto_commit(默认关)
 (build.go 注册)  (build.go 注册)   (build.go 按配置动态注册)   (build.go 按配置注册)
```

所有工具来源统一实现 `pkg.Tool`，统一经 `Register`/`Execute` 调度；所有钩子统一实现 `pkg.ToolHook`，统一经 `RegisterHook` 挂载到同一条链上——不再存在"MCP/Skill 是孤岛"或"两套 Hook 实现并存"的问题。
