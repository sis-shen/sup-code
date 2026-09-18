# Phase 2 — 叶子插件化 任务看板

- 执行 Skill：`harness-phase-2-leaf-plugins`
- 版本：v2.0.0-alpha.2
- 前置阶段：P1
- 集成分支：`phase/2-plugins`
- 验收报告：`tasks/acceptance/phase-2.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
- [ ] plugin-llm + 4 适配器
- [ ] plugin-tools + 6 内置工具
- [ ] plugin-mcp
- [ ] plugin-skill + tool-skill
- [ ] plugin-permission
- [ ] plugin-memory
- [ ] plugin-worktree
- [ ] plugin-hooks-audit/git
- [ ] 新 LLM 适配器（推迟 2.1）

## 子 Agent 派发

| 子 Agent | 分支 | 状态 | 交付物 |
|---|---|---|---|
| （见阶段 Skill） | | ⬜ | |

## 门禁
- [ ] G0 编译 `go build ./... && go vet ./...`
- [ ] G1 静态检查 `bash scripts/lint.sh`
- [ ] G2 单元测试 `go test ./...`
- [ ] G3 竞态 `bash scripts/test.sh`
- [ ] G4 覆盖率（新增代码 ≥80%）
- [ ] G5 架构 `bash scripts/check_arch.sh`
- [ ] G6 Skill `bash scripts/check_skills.sh`
- [ ] G7 文档/任务 `bash scripts/check_task_checkboxes.sh`

## 文档维护
- [ ] 相关 `docs/*` 已更新
- [ ] 关键决策写入 `docs/adr/`
- [ ] 验收报告 `tasks/acceptance/phase-2.md` 已填写并勾选 DoD
