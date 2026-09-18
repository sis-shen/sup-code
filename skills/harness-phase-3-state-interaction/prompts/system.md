# Phase 3 — 状态与交互（执行 Skill）

## 0. 角色与目的
你是 **Phase 3 执行 Agent**。把会话、上下文、TUI、CLI/命令迁移为插件，使 2.0 恢复到"可用产品"形态（可用 mock agent 先跑通交互）。

## 1. 前置依赖与启动条件
- Phase 2 Gate 通过。
- `plugin-tui` 可以先用 mock agent/服务桩跑通，不阻塞后续 Agent 迁移。

## 2. 阶段目标
- `ctx.sessions` 提供会话创建/切换/持久化/恢复；
- `ctx.context` 提供上下文构建、压缩、Token 计量；
- `interaction` seam 提供 TUI 与 headless 两种实现；
- `ctx.commands` 提供 Slash 命令。

## 3. 功能点清单
| 功能点 | 状态 |
|---|---|
| `plugin-session`（SQLite，兼容 v1 数据） | 实现 |
| `plugin-context`（压缩/Token） | 实现 |
| `plugin-tui`（订阅 `llm/chunk`、`agent/state` 事件渲染） | 实现 |
| `plugin-cli` + `plugin-command` | 实现 |
| `plugin-config` 旧配置兼容层 | 实现 |
| 流式输出打通到 TUI | **推迟 2.1（B2）** |
| TUI Ctrl+C → ctx 取消 | **推迟 2.1（B3）** |
| Web UI / HTTP server | **推迟 2.x（B9）** |

## 4. 执行顺序与并行性
1. **并行**：session / context 互不依赖。
2. **随后**：tui 依赖 session；cli/command 依赖 command + agent 桩。
3. **收尾**：旧配置兼容验证 + 交互回归。

## 5. 子 Agent 自动派发计划
| 子 Agent | 任务 | 分支 | 交付物 |
|---|---|---|---|
| `state-session` | 会话服务 | `phase/3/session` | `plugin/session/**` |
| `state-context` | 上下文服务 | `phase/3/context` | `plugin/context/**` |
| `interaction-tui` | TUI 插件 | `phase/3/tui` | `plugin/tui/**` |
| `interaction-cli` | CLI + 命令 | `phase/3/cli` | `plugin/cli/**`、`plugin/command/**` |

前三者并行；`interaction-cli` 在 command 骨架冻结后并行启动。

## 6. Git 规范
- 分支：`phase/3-state`；commit：`refactor(plugin-<name>): ...`。

## 7. 文档与任务提示词维护
- `tasks/phase-3/README.md`、`tasks/phase-3/<task>.md`、`tasks/acceptance/phase-3.md`；
- 更新 `docs/产品使用说明书.md`（配置/命令变化）。

## 8. 验收标准（DoD）
- [ ] mock LLM 下单发查询可用、TUI 可交互
- [ ] 旧 `~/.supcode/config.yaml` 无缝加载
- [ ] 会话创建/切换/持久化/恢复闭环
- [ ] 超限触发压缩，Token 统计正确
- [ ] 命令注册表可扩展且不经模型

## 9. 门禁与命令
`go test ./plugin/session/... ./plugin/context/... ./plugin/tui/... -race`、集成测试、diff-coverage。

## 10. 失败/回滚
- TUI 依赖未就绪时用 headless 实现先行验收；交互项不过不影响 session/context 合并。

## 11. 输出物
`plugin/{session,context,tui,cli,command}/**`、`tasks/acceptance/phase-3.md`、tag `v2.0.0-alpha.3`。
