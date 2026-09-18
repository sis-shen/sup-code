# ADR-0001：Cordis 内核 core 的 Go 语义映射

- 状态：Accepted
- 日期：2026-09-18
- 决策者：Phase 1 执行 Agent（编排 Agent 审定）
- 关联阶段：Phase 1

## 背景

Sup Harness 2.0 需要一套与 Cordis（dsh 内核，TypeScript）语义等价的 Go 微内核。Cordis 的核心概念是：**Plugin / Context / inject / 类型化事件（emit/waterfall/parallel/serial/bail）/ 可逆副作用（effect）**。跨语言无法直接 vendor，需要在 Go 中重建等价语义，并让所有非内核能力以插件形式挂载。

Phase 0 已冻结目录骨架（`core/`、`plugin/`、`cmd/sup`、`cmd/supd`）与门禁（G0–G10）。本 ADR 冻结 `core` 的对外契约，之后事件/Scope/Loader 的实现不得随意改动导出符号。

## 候选方案

| 方案 | 优点 | 缺点 |
|---|---|---|
| A. 字面移植 Cordis（TS/JS） | 生态零成本 | 跨语言不可行；引入 Node/JS 运行时 |
| B. Go 原生等价内核（本文） | 单二进制、类型安全、可测 | 需自行定义语义边界与并发模型 |
| C. 引入第三方 Go 插件框架 | 省事 | 语义不对齐 dsh/Cordis，生态互通无从谈起 |

## 决策

采用 **方案 B**。`core` 导出以下契约（单一事实来源：`core/*.go`）：

### 服务注册表（Context / Provide / Use）
- `Context` 是服务容器 + 作用域 + 生命周期。
- 非隔离作用域的服务注册进 **内核级注册表**（`Kernel.services`），使同一父作用域下加载的兄弟插件可互相解析；隔离作用域（`Isolate`）注册到本地，形成解析屏障。
- `Fork` 派生继承父服务的子作用域；隔离性沿 Fork 传播。
- `Provide` 自动把还原 disposer 登记到当前作用域，卸载即撤销服务（可逆）。
- `Use[T]` 类型化解析，缺失/类型不符时 panic；`MaybeUse[T]` 返回 `(T, bool)`。

### 事件总线五模式
| 模式 | 语义 | 实现要点 |
|---|---|---|
| `Emit` | 通知，按注册顺序，无返回 | 忽略处理结果 |
| `Waterfall` | 环绕中间件，`next` 链；可短路/包装 | 不调用 `next` 即短路；`next` 结果可再加工 |
| `Parallel` | 并发扇出并 await | goroutine + WaitGroup，`errors.Join` 汇总 |
| `Serial` | 按序，非 nil 结果作为下一环输入 | 返回最后一个非 nil 结果 |
| `Bail` | 按序，遇首个非 nil 结果短路 | 返回 `(Resp, bool, error)` |

监听器注册顺序即调用顺序；`Once()` 使监听器在首次触发后自动移除；`Context.On` 注册的监听器随作用域 Dispose 自动移除。

### Scope / Effect / Disposer
- `Disposer func() error`；`Context.Effect(fn)` 立即执行 `fn`，把其返回的 disposer 入栈。
- `Context.Dispose` 以 **LIFO（逆序）** 回放，聚合错误，幂等。
- `Fork` 派生子作用域；`Isolate` 额外形成服务解析屏障。

### Loader / Plugin / Config
- `Plugin{Name, Inject, Provides, Config, Apply}`；`Load` 在父作用域的 Fork 上 Apply，先校验 `Inject`，缺依赖给出明确错误；Apply 失败则回滚作用域。
- `LoadPlugins` 依据 `Provides→key` 映射与直接插件名依赖做**拓扑加载**，检测环与重复。
- 配置树：`Plugin.Config()` 默认值在加载时写入 `Kernel.configs`，绑定到插件作用域（`Context.Config()`），可被 `SetConfig` 覆盖。
- `Unload` 逆序回放该插件作用域副作用；`Reload` = Unload + Load；`Shutdown` 逆序卸载全部插件并释放根作用域。

### 并发模型（与 Cordis/TS 的显式差异）
- 事件监听器默认在调用方 goroutine **同步**执行；仅 `Parallel` 显式 fan-out。
- 所有内核数据结构（`Kernel.mu`、`Context.mu`）以读写锁保护；不使用单线程事件循环。
- 这些差异记录于此，语义等价但不追求二进制级兼容。

## 后果

- 正面：
  - `core` 零业务依赖，`check_arch.sh` 规则 7/8 保证 `core` 不 import `plugin/`、`internal/`；
  - 单测覆盖 `core` ≥ 90%，`-race` 通过；
  - 插件可通过 `Provide/inject/event/effect/scope` 组合，`plugin/demo` 已演示全部原语。
- 负面/代价：
  - Go 无 `async/await`，`waterfall` 用闭包链 `next` 表达，签名以 `any` 装箱 + 泛型入口转换，存在少量运行时断言成本；
  - 服务注册进内核级注册表意味着"同名服务覆盖"是显式语义（disposer 还原前值）。
- 对 dsh/Cordis 兼容性的影响：服务键与事件模式对齐，后续 Phase 6 经 SPP 桥接；不承诺二进制兼容。

## 后续

- 热重载的文件监听、插件清单加载（`sup.plugin.json`）在 Phase 2+ 接入，本阶段只提供 `Plugin`/`Manifest` 数据结构。
- WASM / `.so` 后端推迟至 2.1（B4/B5）。
