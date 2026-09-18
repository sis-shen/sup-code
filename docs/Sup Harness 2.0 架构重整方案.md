# Sup Harness 2.0 架构重整方案

> 副标题：基于 Cordis 内核的「Everything is a Plugin」重构
> 文档版本：v1.0（规划稿，不含代码改动）
> 基线代码：`D:\codes\sup-code`（SupCode v1.0，Go 1.22，生产代码约 8,937 行）
> 参考体系：Cordis（cordiverse/cordis）、DeepSeek Harness（deepseek-ai/deepseek-harness，以下简称 dsh）
> 目标产物：Sup Harness（命令行 `sup`，守护进程 `supd`），`go install .../cmd/sup@latest`

---

## 0. 执行摘要

### 0.1 一句话目标

把 SupCode 从「五层硬编码装配的单体 CLI」重构为「**微内核 + 全插件**」的 Sup Harness：内核只保留 Cordis 五要素（Context / Service / Plugin / inject / 可逆副作用），其余能力（LLM、工具、Agent 循环、记忆、权限、TUI、MCP、Skill、SubAgent……）全部作为插件挂载；在**尽量复用 v1 Go 代码**的前提下，保持全部 ReAct 能力，并对齐 dsh/Cordis 的服务键与事件语义以获得生态兼容性。

### 0.2 已确认的技术路线

| 决策项 | 结论 |
|---|---|
| 内核/运行时 | **方案 A：Go 原生 Cordis 语义内核**（非字面 vendored TypeScript Cordis） |
| 兼容策略 | 语义对齐 dsh/Cordis（服务键、事件模式、effect 可逆、作用域），并额外提供跨运行时桥接 |
| 组织范式 | **Everything is a Plugin**：内核之外无特权模块 |
| 代码复用 | 优先"原样搬运 + 薄适配"，禁止顺带重写业务逻辑 |
| 目标复用率 | 直接搬运约 **50%~55%**，含轻量适配的综合复用率约 **83%**（见 §9） |

### 0.3 为什么不是直接把 Cordis 引进 Go 项目

Cordis 与 dsh 是 TypeScript；SupCode v1 是 Go。cross-language 直接 vendor 不可行。业界已有先例证明 Cordis 的**语义**可以移植到非 TS 语言：`dshbox/cordis-rs`（Rust 移植，明确标注 "the plugin framework at the core of DeepSeek Harness"）、`eavae/cordis-rs`、`y0usaf/cordis-rs`。因此本项目选择**在 Go 中实现一套与 Cordis 语义等价的轻量内核**，同时通过标准化插件协议与 Node 侧车与 dsh 生态互通。

---

## 1. 背景与目标

### 1.1 v1 现状（SupCode 1.0）

v1 采用自上而下五层架构，层间通过 `pkg` 接口解耦，组装根硬编码在 `internal/build.go`：

```
第1层 交互层  TUI / Slash Command / Skill 加载 / 多轮对话
第2层 引擎层  Agent Loop / LLM Client / SubAgent / Plan Mode
第3层 工具层  六大内置工具 / MCP 外部工具 / Hook 钩子
第4层 记忆层  上下文压缩 / Token 管理 / 长期记忆
第5层 安全层  五层权限防御 / Worktree 隔离 / 审计日志
```

现有生产代码规模（按包统计，**不含 `_test.go`**）：

| 包 | 文件数 | 行数 | 包 | 文件数 | 行数 |
|---|---:|---:|---|---:|---:|
| `pkg`（接口/类型/错误） | 3 | 439 | `internal/mcp` | 7 | 738 |
| `internal`（组装根） | 3 | 176 | `internal/memory` | 2 | 383 |
| `internal/agent` | 5 | 804 | `internal/permission` | 4 | 568 |
| `internal/cli` | 4 | 276 | `internal/skill` | 6 | 566 |
| `internal/config` | 2 | 161 | `internal/subagent` | 2 | 262 |
| `internal/contextmgr` | 4 | 514 | `internal/tools`（含内置） | 8 | 1,271 |
| `internal/depslock` | 1 | 5 | `internal/tui` | 6 | 1,116 |
| `internal/hooks/builtin` | 2 | 135 | `internal/worktree` | 1 | 327 |
| `internal/llm`（含 4 适配器） | 6 | 1,027 | `cmd/supcode` | 1 | 169 |
| **合计** | **66** | | | | **8,937** |

### 1.2 v1 的结构性痛点

| # | 痛点 | 表现 | 2.0 要解决的方式 |
|---|---|---|---|
| P1 | 组装根硬编码 | `build.go` 手工 `new` 全部组件；新增一种工具/供应商要改组装根 | 插件自注册 + `inject` 依赖解析，内核按声明装配 |
| P2 | 扩展必须改代码 | 无运行时插拔，第三方能力要 fork 主仓库 | 插件清单 + 动态/进程外插件协议 |
| P3 | 依赖靠构造函数注入 | 组件间通过结构体字段点对点连接，无法声明式表达 | Context + Service Key + `inject` |
| P4 | 无作用域/生命周期撤销 | SubAgent、会话无法隔离挂载/卸载插件；资源释放靠 defer | Cordis Fiber/scope + `ctx.effect` 可逆副作用 |
| P5 | 通信靠直接调用 | 权限/Hook/审计/流式输出耦合在调用链，无法横向拦截 | 类型化事件（emit/waterfall/parallel/serial/bail） |
| P6 | 两套 Hook 历史遗留（已在 v1 task 修复一半） | 审计/Git Hook 曾未接入主链路 | 全部 Hook 以插件 `ctx.on` 注册到统一事件流水线 |
| P7 | 难以对标生态 | 无法复用 dsh 插件、MCP 之外无生态入口 | 对齐 dsh 服务键/事件 + 跨运行时桥 |

