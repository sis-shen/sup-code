# Tool Pool 改造 — 最终验收报告

对应任务：`task.md`（MCP / Skill 接入 Tool Pool，去冗余 Hook 并接入主链路）
完整测试日志：`tasks/acceptance/toolpool-final.log`
基线对照：`tasks/acceptance/toolpool-baseline.md`

## 1. 目标达成情况

| 目标 | 状态 | 证据 |
|---|---|---|
| MCP 以抽象工具方式接入 Tool Pool | ✅ | `internal/mcp/bootstrap.go` + `internal/build.go:connectMCP`；`TestBridge_UnifiedToolPool_WithRealRegistry` |
| Skill 以抽象工具方式接入 Tool Pool | ✅ | `internal/tools/skilltool/tool.go` + `build.go` 注册；`TestBuildAgent_SkillTool_ListsRepoBuiltinSkills` |
| 去除冗余 Hook 实现 | ✅ | 删除 `internal/hooks/engine.go`、`engine_test.go` |
| Hook 接入主逻辑链路 | ✅ | `build.go` 注册 `audit_log`（默认开）/`git_auto_commit`（默认关）；`TestAuditHook_EndToEnd_RecordsToolCall` |

改造后，内置工具、Skill 工具、MCP 工具全部实现 `pkg.Tool`，统一经同一个 `tools.Registry` 的
`Register` / `ListSchemas` / `Execute` 调度；所有 Hook 实现 `pkg.ToolHook`，统一经 `RegisterHook`
挂载到 `Registry.Execute` 内唯一一条钩子链。`internal/agent/loop.go` 的 `Run` / `executeStep`
核心逻辑未改动。

## 2. 主要改动文件清单

### 新增
- `internal/mcp/bootstrap.go` — `ConnectConfigured`，批量连接 MCP server 并优雅降级
- `internal/mcp/pool_integration_test.go` — MCP 工具与内置工具共存同一 Registry 的关键集成测试
- `internal/tools/skilltool/tool.go` — 把技能库包装为标准 `pkg.Tool`（`action=list|load`）
- `internal/tools/skilltool/tool_test.go`
- `internal/build_test.go` — BuildAgent 的 MCP/Skill/Hook 接线测试
- `tasks/acceptance/toolpool-baseline.md` / `toolpool-final.log` / `report-toolpool.md`

### 修改
- `pkg/interfaces.go`
  - `Config` 新增 `UnmarshalKey`
  - `ToolRegistry` 新增 `ListHookNames`
  - `Agent` 新增 `ListTools` / `ListHooks` / `ExecuteTool`（诊断/测试用）
- `internal/config/config.go` — 实现 `UnmarshalKey`
- `internal/config/defaults.go` — 新增 `hooks.audit_log.enabled`(默认 true)、`hooks.git_auto_commit.enabled`(默认 false)
- `internal/build.go` — 注册 Skill 工具、按配置连接 MCP、注册内置 Hook；`agentWithCleanup` 管理 MCP 生命周期
- `cmd/supcode/main.go` — `BuildAgent` 返回值若实现 `io.Closer` 则 `defer Close()`
- `internal/agent/loop.go` — 新增只读诊断方法
- `internal/tools/registry.go` — 钩子链加入每钩子超时与 ctx 取消检测；`AfterTool` 错误记录日志（原为静默吞掉）；新增 `ListHookNames`、`NewRegistryWithHookTimeout`
- `internal/skill/manifest.go` — `ParseManifest` 容忍 UTF-8 BOM（见 §4）
- 各测试 mock：`internal/agent/mock_llm_test.go`、`internal/subagent/mock_orchestrator_test.go`、`internal/tools/mock_registry_test.go`、`internal/mcp/bridge_test.go` — 同步实现 `ListHookNames`

### 删除
- `internal/hooks/engine.go`、`internal/hooks/engine_test.go`（功能与 `Registry.Execute` 内联链重复）

