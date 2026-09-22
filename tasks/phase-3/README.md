# Phase 3 — 状态与交互 任务看板

- 执行 Skill：`harness-phase-3-state-interaction`
- 版本：v2.0.0-alpha.3
- 前置阶段：P2
- 集成分支：`phase/3-state`
- 验收报告：`tasks/acceptance/phase-3.md`
- 契约：`docs/adr/ADR-0003-state-interaction.md`

## 功能点
- [x] plugin-session（SQLite，兼容 v1 数据）
- [x] plugin-context（压缩/Token）
- [x] plugin-tui（interaction seam，TUI 实现）
- [x] plugin-cli + plugin-command（headless interaction + 斜杠命令）
- [x] plugin-config 旧配置兼容层
- [ ] 流式输出打通 TUI（推迟 2.1 / B2）
- [ ] TUI Ctrl+C → ctx 取消（推迟 2.1 / B3）

## 子 Agent 派发

| 子 Agent | 分支 | 状态 | 交付物 |
|---|---|---|---|
| leaf-config | `phase/3-state` | ✅ | `plugin/config/**` |
| state-session | `phase/3-state` | ✅ | `plugin/session/**` |
| state-context | `phase/3-state` | ✅ | `plugin/context/**` |
| interaction-tui | `phase/3-state` | ✅ | `plugin/tui/**` |
| interaction-cli | `phase/3-state` | ✅ | `plugin/cli/**`、`plugin/command/**` |

## 门禁
- [x] G0 编译 `go build ./... && go vet ./...` → exit 0
- [x] G1 静态检查 `golangci-lint run ./...` → 0 issues
- [x] G2 单元测试 `go test ./...` → 通过（宿主机 2 项已知环境失败除外）
- [x] G3 竞态 `go test -race ./plugin/... ./tests/integration/...` → PASS
- [x] G4 覆盖率：config 81.8%、session 90.0%、context 100%、tui 100%、cli 90.5%、command 97.9%
- [x] G5 架构 `bash scripts/check_arch.sh` → PASS
- [x] G6 Skill `bash scripts/check_skills.sh` → validated 11
- [x] G7 文档/任务 `bash scripts/check_task_checkboxes.sh` → OK

## 文档维护
- [x] `docs/adr/ADR-0003-state-interaction.md`（含补充）
- [x] 验收报告 `tasks/acceptance/phase-3.md`
- [x] `docs/STATUS.md`