### 1.3 目标（Goals）

- G1 **Everything is a Plugin**：除内核外，所有能力以插件形式存在，插件可热插拔、可替换、可独立测试。
- G2 **ReAct 能力零丢失**：v1 的全部能力（Agent 循环、Plan Mode、工具池、Hook、记忆、权限、SubAgent、Worktree、TUI/CLI、多供应商 LLM、MCP、Skill）在 2.0 全部保留（§6 矩阵逐项对账）。
- G3 **最大化复用 v1 代码**：以"搬运 + 薄适配"为主，禁止借重构之名重写业务逻辑（§9 量化）。
- G4 **dsh/Cordis 生态兼容**：服务键、事件模式、effect 语义、插件清单对齐 dsh；提供跨运行时插件桥（§7）。
- G5 **保持终端原生优势**：仍为单二进制、纯 CLI/TUI、SSH 可用；插件加载不引入强制 Node 依赖（Node 仅在跨运行时插件时需要）。

### 1.4 非目标（Non-Goals）

- 不追求与 Cordis/TypeScript 的**二进制级**兼容（跨语言不可行），只承诺协议与语义级互通。
- 不在本次重构中新增业务功能（如并行工具调用、TUI 流式渲染打通可作为 2.1 迭代）。
- 不引入重型依赖（如完整工作流引擎、Web UI），保留后续以插件方式扩展的空间。

---

## 2. Cordis 内核语义与 Go 映射

### 2.1 Cordis 五大核心概念（摘自 dsh 官方 primer）

| 概念 | 含义 |
|---|---|
| **Plugin** | 实现 Service 的对象；可以是带 `inject`/`apply(ctx)` 的函数，也可以是 Service 子类；生命周期由框架挂载到当前上下文 |
| **Context** | 服务容器。一个服务占据稳定的 `ctx.<key>`（如 `ctx.tools`、`ctx.llm`）；消费者按 key 查找，而非 import 具体实现 |
| **inject** | 声明服务依赖；插件等待依赖就绪后才启动；加载顺序由依赖表达而非手工编排 |
| **类型化事件** | `emit` / `waterfall` / `parallel` / `serial` / `bail` 五种分发模式 |
| **可逆副作用** | `ctx.effect()` / `ctx.on()` 安装注册，reload/teardown 时按预期撤销 |

事件分发模式精确语义：

| 模式 | 是否 await | 顺序 | 返回值 | 用途 |
|---|---|---|---|---|
| `emit` | 否 | 按注册顺序观察 | 无 | 通知/观察（审计、遥测） |
| `waterfall` | 否 | 环绕中间件，`(...args, next)` | 有 | 包装/改写请求（工具执行流水线、审批） |
| `parallel` | 是 | 全部并行 | 无 | 扇出 |
| `serial` | 是 | 按序 | 有 | 串行决策 |
| `bail` | 否 | 按序到首个 bail 值 | 有 | 首个短路优先（策略/选择） |

### 2.2 Go 侧内核 API 草案

> 包路径拟定：`github.com/supcode/sup-harness/core`（可简记为 `core`）。完整接口见附录 A。

```go
// Context：服务容器 + 事件总线 + 作用域 + 可逆副作用
type Context struct {
    kernel  *Kernel
    parent  *Context      // 作用域父节点（对应 Cordis Fiber）
    scope   *Scope
    services map[string]any
    disposers []Disposer
}

// 声明式插件
type Plugin struct {
    Name    string
    Inject  []string               // 依赖的服务键
    Config  func() any             // 默认配置类型（供 loader 生成 schema）
    Apply   func(ctx *Context) error
}

// 服务注册（占据稳定 key）
func Provide[T any](ctx *Context, key string, impl T) Disposer

// 服务消费（消费前需在 Inject 中声明）
func Use[T any](ctx *Context, key string) T

// 可逆副作用：fn 的返回值注册为 disposer；插件卸载/重载时自动回滚
func (c *Context) Effect(fn func() (Disposer, error)) error

// 事件注册，返回解绑函数
func (c *Context) On(event string, h Handler, opts ...EventOption) Disposer

// 五种分发
func Emit(ctx *Context, event string, args ...any)
func Waterfall[Req, Resp any](ctx *Context, event string, req Req, ...) (Resp, error)
func Parallel(ctx *Context, event string, args ...any) error
func Serial[Resp any](ctx *Context, event string, args ...any) (Resp, error)
func Bail[Resp any](ctx *Context, event string, args ...any) (Resp, bool, error)

// 作用域：为 session / subagent / agent 派生隔离上下文
func (c *Context) Fork(name string) *Context
func (c *Context) Isolate() *Context

// 生命周期
func (k *Kernel) Load(ctx *Context, p Plugin) error
func (k *Kernel) Unload(ctx *Context, name string) error
func (k *Kernel) Reload(ctx *Context, name string) error
```

**Go 适配要点：**

