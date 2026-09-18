# Phase 2 — 叶子插件化 任务看板

- 执行 Skill：`harness-phase-2-leaf-plugins`
- 版本：v2.0.0-alpha.2
- 前置阶段：P1
- 集成分支：`phase/2-plugins`
- 验收报告：`tasks/acceptance/phase-2.md`
- 契约：`docs/adr/ADR-0002-plugin-contract.md`

## 功能点
- [x] plugin-llm + 4 适配器（openai/anthropic/deepseek/ollama）
- [x] plugin-tools + 6 内置工具（含工具流水线事件桥接）
- [x] plugin-mcp
- [x] plugin-skill + tool-skill
- [x] plugin-permission
- [x] plugin-memory
- [x] plugin-worktree
- [x] plugin-hooks-audit / plugin-hooks-git
- [ ] 新 LLM/Embedding 适配器（推迟 2.1 / B6）

## 子 Agent 派发

| 子 Agent | 分支 | 状态 | 交付物 |
|---|---|---|---|
| leaf-llm | `phase/2-plugins` | ✅ | `plugin/llm/**` |
| leaf-tools | `phase/2-plugins` | ✅ | `plugin/tools/**` |
| leaf-mcp | `phase/2-plugins` | ✅ | `plugin/mcp/**` |
| leaf-skill | `phase/2-plugins` | ✅ | `plugin/skill/**` |
| leaf-permission | `phase/2-plugins` | ✅ | `plugin/permission/**` |
| leaf-memory | `phase/2-plugins` | ✅ | `plugin/memory/**` |
| leaf-worktree | `phase/2-plugins` | ✅ | `plugin/worktree/**` |
| leaf-hooks | `phase/2-plugins` | ✅ | `plugin/hooks/**` |

> 硬约束：各子 Agent 仅改自身 `plugin/<name>/`；未修改 `core/`；无跨插件 import。编排 Agent 授权两处跨阶段契约补充：`core.Manifest.Events`、`internal/memory.Store.UpdateEmbedding`（见 ADR-0002 补充）。

## 门禁
- [x] G0 编译 `go build ./... && go vet ./...` → exit 0
- [x] G1 静态检查 `golangci-lint run ./...` → 0 issues
- [x] G2 单元测试 `go test ./...` → 通过（宿主机 2 项已知环境失败除外）
- [x] G3 竞态 `go test -race ./plugin/...` → PASS
- [x] G4 覆盖率：全部插件 ≥ 82.6%（llm/mcp/worktree 100%）
- [x] G5 架构 `bash scripts/check_arch.sh` → PASS（规则 11：插件间无直接 import）
- [x] G6 Skill `bash scripts/check_skills.sh` → validated 11
- [x] G7 文档/任务 `bash scripts/check_task_checkboxes.sh` → OK

## 文档维护
- [x] `docs/adr/ADR-0002-plugin-contract.md`（含补充）
- [x] 验收报告 `tasks/acceptance/phase-2.md` 已填写
- [x] `docs/STATUS.md` 已更新
