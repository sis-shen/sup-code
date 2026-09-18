# Phase 1 — Cordis 内核 core（执行 Skill）

## 0. 角色与目的
你是 **Phase 1 执行 Agent**，负责实现与 Cordis 语义等价的 Go 微内核。内核是唯一非插件的部分，必须**零业务依赖**。

## 1. 前置依赖与启动条件
- Phase 0 Gate 通过（CI 全绿、骨架就绪）。
- 接口契约（`core` 的导出符号）在动手前**先冻结**并写入 `docs/adr/ADR-0001-cordis-kernel.md`。

## 2. 阶段目标
实现：Context、服务注册表、事件总线五模式、Scope/Effect、Loader（inject 拓扑/生命周期/热重载）、插件清单与配置树；用 Demo 插件证明全部原语可用。

## 3. 功能点清单
| 功能点 | 状态 |
|---|---|
| `Context` + `Provide/Use` 服务注册表 | 实现 |
| 事件五模式 emit/waterfall/parallel/serial/bail | 实现 |
| `Scope`/`Fork`/`Isolate` + 引用计数 | 实现 |
| `Effect` 可逆副作用 + 逆序 Disposer | 实现 |
| `Loader`：inject 拓扑、Load/Unload/Reload | 实现 |
| 插件清单（manifest）与每插件配置树 | 实现 |
| 内核自测 + `plugin/demo` | 实现 |
| WASM 插件运行时 | **推迟 2.1（B4）** |
| 动态 `.so` 插件后端 | **推迟 2.1（B5）** |

## 4. 执行顺序与并行性
1. **串行**：`Context` + 服务注册表（其它模块都依赖它）。
2. **并行**：事件总线 / Scope+Effect / Loader 三组在接口冻结后并行。
3. **串行**：Demo 插件集成 + 全量内核测试。

## 5. 子 Agent 自动派发计划
| 子 Agent | 任务 | 分支 | 交付物 |
|---|---|---|---|
| `kernel-context` | Context + Service | `phase/1/context` | `core/context.go`、`core/service.go` |
| `kernel-event` | 五模式事件总线 | `phase/1/event` | `core/event.go` |
| `kernel-scope-effect` | Scope + Effect + Disposer | `phase/1/scope` | `core/scope.go`、`core/effect.go` |
| `kernel-loader` | Loader + manifest + config | `phase/1/loader` | `core/loader.go`、`core/plugin.go` |

**并行前提**：先由阶段 Agent 提交接口骨架（`core/api.go`）并冻结，四个子 Agent 方可并行。合并由阶段 Agent 串行完成。

## 6. Git 规范
- 分支：`phase/1-kernel`；commit：`feat(kernel): ...`。
- 接口变更必须单独 commit 并在 ADR 记录。

## 7. 文档与任务提示词维护
- `docs/adr/ADR-0001-cordis-kernel.md`（决策与语义映射表）；
- `docs/cordis-go-api.md`（对外 API 使用说明）；
- `tasks/phase-1/README.md`、`tasks/phase-1/<task>.md`、`tasks/acceptance/phase-1.md`。

## 8. 验收标准（DoD）
- [ ] 五模式事件均有测试；waterfall 可短路与包装
- [ ] `inject` 缺失依赖时报错明确；拓扑加载顺序正确
- [ ] `Effect` 卸载逆序回放；循环 reload 压测无泄漏
- [ ] `Scope` 隔离测试 `-race` 通过
- [ ] `core` 不 import `plugin/*` 或 `internal/*`（G5）
- [ ] 内核核心包覆盖率 ≥ 85%
- [ ] Demo 插件完整演示 Provide/inject/event/effect/scope

## 9. 门禁与命令
`go test ./core/... -race`、`go build ./...`、`bash scripts/check_arch.sh`、`bash scripts/lint.sh`。

## 10. 失败/回滚
- 语义正确性存疑时，以 Cordis/dsh 官方文档与 `cordis-rs` 为准并在 ADR 记录偏差；
- 并发问题（死锁/泄漏）阻塞：回退到 `v2.0.0-alpha.0` tag 重做。

## 11. 输出物
`core/*`、`plugin/demo/*`、`docs/adr/ADR-0001-cordis-kernel.md`、`docs/cordis-go-api.md`、`tasks/acceptance/phase-1.md`、tag `v2.0.0-alpha.1`。