| Cordis/TS 机制 | Go 侧实现 |
|---|---|
| `ctx.serviceName` 属性访问 | `core.Use[T](ctx, "tools")`（泛型 + 显式 key，编译期类型安全） |
| `inject` 依赖等待 | 内核维护 `pending → ready` 状态机，依赖就绪后拓扑启动 |
| `ctx.effect()` 可逆副作用 | 返回 `Disposer`，由 `Scope` 逆序回放 |
| `Fiber`/`Scope` 继承 | `Context.Fork/Isolate`，服务解析先查自身再沿父链上溯 |
| `waterfall` 环绕中间件 | `Waterfall` 泛型函数，next 闭包链 |
| Loader `!!js` 配置表达式 | Go 侧用受限表达式求值（或 YAML 变量插值），不引入 JS 引擎 |
| 热重载 | 基于 `Scope` 的卸载/重装 + 文件监听（`internal/depslock` 已具备依赖锁雏形） |

### 2.3 与 cordis-rs 的对照

`cordis-rs` 已证明以下映射成立，Sup Harness 沿用同一套语义：

- Service 通过 trait object + key 注册，`inject` 由运行时解析；
- `effect` 用 RAII/Disposer 表达，析构即撤销；
- 事件五模式原样保留；
- 作用域（scope）承载每个 agent/session 的独立服务实例。

差异点：Go 无 `async/await` 单线程事件循环，需显式处理 goroutine 并发与 `context.Context` 取消；事件监听器默认在调用方 goroutine 同步执行，`Parallel` 内部才 fan-out。

---

## 3. Sup Harness 总体架构

### 3.1 微内核 + 插件图

```
                         ┌──────────────────────────── Sup Harness 2.0 ────────────────────────────┐
                         │                        Kernel / core（唯一非插件）                         │
                         │   Context · Service Registry · Event Bus · Loader · Scope · Effect        │
                         │              Plugin Manifest · Config · Lifecycle · Disposer               │
                         └───┬──────────┬──────────┬──────────┬──────────┬──────────┬──────────┬─────┘
              ctx.config ────┘  ctx.llm │ ctx.tools│ ctx.agent│ctx.session│ctx.memory│ctx.permis│  ...
                                       │          │          │          │          │          │
              ┌────────────────────────┴──┐   ┌───┴──────┐ ┌─┴────────┐ ┌┴────────┐ ┌┴────────┐ ┌┴────────────┐
              │ plugin-llm (seam)         │   │plugin-   │ │plugin-   │ │plugin-  │ │plugin-  │ │plugin-      │
              │  ├ plugin-llm-openai      │   │tools     │ │agent     │ │session  │ │memory   │ │permission   │
              │  ├ plugin-llm-anthropic   │   │ ├ read   │ │(ReAct)   │ │(SQLite) │ │(压缩+   │ │(5层防御+    │
              │  ├ plugin-llm-deepseek    │   │ ├ write  │ │ ├plan    │ │         │ │ 长期)   │ │ 审计)       │
              │  └ plugin-llm-ollama      │   │ ├ edit   │ │ ├select  │ │         │ │         │ │             │
              └───────────────────────────┘   │ ├ bash   │ │ └correct │ │         │ │         │ │             │
                                              │ ├ glob   │ └──────────┘ └─────────┘ └─────────┘ └─────────────┘
              plugin-mcp · plugin-skill · plugin-hooks · plugin-subagent · plugin-worktree
              plugin-tui · plugin-cli · plugin-command · plugin-dsh-bridge · plugin-mcp-server
```

**内核契约（唯一特权的部分）：**
1. 维护服务注册表与 `Context` 树；
2. 解析插件 `inject` 依赖并按拓扑加载；
3. 提供事件总线（五种分发模式）；
4. 管理 `Scope` 生命周期与可逆副作用回放；
5. 加载插件清单、合并配置、日志。

**约束：内核代码不得 import 任何业务包**；所有 `internal/*` 能力只能通过插件与服务被内核感知。

### 3.2 目录结构（拟定）

```
sup-harness/
├── core/                      # 内核（新增，唯一非插件）
│   ├── context.go  service.go  event.go  scope.go  effect.go  loader.go  plugin.go
├── plugin/                    # 内置插件（全部是插件）
│   ├── config/                # 配置服务（基于 v1 internal/config）
│   ├── llm/                   # LLM seam + 适配器子插件
│   ├── tools/                 # 工具池 seam
│   │   ├── readfile/ writefile/ editfile/ bash/ glob/ grep/
│   ├── agent/                 # ReAct 循环插件（v1 internal/agent 迁移）
│   ├── session/               # 会话管理（v1 tui/session.go 迁移）
│   ├── contextmgr/            # 上下文压缩（v1 internal/contextmgr）
│   ├── memory/                # 长期记忆（v1 internal/memory）
│   ├── permission/            # 权限引擎（v1 internal/permission）
│   ├── skill/  mcp/  hooks/  subagent/  worktree/
│   ├── tui/  cli/  command/
│   ├── dshbridge/             # Node 侧车桥（新增）
│   └── mcpserver/             # 把 Harness 暴露为 MCP server（新增）
├── cmd/
│   ├── sup/main.go            # 面向用户 CLI/TUI（替代 cmd/supcode）
│   └── supd/main.go           # 常驻 Harness 守护（可选，供远程/多会话）
├── pkg/                       # 公共 DTO 与协议（v1 pkg 迁移）
├── docs/
└── go.mod
```

### 3.3 架构对比（v1 vs 2.0）

