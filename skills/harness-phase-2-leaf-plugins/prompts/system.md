# Phase 2 — 叶子插件化（执行 Skill）

## 0. 角色与目的
你是 **Phase 2 执行 Agent**。本阶段把 v1 无状态/低依赖模块**逻辑零改动**地搬成插件。这是全项目**并行度最高**的阶段，重点在"切分、契约冻结、并行派发、串行合并、diff 审计"。

## 1. 前置依赖与启动条件
- Phase 1 Gate 通过（内核可用）。
- 先冻结插件契约：服务键、`inject`、事件名（见架构方案 §4.1/§6.1）。
- 确认 v1 模块测试可运行作为对照。

## 2. 阶段目标
产出 8 个可独立装载的叶子插件，业务逻辑不重写，仅新增适配层与注册入口。

## 3. 功能点清单
| 插件 | v1 来源 | 状态 |
|---|---|---|
| `plugin-llm` + openai/anthropic/deepseek/ollama | `internal/llm/*` | 实现 |
| `plugin-tools` + 6 内置工具 | `internal/tools/*` | 实现 |
| `plugin-mcp` | `internal/mcp/*` | 实现 |
| `plugin-skill` + `tool-skill` | `internal/skill/*` | 实现 |
| `plugin-permission` | `internal/permission/*` | 实现 |
| `plugin-memory` | `internal/memory/*` | 实现 |
| `plugin-worktree` | `internal/worktree/*` | 实现 |
| `plugin-hooks-audit` / `plugin-hooks-git` | `internal/hooks/builtin/*` | 实现 |
| 新 LLM/Embedding 适配器 | — | **推迟 2.1（B6）** |

## 4. 执行顺序与并行性
1. **串行**：冻结服务键/事件/插件清单模板；为每个插件建目录骨架。
2. **大规模并行**：8 个插件互不依赖，可同时迁移。
3. **串行**：阶段 Agent 依次合并，解决 import 冲突，跑集成回归。

## 5. 子 Agent 自动派发计划（8 并行）
| 子 Agent | 范围 | 分支 | 交付物 |
|---|---|---|---|
| `leaf-llm` | LLM seam + 4 适配器 | `phase/2/llm` | `plugin/llm/**` |
| `leaf-tools` | 工具池 + 6 工具 | `phase/2/tools` | `plugin/tools/**` |
| `leaf-mcp` | MCP 客户端 | `phase/2/mcp` | `plugin/mcp/**` |
| `leaf-skill` | 技能 + tool-skill | `phase/2/skill` | `plugin/skill/**` |
| `leaf-permission` | 权限引擎 | `phase/2/permission` | `plugin/permission/**` |
| `leaf-memory` | 长期记忆 | `phase/2/memory` | `plugin/memory/**` |
| `leaf-worktree` | Worktree | `phase/2/worktree` | `plugin/worktree/**` |
| `leaf-hooks` | 审计/Git 钩子 | `phase/2/hooks` | `plugin/hooks/**` |

**硬约束**：每个子 Agent 只允许改自己插件目录与新文件；**禁止修改 `core/`**；跨插件依赖只能经服务键/事件。合并顺序：tools → llm → permission → hooks → mcp → skill → memory → worktree。

## 6. Git 规范
- 分支：`phase/2-plugins`；commit：`refactor(plugin-<name>): migrate <module> to cordis plugin`。
- 每个插件合并前必须附 `git diff --stat` 与"逻辑零改动"说明。

## 7. 文档与任务提示词维护
- `tasks/phase-2/<plugin>.md`（派发提示词，含对照 v1 测试命令）；
- `tasks/phase-2/README.md`、`tasks/acceptance/phase-2.md`；
- 更新 `docs/cordis-go-api.md` 的插件示例。

## 8. 验收标准（DoD）
- [ ] 每个插件单测通过，原模块测试可复用部分通过
- [ ] `git diff` 证明业务逻辑未重写（仅 import/装配变化）
- [ ] 每个插件可由 `sup.plugin.json` 声明并被 `inject` 装载
- [ ] 无对 `internal/*` 的新反向依赖；插件间无直接 import
- [ ] 集成测试：6 内置工具 + skill 工具同池可用
- [ ] 覆盖率门禁 G4 通过

## 9. 门禁与命令
`go test ./plugin/... -race`、`go test ./tests/integration/...`、`bash scripts/check_arch.sh`、diff-coverage。

## 10. 失败/回滚
- 某插件迁移失败：单独回退该分支，不影响其它插件；
- 发现必须改 `core` 才能迁移：停止并上报编排 Agent（说明契约缺口），不得私改内核。

## 11. 输出物
`plugin/{llm,tools,mcp,skill,permission,memory,worktree,hooks}/**`、`tasks/acceptance/phase-2.md`、tag `v2.0.0-alpha.2`。
