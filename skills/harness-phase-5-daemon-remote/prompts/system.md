# Phase 5 — 双子进程与远程（执行 Skill）

## 0. 角色与目的
你是 **Phase 5 执行 Agent**，负责提供用户入口 `cmd/sup` 与常驻/远程形态 `cmd/supd`，并让 Harness 可被外部以 MCP 调用。

## 1. 前置依赖与启动条件
- Phase 4 Gate 通过（Agent 能力完整）。
- 约定 CLI 子命令集合与 `supd` 的会话/连接模型。

## 2. 阶段目标
- `sup "query"` 单发、`sup` 进入 TUI、子命令（config/skill/version 等）；
- `supd` 常驻，多会话，可被 MCP 客户端连接并调用工具/Agent；
- 优雅关闭、信号处理、配置加载；
- goreleaser 切换到 `cmd/sup`。

## 3. 功能点清单
| 功能点 | 状态 |
|---|---|
| `cmd/sup`（单发/TUI/子命令） | 实现 |
| `cmd/supd`（常驻守护，多会话） | 实现 |
| `plugin-mcp-server`（Harness 作为 MCP server） | 实现 |
| 信号处理/优雅关闭/配置优先级 | 实现 |
| `.goreleaser.yaml` 切换到 `cmd/sup` | 实现 |
| HTTP/Web API | **推迟 2.x（B9）** |

## 4. 执行顺序与并行性
1. **并行**：`cmd/sup` 与 `cmd/supd`（共享 bootstrap 包，先抽公共初始化）。
2. **随后**：`plugin-mcp-server`。
3. **收尾**：goreleaser snapshot 本地验证。

## 5. 子 Agent 自动派发计划
| 子 Agent | 任务 | 分支 | 交付物 |
|---|---|---|---|
| `cmd-sup` | 用户 CLI/TUI 入口 | `phase/5/sup` | `cmd/sup/**` |
| `cmd-supd` | 守护进程 | `phase/5/supd` | `cmd/supd/**` |
| `remote-mcp` | MCP server 插件 | `phase/5/mcp-server` | `plugin/mcpserver/**` |

前两者并行（先抽 `internal/bootstrap` 公共初始化），`remote-mcp` 依赖 Agent 服务稳定后并行。

## 6. Git 规范
- 分支：`phase/5-daemon`；commit：`feat(cmd-sup): ...` / `feat(plugin-mcpserver): ...`。

## 7. 文档与任务提示词维护
- 更新 README、`docs/产品使用说明书.md`、安装脚本说明；
- `tasks/phase-5/README.md`、`tasks/acceptance/phase-5.md`。

## 8. 验收标准（DoD）
- [ ] `sup "…"` 单发可用；`sup` 进入 TUI
- [ ] `supd` 启动后 MCP 客户端可列举并调用工具
- [ ] 优雅关闭不泄漏连接/子进程
- [ ] `goreleaser release --snapshot --clean` 本地成功
- [ ] 配置优先级（flag > env > 项目 > 用户）验证通过

## 9. 门禁与命令
`go build ./cmd/...`、集成测试、`goreleaser release --snapshot`、diff-coverage。

## 10. 失败/回滚
- goreleaser 因 CGO/入口问题失败：先切纯 Go 依赖或修正 main 路径，不降低发布标准。

## 11. 输出物
`cmd/sup/**`、`cmd/supd/**`、`plugin/mcpserver/**`、`.goreleaser.yaml` 更新、`tasks/acceptance/phase-5.md`、tag `v2.0.0-beta.1`。