```
【v1 分层单体】                          【2.0 微内核 + 插件图】
  TUI/CLI                                  cmd/sup ──▶ Kernel(loader)
     │ 直接持有 Agent                        │
  Agent(loop/planner/selector)             ctx.load(agent) ─ inject(llm, tools,
     │ new 具体实现                                        contextmgr, session, permission)
  ToolRegistry ── tool/*, mcp, skill       ctx.tools.Register(插件贡献)
     │ 内联 Hook 循环                        tools/pre-execute ─▶ permission 插件
  ContextManager / Memory / Permission      tools/execute  ─▶ tool 插件
     │ 构造注入                              tools/post-execute ─▶ audit/git 插件
  build.go 手工 new 全部                   ctx.effect 逆序卸载；scope 支持会话/子代理隔离
```

| 维度 | v1 SupCode | 2.0 Sup Harness |
|---|---|---|
| 装配方式 | `build.go` 硬编码 `new` | 插件清单 + `inject` 拓扑加载 |
| 依赖表达 | 构造函数参数 | 服务键 + `inject` 声明 |
| 扩展方式 | 改代码/重新编译 | 编译期插件注册 / 进程外插件 / Node/dsh 插件 |
| 生命周期 | `defer` + 手工释放 | `Scope` + `Disposer` 逆序回放，支持 reload |
| 作用域 | 全局单例为主 | Context 树；session/agent/subagent 各自 Fork |
| 组件通信 | 直接方法调用 | 类型化事件（5 模式）+ 服务方法 |
| 可替换实现 | 接口 + 手工注入 | 按 key 覆盖实现，插件级替换 |
| 配置 | 全局 viper | 内核配置服务 + 每插件配置树 |
| 热插拔 | 不支持 | 支持（Go 编译期插件内部热重载；外部插件支持装卸） |
| 生态 | 无 | 对齐 dsh/Cordis，跨运行时桥 |
| 单二进制 | 是 | 是（跨运行时插件才需要 Node） |

---

## 4. Everything is a Plugin：插件清单

### 4.1 服务键（与 dsh 对齐）

| 服务键 | 角色 | v1 对应 | 说明 |
|---|---|---|---|
| `config` | core | `internal/config` | 配置服务 |
| `llm` | seam | `internal/llm` | 供应商无关流式补全；适配器作为子插件注册 |
| `tools` | core | `internal/tools/registry.go` | 工具注册与统一执行入口（含流水线） |
| `agent` | core | `internal/agent` | ReAct 循环 + Plan Mode |
| `agents` | core | （部分）`subagent` | 实时 Agent 句柄/工厂（对应 dsh `ctx.agents`） |
| `sessions` | core | `internal/tui/session.go` | 会话与持久化事件 |
| `context` | core | `internal/contextmgr` | 上下文构建/压缩/Token 计量 |
| `memory` | seam | `internal/memory` | 长期记忆与检索 |
| `permission` | core | `internal/permission` | 权限决策与审计 |
| `skills` | seam | `internal/skill` | 技能目录与加载 |
| `mcp` | seam | `internal/mcp` | MCP 客户端/传输 |
| `subagents` | seam | `internal/subagent` | 子代理编排/传输 |
| `worktree` | seam | `internal/worktree` | Git Worktree 隔离 |
| `interaction` | seam | `internal/tui` 抽象 | 面向用户的交互（TUI/headless/ACP） |
| `commands` | core | `internal/cli` | 斜杠命令注册（不经过模型） |
| `hooks` | — | `internal/hooks/builtin` | 以事件监听器形式存在，无独立服务键 |

> 服务键命名与 dsh 的 `ctx.llm` / `ctx.tools` / `ctx.sessions` / `ctx.agents` / `ctx.skills` / `ctx.approval` 等保持一致，便于后续协议互通。

### 4.2 插件清单

| 插件 | inject | provides | 贡献的事件监听 | 对应 v1 |
|---|---|---|---|---|
| `plugin-config` | — | `config` | — | `internal/config` |
| `plugin-llm` | `config` | `llm` | `llm/register-adapter` | `internal/llm/factory.go`, `fallback.go` |
| `plugin-llm-{openai,anthropic,deepseek,ollama}` | `llm` | — | — | `internal/llm/*` |
| `plugin-tools` | `permission` | `tools` | `tools/pre-execute`, `tools/execute`, `tools/post-execute`, `tools/result` | `internal/tools/registry.go` |
| `plugin-tool-{readfile,writefile,editfile,bash,glob,grep}` | `tools` | — | — | `internal/tools/*` |
| `plugin-agent` | `llm, tools, context, sessions, permission` | `agent`, `agents` | `agent/*` | `internal/agent/*` |
| `plugin-session` | `config` | `sessions` | — | `internal/tui/session.go` |
| `plugin-context` | `llm, sessions` | `context` | — | `internal/contextmgr/*` |
| `plugin-memory` | `config, llm` | `memory` | — | `internal/memory/*` |
| `plugin-permission` | `config, interaction` | `permission` | `tools/pre-execute` | `internal/permission/*` |
| `plugin-skill` | `tools` | `skills` | — | `internal/skill/*` + `tools/skilltool` |
| `plugin-mcp` | `tools, config` | `mcp` | — | `internal/mcp/*` |
| `plugin-hooks-audit` | `permission` | — | `tools/post-execute`（observing，emit） | `internal/hooks/builtin/audit.go` |
| `plugin-hooks-git` | `worktree` | — | `tools/post-execute`（waterfall，可选改写） | `internal/hooks/builtin/git_commit.go` |
| `plugin-subagent` | `agents, sessions, tools` | `subagents` | `agent/child-created` | `internal/subagent/*` |
| `plugin-worktree` | `config` | `worktree` | — | `internal/worktree/*` |
| `plugin-tui` | `sessions, interaction` | `interaction` | `llm/chunk`, `agent/state` | `internal/tui/*` |
| `plugin-cli` | `commands, agent, config` | — | — | `internal/cli/*` |
| `plugin-dsh-bridge`（新） | `tools, llm, agents` | `dsh` | 双向映射 | — |
| `plugin-mcp-server`（新） | `agent, tools` | `mcpServer` | — | — |

