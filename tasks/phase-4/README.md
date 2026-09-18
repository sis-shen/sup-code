# Phase 4 — 心脏迁移（Agent） 任务看板

- 执行 Skill：`harness-phase-4-agent-core`
- 版本：v2.0.0-alpha.4
- 前置阶段：P3
- 集成分支：`phase/4-agent`
- 验收报告：`tasks/acceptance/phase-4.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
- [ ] plugin-agent ReAct 循环 + PlanMode + Corrector
- [ ] agent/* 与 llm/chunk 事件
- [ ] 工具流水线 pre/guard/execute/post/result
- [ ] plugin-subagent + Context.Fork 隔离
- [ ] 能力矩阵 C1~C25 全绿
- [ ] 并行工具调用（推迟 2.1）

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
- [ ] 验收报告 `tasks/acceptance/phase-4.md` 已填写并勾选 DoD
