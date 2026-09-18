# Phase 3 — 状态与交互 任务看板

- 执行 Skill：`harness-phase-3-state-interaction`
- 版本：v2.0.0-alpha.3
- 前置阶段：P2
- 集成分支：`phase/3-state`
- 验收报告：`tasks/acceptance/phase-3.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
- [ ] plugin-session（SQLite 兼容）
- [ ] plugin-context（压缩/Token）
- [ ] plugin-tui（interaction seam）
- [ ] plugin-cli + plugin-command
- [ ] plugin-config 旧配置兼容
- [ ] 流式输出打通 TUI（推迟 2.1）
- [ ] TUI Ctrl+C → ctx 取消（推迟 2.1）

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
- [ ] 验收报告 `tasks/acceptance/phase-3.md` 已填写并勾选 DoD
