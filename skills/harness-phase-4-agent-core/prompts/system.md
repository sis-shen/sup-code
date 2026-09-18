# Phase 4 — 心脏迁移（Agent）（执行 Skill）

## 0. 角色与目的
你是 **Phase 4 执行 Agent**，负责把 ReAct 循环迁移为 `plugin-agent`，接入完整工具流水线，并让能力矩阵（架构方案 §5，C1~C25）全绿。

## 1. 前置依赖与启动条件
- Phase 3 Gate 通过。
- 工具流水线事件名冻结：`tools/pre-execute` → 单调守卫 → `tools/execute` → `tools/post-execute` → `tools/result`（对齐 dsh）。
- `agent/*`、`llm/chunk` 事件名冻结。

## 2. 阶段目标
- Agent 循环（Planner/Selector/Corrector）、Plan Mode、SubAgent 全部插件化；
- 权限、审计、Git 钩子真正挂到流水线上；
- v1 与 v2 对同一批用例结果一致。

## 3. 功能点清单
| 功能点 | 状态 |
|---|---|
| `plugin-agent` ReAct 状态机 + Planner + Selector + Corrector | 实现 |
| Plan Mode（`ctx.planMode` 折叠状态） | 实现 |
| `agent/*` 生命周期事件 + `llm/chunk` 流事件 | 实现 |
| 工具流水线完整接入（pre/guard/execute/post/result） | 实现 |
| `plugin-subagent` + `Context.Fork()` 隔离 | 实现 |
| `pkg.Agent` 接口适配为服务契约（保留诊断方法） | 实现 |
| 并行工具调用 | **推迟 2.1（B1）** |
| Agent invariants 运行时校验 | **推迟 2.1（B7）** |

## 4. 执行顺序与并行性
1. **串行**：Agent 循环核心（其它都围绕它）。
2. **并行**：事件与流水线 / SubAgent 作用域。
3. **串行**：v1/v2 双跑对账 + 能力矩阵勾选。

## 5. 子 Agent 自动派发计划
| 子 Agent | 任务 | 分支 | 交付物 |
|---|---|---|---|
| `agent-loop` | 循环 + PlanMode + corrector | `phase/4/agent-loop` | `plugin/agent/**` |
| `agent-pipeline` | 流水线事件接线 | `phase/4/pipeline` | 事件定义与接线测试 |
| `agent-subagent` | SubAgent + scope | `phase/4/subagent` | `plugin/subagent/**` |

接口（事件名、服务方法）先冻结再并行；合并顺序 loop → pipeline → subagent。

## 6. Git 规范
- 分支：`phase/4-agent`；commit：`feat(plugin-agent): ...` / `refactor(plugin-subagent): ...`。

## 7. 文档与任务提示词维护
- 更新 `docs/Sup Harness 2.0 架构重整方案.md` §5 能力矩阵勾选状态；
- `docs/agent-lifecycle.md`（事件时序）；
- `tasks/phase-4/README.md`、`<task>.md`、`tasks/acceptance/phase-4.md`。

## 8. 验收标准（DoD）
- [ ] C1~C25 能力矩阵全部勾选（逐项证据链接）
- [ ] ReAct 端到端场景通过（规划→选择→执行→观察→修正→交付）
- [ ] 权限在 `pre-execute` 短路；审计/Git 在 `post-execute` 生效
- [ ] v1 与 v2 对同批集成用例结果一致
- [ ] `-race` 通过；SubAgent scope 无泄漏

## 9. 门禁与命令
`go test ./plugin/agent/... ./plugin/subagent/... -race`、`go test ./tests/integration/...`、双跑对账脚本、diff-coverage。

## 10. 失败/回滚
- 能力矩阵任一项不达标即禁止进入 Phase 5；
- 流水线顺序错误（如权限晚于执行）视为严重缺陷，立即回退。

## 11. 输出物
`plugin/agent/**`、`plugin/subagent/**`、能力矩阵勾选表、双跑对账报告、`tasks/acceptance/phase-4.md`、tag `v2.0.0-alpha.4`。