### 4.3 插件依赖图（拓扑）

```
config
  └─ llm ──────────────┐
       ├─ llm-openai/… │
       └─ context ◀────┼─ sessions
                       │
tools ◀── permission ◀─┘
  ├─ tool-read/write/edit/bash/glob/grep
  ├─ skills ─ skill
  └─ mcp
agent ── inject(llm, tools, context, sessions, permission)
  ├─ subagents
  └─ worktree
interaction ◀─ tui / cli / command
hooks-audit / hooks-git ── on(tools/post-execute)
dsh-bridge ── llm/tools/agents 双向
mcp-server
```

---

## 5. ReAct 能力保全矩阵

> 目标：v1 的每一项能力都能在 2.0 找到明确落点，**逐项对账，防止重构丢能力**。

| # | ReAct 能力 | v1 位置 | 2.0 落点 | 形态 |
|---|---|---|---|---|
| C1 | Agent 主循环状态机 | `internal/agent/loop.go` | `plugin-agent` + `ctx.agent.Run` | 服务方法 |
| C2 | 任务规划（Plan） | `internal/agent/planner.go` | `plugin-agent` 内部子服务 | 服务方法 |
| C3 | 工具选择（function calling） | `internal/agent/selector.go` | `plugin-agent`；可被插件用 `agent/select` 事件覆盖 | 服务 + 事件 |
| C4 | 自我修正/重试退避 | `internal/agent/corrector.go` | `plugin-agent`；`agent/step-failed` 事件 | 服务 + 事件 |
| C5 | Plan Mode（先计划后执行） | `internal/agent/plan_mode.go` | `plugin-agent` + `ctx.planMode` 折叠状态 | 服务 + 事件 |
| C6 | 多供应商 LLM | `internal/llm/*` | `plugin-llm` + 适配器插件 | seam + 插件 |
| C7 | 流式输出 | `StreamEvent`/`Chat` | `llm/chunk` 事件（emit） | 事件 |
| C8 | 统一工具池 | `internal/tools/registry.go` | `plugin-tools`（`ctx.tools`） | core 服务 |
| C9 | 六大内置工具 | `internal/tools/*` | 六个工具插件 | 插件 |
| C10 | 工具 Hook 链 | registry 内联钩子 | 统一 `tools/*` waterfall 事件 | 事件 |
| C11 | 审计日志 | `hooks/builtin/audit.go` | `plugin-hooks-audit` 监听 `tools/result` | 插件 |
| C12 | Git 自动提交 | `hooks/builtin/git_commit.go` | `plugin-hooks-git` 监听 `tools/post-execute` | 插件 |
| C13 | MCP 外部工具 | `internal/mcp/*` | `plugin-mcp`，工具贡献到 `ctx.tools` | 插件 |
| C14 | Skill 技能包 | `internal/skill/*` + `tools/skilltool` | `plugin-skill` + `tool-skill` | 插件 |
| C15 | 上下文压缩/Token | `internal/contextmgr/*` | `plugin-context`（`ctx.context`） | 插件 |
| C16 | 长期记忆/检索 | `internal/memory/*` | `plugin-memory` | seam |
| C17 | 五层权限防御 | `internal/permission/*` | `plugin-permission` 在 `tools/pre-execute` 短路 | 插件 + 事件 |
| C18 | 用户确认回调 | `ConfirmCallback` | `approval/request` waterfall（对齐 dsh） | 事件 |
| C19 | Worktree 隔离 | `internal/worktree/*` | `plugin-worktree` | seam |
| C20 | SubAgent 并行编排 | `internal/subagent/*` | `plugin-subagent` + `ctx.Fork()` 作用域 | seam + scope |
| C21 | 会话管理/持久化 | `internal/tui/session.go` | `plugin-session`（SQLite） | 插件 |
| C22 | TUI 交互 | `internal/tui/*` | `plugin-tui`（`interaction` seam） | 插件 |
| C23 | Slash 命令 | `internal/cli/*` | `plugin-command` + `ctx.commands` | 插件 |
| C24 | 配置管理 | `internal/config/*` | `plugin-config` + 每插件配置树 | 插件 |
| C25 | 公共 DTO/错误 | `pkg/*` | `pkg` 保留；接口适配为服务契约 | 库 |

**结论：C1~C25 全部有落点，无能力丢失。** 其中 C8/C10/C17/C18 由"内联调用"升级为"事件流水线"，是 2.0 的主要增强而非替换。

---

## 6. dsh 生态兼容设计

### 6.1 对齐点（零成本对齐）

