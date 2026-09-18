# ADR-0002：叶子插件契约（Phase 2）

- 状态：Accepted
- 日期：2026-09-18
- 决策者：Phase 2 编排 Agent
- 关联阶段：Phase 2

## 背景

Phase 2 要把 v1 的 8 个无状态/低依赖模块迁移为 Cordis 插件，且**业务逻辑零改动**。插件之间不得直接 import，只能通过服务键与事件通信；每个插件由 `sup.plugin.json` 声明并经 `inject` 装载。为此需要先冻结服务键、事件名与插件包的结构约定。

## 决策

### 1. 服务键（`pkg/servicekeys.go`）
| 键常量 | 值 | 适配的 v1 接口/类型 | 提供者 |
|---|---|---|---|
| `pkg.ServicePermission` | `permission` | `pkg.PermissionEngine` | plugin-permission |
| `pkg.ServiceLLM` | `llm` | `pkg.LLMClient` | plugin-llm |
| `pkg.ServiceTools` | `tools` | `pkg.ToolRegistry` | plugin-tools |
| `pkg.ServiceMCP` | `mcp` | `pkg.MCPClient` | plugin-mcp |
| `pkg.ServiceSkills` | `skills` | `*skill.SkillLoader`（具体类型） | plugin-skill |
| `pkg.ServiceMemory` | `memory` | `pkg.MemoryStore` | plugin-memory |
| `pkg.ServiceWorktree` | `worktree` | `pkg.WorktreeManager` | plugin-worktree |

`config` 键由 Phase 3 的 `plugin-config` 提供；Phase 2 各插件将配置设为**可选**（`core.MaybeUse[pkg.Config]`），不作为硬 `inject`，以免阻塞 Phase 2。

### 2. 工具流水线事件（`pkg/pipeline.go`）
```
tools/pre-execute  (waterfall)  权限短路/参数改写
tools/execute      (waterfall)  执行包装
tools/post-execute (waterfall)  结果改写/Git 自动提交
tools/result       (emit)       审计/遥测
```
- 载荷类型：`*pkg.ToolInvocation`（同一个值贯穿四个事件）。
- `plugin-tools` 提供的 `pkg.ToolRegistry.Execute` 负责按上述顺序分发事件（装饰器包装 v1 `internal/tools.Registry`，v1 逻辑不改）。
- `plugin-permission` 监听 `tools/pre-execute`，`DecisionDeny` 时返回 `pkg.ErrPermissionDenied` 短路。
- `plugin-hooks` 的 audit 监听 `tools/result`（emit），git 监听 `tools/post-execute`（waterfall），内部复用 v1 `internal/hooks/builtin` 的 Hook 对象，逻辑不改。

### 3. 插件包结构约定
每个插件目录 `plugin/<name>/` 必须导出：
```go
func Plugin(opts Options) core.Plugin // 构造可装载插件（Name/Inject/Provides/Apply）；简单插件可省略 Options
func Manifest() core.Manifest         // 元数据（与 sup.plugin.json 同构）
```
- `Apply` 通过 `core.Provide(ctx, <key>, impl)` 注册服务；副作用用 `ctx.Effect`（Close 等）；事件用 `ctx.On`。
- 同目录提供 `sup.plugin.json`（`name/version/inject/provides/events/entry`）。
- 单测放 `plugin/<name>/*_test.go`，用 `core.New()` 构造内核装载验证。

### 4. 迁移规则（硬约束）
- 只允许新增 `plugin/<name>/**`；**禁止修改 `core/`**；`internal/*` 原则不改（如需新增导出，先上报编排 Agent）。
- 允许迁移期 `plugin/*` import `internal/*`（Phase 7 收敛）；**禁止 `plugin/X` import `plugin/Y`**（G5 规则 11）。
- 业务逻辑复用 v1 实现，适配层只做装配与事件桥接。

## 后果

- 正面：插件可独立测试与装载；服务键/事件与 dsh 对齐；`internal/*` 单测继续作为对照。
- 负面/代价：迁移期双份装配（v1 `internal/build.go` 与 v2 插件）并存，Phase 7 清理。
- 对 dsh/Cordis 兼容性的影响：服务键与事件名对齐，为 Phase 6 SPP 桥接打基础。

## 后续

- Phase 3 引入 `plugin-config`，各插件从可选读取切换为 `inject config`。
- Phase 7 禁止 `plugin/*` import `internal/*`。

## 补充（2026-09-18，Phase 2 执行期）

子 Agent 在迁移中报告两处契约缺口，由编排 Agent 授权在**契约层**（非业务逻辑）补充：

1. **`core.Manifest` 增加 `Events []ManifestEvent` 与 `Config map[string]any`**：原 Manifest 无法表达 `sup.plugin.json` 的事件声明，导致 `Manifest()` 与 JSON 不能同构。新增 `ManifestEvent{Name,Mode}`。属内核契约的加法扩展，不改内核语义。
2. **`internal/memory.Store` 增加 `UpdateEmbedding`**：`Store` 原先未实现 `pkg.MemoryStore` 的 `UpdateEmbedding` 方法（接口与实现不一致），导致插件无法直接 `Provide`。补充该方法（按 ID 更新 embedding 列），使 `*memory.Store` 真正实现 `pkg.MemoryStore`；`plugin-memory` 直接提供 v1 实现，不再需要语义有损的装饰器。

此外统一了各插件 `Manifest()` 与 `sup.plugin.json` 的事件声明格式为 `[{name,mode}]`。
