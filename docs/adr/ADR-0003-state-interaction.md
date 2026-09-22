# ADR-0003：状态与交互插件契约（Phase 3）

- 状态：Accepted
- 日期：2026-09-18
- 决策者：Phase 3 编排 Agent
- 关联阶段：Phase 3

## 背景

Phase 3 把 v1 的会话、上下文、TUI/CLI、斜杠命令与配置迁移为插件，使 2.0 恢复"可用产品"形态。需要冻结服务键、交互 seam 的唯一提供者规则，以及命令注册表契约。

## 决策

### 1. 服务键（扩展 `pkg/servicekeys.go`）
| 键常量 | 值 | 类型 | 提供者 | v1 来源 |
|---|---|---|---|---|
| `pkg.ServiceConfig` | `config` | `pkg.Config` | plugin-config | `internal/config` |
| `pkg.ServiceSessions` | `sessions` | `pkg.SessionManager` | plugin-session | `internal/tui/session.go` |
| `pkg.ServiceContext` | `context` | `pkg.ContextManager` | plugin-context | `internal/contextmgr` |
| `pkg.ServiceInteraction` | `interaction` | `pkg.InteractionService` | plugin-tui **或** plugin-cli | `internal/tui/service.go` |
| `pkg.ServiceCommands` | `commands` | `pkg.CommandRegistry` | plugin-command | `internal/tui/completions.go`（CmdDef） |

- `sessions`/`context`/`interaction`/`commands` 均**恰好一个**提供者。
- `interaction` 是**可替换 seam**：`plugin-tui` 提供 TUI 实现，`plugin-cli` 提供 headless 实现；同一进程只装载其一（后者用于无终端/测试环境与单发查询）。

### 2. 命令注册表（`pkg/commands.go`）
- `pkg.SlashCommand`：`Name() / Description() / Run(ctx, sessionID, args) (CommandResult, error)`。
- `pkg.CommandRegistry`：`Register / Unregister / Get / List`。
- 约定：命令**不经模型**，由交互层本地分发；`plugin-command` 预注册内置命令（help/version/session/clear），并可被后续插件扩展。

### 3. 配置兼容（plugin-config）
- `plugin-config` 包装 v1 `internal/config.Manager`（Viper），提供 `pkg.Config`。
- 兼容 `~/.supcode/config.yaml` 与项目级 `.supcode/config.yaml`、`SUPCODE_*` 环境变量、默认值合并；`Options.Path` 可显式指定配置文件。
- API key 校验沿用 v1 设计：延迟到 Agent 层，配置本身可离线加载。

### 4. 事件与流式（范围界定）
- `plugin-tui` 消费 `interaction` seam；`llm/chunk`、`agent/state` 事件由 Phase 4 的 agent 插件发出，"流式输出打通 TUI" 与 "Ctrl+C → ctx 取消" 按规划推迟到 2.1（B2/B3）。
- 本阶段 `interaction` 实现需覆盖 `StreamResponse/RequestConfirmation/ReadInput/Notify/HandleCommand`，以便 Phase 4/5 直接接线。

## 后果

- 正面：session/context 可独立测试；interaction 可 TUI 或 headless 替换；命令不经模型、可扩展。
- 负面/代价：TUI 交互（bubbletea）难以在 CI 断言，验收以 headless 实现 + 单元测试为主，TUI 组件测试沿用 v1 `internal/tui`。
- 对 dsh/Cordis 兼容性的影响：`sessions`/`context`/`interaction`/`commands` 与 dsh 服务键语义一致。

## 后续

- Phase 4 的 `plugin-agent` 注入 `llm/tools/context/sessions/permission` 并发出 `agent/*`、`llm/chunk` 事件，由 `plugin-tui` 订阅（B2 完成后）。
- Phase 5 `cmd/sup` 装载 `plugin-tui` 或 `plugin-cli` 作为 `interaction` 提供者。

## 补充（2026-09-18，Phase 3 执行期）

子 Agent 报告的实现细节与适配（均未修改 `internal/`）：

1. **plugin-config 显式路径**：v1 `Manager.Load()` 调用 viper `SetConfigName`，会重置此前 `SetConfigFile`。故 `Options.Path` 在 `Load()` **之后**通过 `ReadInConfig` 应用；`Path==""` 时行为与 v1 完全一致。
2. **plugin-tui 具体类型**：v1 `tui.NewService` 接收具体 `*tui.SessionManager`，而 seam 类型为 `pkg.SessionManager`；插件用类型断言适配（非 v1 提供者时退回 nil）。v1 `Service` 的 `Notify`/`HandleCommand`/`StreamResponse` 无需运行 Bubble Tea 即可用；`RequestConfirmation`/`ReadInput` 需运行程序或取消 ctx。
3. **plugin-cli 默认值**：`Options.AutoConfirm bool` 零值为 false，无法表达"默认 true"；非交互调用需显式传 `true`（后续可改 `*bool`）。
4. **arch 规则 11 修正**：`plugin/cli` 的测试原直接 import `plugin/command` 以装载命令注册表，触发跨插件规则；改为测试内置一个 `pkg.CommandRegistry` 假实现，生产代码本就只经 `pkg.ServiceCommands` 交互。
5. **命令注册表**：`plugin/command` 提供全新的内存 `pkg.CommandRegistry`（v1 仅有点 `CmdDef` 补全表），内置 `help`/`version`，命令本地分发不经模型。
6. `SingleShot` 的 `sessionMgr` 参数为 Phase 4/5 预留，当前仅经 `ctxMgr` 读写上下文。