| dsh/Cordis | Sup Harness 2.0 |
|---|---|
| 服务键 `ctx.llm` / `ctx.tools` / `ctx.sessions` / `ctx.agents` / `ctx.skills` | 同名服务键 |
| 事件模式 emit/waterfall/parallel/serial/bail | 同语义 API |
| `tools/pre-execute` → guard → `tools/execute` → `tools/post-execute` → `tools/result` | 工具流水线同名同序 |
| `approval/request` waterfall | 权限确认同名事件 |
| `ctx.effect()` / `ctx.on()` 可逆注册 | `ctx.Effect` / `ctx.On` + Disposer |
| 插件 `inject` + `apply(ctx)` + 配置 | `Plugin{Name, Inject, Apply, Config}` |
| agent/scope 作用域 | `Context.Fork/Isolate` |

### 6.2 跨运行时插件协议（新增规范）

由于 Go 无法加载 TS 插件，定义 **Sup Plugin Protocol（SPP）v1**：JSON-RPC 2.0 over stdio/WebSocket，语义镜像 Cordis：

```
initialize(manifest)            → {services, configSchema}
provide(key, valueRef)          → ack               // 服务注册
consume(key)                    → serviceRef        // 依赖获取
emit(event, args)               → ack
waterfall(event, req)           → resp              // 往返
effect.register(id, desc)       → ack
effect.dispose(id)              → ack
shutdown()                      → ack
```

### 6.3 三种插件形态

| 形态 | 加载方式 | 适用 | 语言 |
|---|---|---|---|
| **编译期插件**（默认） | `init()` 注册 + `cmd/sup/plugins.go` 引用 | 内置能力、性能敏感 | Go |
| **进程外插件** | Subprocess + SPP（stdio） | 第三方、崩溃隔离、跨语言 | 任意 |
| **dsh 侧车插件** | Node 侧车运行 Cordis，经 SPP 桥接 | 直接复用 dsh 生态插件 | TypeScript |
| **WASM 插件**（可选，未来） | wazero 沙箱 | 不可信插件 | 任意→WASM |

> 不使用 Go 标准库 `plugin`（`.so`）作为主方案：仅 Linux/macOS 支持、版本强耦合、跨平台差。仅在"高性能本地插件"场景作为可选后端。

### 6.4 双向兼容

1. **Sup 作为 dsh 插件运行**：`plugin-mcp-server` 把 Sup Harness 暴露为 MCP server；dsh 侧用 MCP 接入即可调用 Sup 的 agent/tools。
2. **dsh 插件在 Sup 中运行**：`plugin-dsh-bridge` 启动 Node 侧车，加载 `dsh-plugin` 包，映射：
   - dsh `ctx.tools.register` ↔ Sup `ctx.tools.Register`
   - dsh `ctx.llm` provider ↔ Sup `ctx.llm` seam
   - dsh `tools/*` 事件 ↔ Sup 同名事件（waterfall 往返由 SPP 承载）
3. **清单兼容**：支持 `package.json` 的 `dsh` 字段用于识别 dsh 插件；Sup 原生插件使用 `sup.plugin.json`（字段与 Cordis 插件声明同构）。

### 6.5 能力 seams 映射（v1 ↔ dsh）

| Sup 服务键 | dsh `ctx.*` | 备注 |
|---|---|---|
| `llm` | `ctx.llm` | 适配器注册模式一致 |
| `tools` | `ctx.tools` | 流水线一致 |
| `sessions` | `ctx.sessions` | dsh 为仅追加事件流，Sup v1 为 SQLite 会话，2.0 逐步向事件流靠拢 |
| `context` | `ctx.systemPrompt` + `ctx.compaction` | dsh 拆得更细，2.0 保留 `context` 聚合，后续可拆 |
| `skills` | `ctx.skills` | 一致 |
| `subagents` | `ctx.subagents` | 一致 |
| `permission` | `ctx.approval` + `ctx.permissionPresets` | 事件名对齐 `approval/request` |
| `worktree` | `ctx.workspace` | 语义相近，后续对齐 |
| `mcp`（客户端） | dsh 以插件形式集成 MCP | — |

---

## 7. 复用率评估

### 7.1 分类标准

| 类别 | 定义 | 计数 |
|---|---|---|
| **D 直接复用** | 文件基本原样移动，仅改 import/包名，逻辑零改动 | 按 90%~100% 计 |
| **A 适配复用** | 逻辑保留，但接口/装配/生命周期需薄适配 | 按 70%~90% 计 |
| **R 重写** | 由内核或新架构取代 | 按 10%~40% 计 |

### 7.2 模块级复用率表