## 3. 验收命令与结果

执行环境：`D:\codes\sup-code`，`go 1.25`，运行前清空 `SUPCODE_LLM_API_KEY`。

```
go build ./...      # 通过
go vet ./...        # 无告警
go test ./...       # 见下方对照
```

按包结果与 Phase 0 基线对照：

- 所有原通过包仍然通过。
- 新增包 `internal/tools/skilltool` 通过。
- `internal/hooks` 因删除唯一的测试文件不再出现在结果中（预期，非失败）。
- **未新增任何失败**。剩余失败与基线完全一致，共 4 项：
  - `internal/TestNewSupCode_MissingAPIKey`（过时测试：API key 校验已按设计推迟到 Agent 层）
  - `internal/config/TestMissingAPIKey`（同上）
  - `internal/config/TestDefaultValues/llm.provider`（本机真实用户配置 `~/.supcode/config.yaml` 污染）
  - `internal/cli/TestConfigGetExistingKey`（同上）

关键新增测试（全部通过）：
- `internal/mcp`：`TestBridge_UnifiedToolPool_WithRealRegistry`、`TestConnectConfigured_GracefulDegradation`
- `internal/tools/skilltool`：8 个用例（list/load/错误分支/ctx 取消/Schema）
- `internal`：`TestBuildAgent_*`（MCP 降级/关闭、Skill list、Hook 默认策略）、`TestAuditHook_EndToEnd_RecordsToolCall`
- `internal/tools`：`TestRegistry_Execute_BeforeToolHook_TimesOut`、`TestRegistry_Execute_ContextCancelled_DuringHookChain`、`TestRegistry_Execute_AfterToolHookError_DoesNotAffectResult`
- `internal/skill`：`TestParseManifestWithUTF8BOM`
- `internal/config`：`TestManager_UnmarshalKey_MCPServers`、`TestManager_UnmarshalKey_MissingKey`

## 4. 过程中发现并修复的既有缺陷

1. **Skill 功能实际不可用**：仓库自带 `skills/code-review/skill.json`、`skills/test-generator/skill.json`
   均带 UTF-8 BOM（`EF BB BF`），`json.Unmarshal` 直接报
   `invalid character 'ï' looking for beginning of value`。已在 `ParseManifest` 中剥离 BOM，
   使内置技能可被正常发现/加载。
2. **AfterTool 钩子错误被静默吞掉**：原 `Registry.Execute` 完全不检查 `hook.AfterTool` 的返回值。
   现改为记录日志（`[tools] AfterTool hook ... error`），并在新超时/取消保护下运行。

## 5. 本任务范围之外（未改动，明确记录）

- Agent Loop 流式输出未打通到 TUI（`InteractionService.StreamResponse` 仍未接线）
- 同轮对话内并行工具调用不支持（`ToolSelector` 只取首个 `tool_call`）
- TUI 的 Ctrl+C 未接入可取消 `ctx`
- `docs/产品使用说明书.md` 提及的 `supcode mcp list` 子命令尚不存在（文档/实现不一致，属既有问题）
- `internal/mcp/discovery.go` 的 `DiscoverFromDirectory` 仍为占位实现；MCP 接入采用
  `mcp.servers` 配置直读（viper `UnmarshalKey`），与产品文档第 11 节一致，不依赖目录发现

## 6. 配置示例（改造后可用）

```yaml
# ~/.supcode/config.yaml
mcp:
  servers:
    - name: filesystem
      transport: stdio
      command: npx
      args: ["-y", "@modelcontextprotocol/server-filesystem", "./mcp-test"]

hooks:
  audit_log:
    enabled: true      # 默认 true
  git_auto_commit:
    enabled: false     # 默认 false
```

Skill 无需配置：`BuildAgent` 会扫描 builtin(`./skills`)、user(`~/.supcode/skills`)、
project 三级目录，并注册一个名为 `skill` 的工具供 LLM 按需 `list`/`load`。