| 模块 | v1 行数 | 类别 | 综合复用率 | 复用行数 | 迁移方式 |
|---|---:|---|---:|---:|---|
| `internal/llm`（含 4 适配器） | 1,027 | D | 90% | 924 | `plugin-llm` + 适配器子插件，`Chat/Models` 逻辑不动 |
| `internal/tools`（含 6 内置） | 1,271 | D | 90% | 1,144 | `plugin-tools` + 工具插件；仅注册方式改为插件贡献 |
| `internal/mcp` | 738 | D | 88% | 649 | `plugin-mcp`，传输/JSON-RPC 逻辑不动 |
| `internal/permission` | 568 | D | 88% | 500 | `plugin-permission`，挂到 `tools/pre-execute` |
| `internal/skill` | 566 | D | 88% | 498 | `plugin-skill` + `tool-skill` |
| `internal/memory` | 383 | D | 90% | 345 | `plugin-memory` seam |
| `internal/worktree` | 327 | D | 90% | 294 | `plugin-worktree` seam |
| `internal/hooks/builtin` | 135 | D | 95% | 128 | 两个 Hook 插件，监听事件 |
| `internal/depslock` | 5 | D | 100% | 5 | 直接复用 |
| `internal/contextmgr` | 514 | A | 88% | 452 | `plugin-context`，对外暴露 `ctx.context` |
| `internal/agent` | 804 | A | 70% | 563 | 逻辑保留，依赖改为 `Use[T](ctx,…)` + 发事件 |
| `internal/tui` | 1,116 | A | 80% | 893 | `plugin-tui`，去掉对 Agent 的直连，改订阅事件 |
| `internal/cli` | 276 | A | 80% | 221 | `plugin-command`，命令注册到 `ctx.commands` |
| `internal/subagent` | 262 | A | 78% | 204 | `plugin-subagent` + `ctx.Fork()` 作用域隔离 |
| `internal/config` | 161 | A | 75% | 121 | `plugin-config` + 每插件配置树 |
| `pkg` | 439 | A | 90% | 395 | DTO/错误保留；接口改为服务契约 |
| `cmd/supcode` | 169 | A | 40% | 68 | 拆为 `cmd/sup` + `cmd/supd`，入口逻辑改写 |
| `internal`（build/app/wire 组装根） | 176 | R | 20% | 35 | 被内核 loader 取代 |
| **v1 小计** | **8,937** | | **83.3%** | **~7,439** | |

### 7.3 新增代码估算

| 新增项 | 估算行数 | 说明 |
|---|---:|---|
| `core` 内核（Context/Service/Event/Scope/Effect/Loader） | 1,800~2,200 | 项目核心增量 |
| `plugin-dsh-bridge`（SPP + Node 侧车桥） | 600~900 | 生态兼容 |
| `plugin-mcp-server` | 200~300 | 双向兼容 |
| 插件清单/配置/脚手架 | 300~500 | loader、manifest |
| 迁移胶水与测试补充 | 800~1,500 | 接口适配、端到端 |
| **新增合计** | **3,700~5,400** | |

### 7.4 2.0 总量与复用结论

| 指标 | 数值 |
|---|---:|
| v1 生产代码 | 8,937 |
| 复用行数（综合） | ~7,439（**83.3%**） |
| 其中"零/近零改动"直接搬运 | 约 4,500~4,900（**50%~55%**） |
| 新增代码 | 3,700~5,400 |
| 2.0 预计总量 | ~12,600~14,300 |
| 需要重写/废弃的 v1 代码 | ~1,500（组装根、entry、少量耦合代码） |

> 说明：以上为工程估算（基于包级行数），实际以迁移 PR 的 `git diff --stat` 为准。测试代码由于受 `.gocache` 干扰未精确统计，预计接口不变的模块（工具/LLM/MCP/Skill/Permission）单测可复用约 70%+。

---

## 8. 迁移路线图

> 原则：**先立内核、再搬叶子、最后搬心脏（Agent）**；每一步保持主干可编译、可运行、可回归（沿用 v1 `task.md` 的验收文化）。

| 阶段 | 目标 | 主要产出 | 验收标准 |
|---|---|---|---|
| **Phase 0 基线冻结** | 固化 v1 行为与测试基线 | `docs/baseline-v1.md`；`go test ./...` 快照 | 记录已知失败项，后续不新增失败 |
| **Phase 1 内核落地** | 实现 `core`，不接业务 | Context/Service/Event/Scope/Effect/Loader + 单测 | 内核单测覆盖；用一个 demo 插件验证 inject/effect/事件五模式 |
| **Phase 2 叶子插件化** | 搬运无状态/低依赖模块 | `plugin-llm(+适配器)`、`plugin-tools(+工具)`、`plugin-mcp`、`plugin-skill`、`plugin-permission`、`plugin-memory`、`plugin-worktree`、`plugin-hooks-*` | 功能对等测试通过；`git diff --stat` 证明逻辑零改动 |
| **Phase 3 状态与交互** | 会话/上下文/交互 | `plugin-session`、`plugin-context`、`plugin-tui`、`plugin-cli`、`plugin-command` | TUI 与单发查询可用；流式输出经事件订阅渲染 |
| **Phase 4 心脏迁移** | Agent 循环成为插件 | `plugin-agent`（ReAct + Plan Mode + corrector）；`plugin-subagent` + scope | v1 全部 ReAct 场景端到端通过；能力矩阵 §5 全部勾选 |
| **Phase 5 双子进程** | 常驻与远程 | `cmd/sup`、`cmd/supd`；`plugin-mcp-server` | sup 单发/TUI；supd 可被 MCP 客户端接入 |
| **Phase 6 生态兼容** | dsh 互通 | `plugin-dsh-bridge`、SPP 规范、`sup.plugin.json` | dsh 插件经桥可注册工具并被调用；Sup 可作 dsh 工具源 |
| **Phase 7 收尾** | 去 v1 残留、性能与文档 | 删除 `internal/build.go`、`wire.go`；更新全部文档 | 无死代码；覆盖率达标；发布 2.0 |

**Phase 2 之后即可双跑对比**：同一 `sup.plugin.json` 下分别用 v1 `build.go` 与 v2 loader 装配，跑同一套集成测试，逐项对账结果一致。

---

## 9. 风险与对策

| # | 风险 | 影响 | 对策 |
|---|---|---|---|
| R1 | Cordis 是 TS 异步模型，Go 并发语义不同 | 事件顺序/取消/竞态不一致 | 内核事件默认同步、同 goroutine；`Parallel` 显式 fan-out；全部 API 带 `context.Context`；对 scope 卸载加引用计数 |
| R2 | Go 动态插件加载弱 | 第三方插件难 | 主推编译期插件 + 进程外 SPP 插件；`.so` 仅作可选；预留 WASM |
| R3 | dsh 生态兼容预期过高 | 兼容性不达标 | 明确"语义/协议兼容，非二进制"；先做工具与 LLM 两个 seam 的互通作为 MVP，再扩展 |
| R4 | 借重构重写业务逻辑 | 引入回归、复用率下降 | 每阶段以 `git diff --stat` 审计；叶子插件要求"逻辑零改动" |
| R5 | 迁移期双套装配并存 | 维护成本 | Phase 2~4 双跑只在 CI 中保留；Phase 7 强制删除 v1 组装根 |
| R6 | 事件总线性能开销 | Agent 热路径变慢 | 事件 key 预注册；热路径（工具执行）用预分配 handler 切片；benchmark 门禁 |
| R7 | 作用域泄漏/资源未释放 | 长驻 `supd` 内存增长 | `Scope` 强制 Disposer；引用计数 + `go test -race`；引入 invariants 插件做运行时校验（参考 dsh `ctx.invariants`） |
| R8 | 配置从全局变每插件 | 兼容旧配置 | `plugin-config` 提供兼容层，旧 `~/.supcode/config.yaml` 映射到新配置树 |

---

## 10. 附录

### 附录 A：内核接口草案（Go）

```go
package core

import "context"

// ── 插件 ──────────────────────────────────────────────
type Plugin struct {
    Name    string
    Inject  []string
    Config  func() any
    Apply   func(ctx *Context) error
}

// ── 上下文 ────────────────────────────────────────────
type Context struct {
    k        *Kernel
    parent   *Context
    name     string
    services map[string]any
    scopes   []*Scope
    disposers []Disposer
}

type Disposer func() error

func (c *Context) Effect(fn func() (Disposer, error)) error
func (c *Context) On(event string, h Handler, opts ...EventOption) Disposer
func (c *Context) Fork(name string) *Context
func (c *Context) Isolate() *Context

func Provide[T any](c *Context, key string, impl T) Disposer
func Use[T any](c *Context, key string) T

// ── 事件 ──────────────────────────────────────────────
type Handler func(ctx context.Context, args ...any) (any, error)

func Emit(c context.Context, k *Kernel, event string, args ...any)
func Waterfall[Req, Resp any](c context.Context, k *Kernel, event string, req Req) (Resp, error)
func Parallel(c context.Context, k *Kernel, event string, args ...any) error
func Serial[Resp any](c context.Context, k *Kernel, event string, args ...any) (Resp, error)
func Bail[Resp any](c context.Context, k *Kernel, event string, args ...any) (Resp, bool, error)

// ── 内核 ──────────────────────────────────────────────
type Kernel struct{ /* registry, services, events, plugins */ }

func New() *Kernel
func (k *Kernel) Load(ctx *Context, p Plugin) error
func (k *Kernel) Unload(ctx *Context, name string) error
func (k *Kernel) Reload(ctx *Context, name string) error
func (k *Kernel) Start(ctx context.Context) error
func (k *Kernel) Shutdown(ctx context.Context) error
```

### 附录 B：插件清单示例 `sup.plugin.json`

```json
{
  "name": "plugin-hooks-audit",
  "version": "2.0.0",
  "inject": ["permission"],
  "provides": [],
  "events": [
    { "name": "tools/result", "mode": "emit" }
  ],
  "config": {
    "enabled": { "type": "boolean", "default": true },
    "path":    { "type": "string",  "default": "~/.sup/harness/audit.log" }
  },
  "entry": "builtin:hooks-audit"
}
```

### 附录 C：工具流水线事件（与 dsh 对齐）

```
tools/pre-execute   (waterfall)  权限短路 / 参数改写 / Hook BeforeTool
        │
     单调守卫（不可重排的所有者策略）
        │
tools/execute       (waterfall)  实际调用工具（可被超时/沙箱包裹）
        │
tools/post-execute  (waterfall)  结果改写 / Git 自动提交 / Hook AfterTool
        │
tools/result        (emit)       审计 / 遥测 / 不可变结果观察
```

### 附录 D：术语表

| 术语 | 含义 |
|---|---|
| Cordis | 时空可组合性的元框架，一切皆插件，dsh 的内核 |
| dsh | DeepSeek Harness，基于 Cordis 的开源 Agent Harness |
| seam | 可替换能力缝：定义与实现分离，消费者依赖抽象服务键 |
| Fiber / Scope | Cordis 的作用域，承载隔离子树与生命周期 |
| effect | 可逆副作用，注册即获得撤销能力 |
| waterfall | 环绕中间件式事件分发，可短路、可包装 |
| SPP | Sup Plugin Protocol，跨运行时插件协议（本文新增） |

---

> 本方案为纯规划文档，未对任何代码做修改。所有复用率与工作量均为工程估算，最终以迁移 PR 的 `git diff --stat` 与回归测试为准。
