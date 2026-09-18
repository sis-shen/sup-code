# Phase 5 — 双子进程与远程 任务看板

- 执行 Skill：`harness-phase-5-daemon-remote`
- 版本：v2.0.0-beta.1
- 前置阶段：P4
- 集成分支：`phase/5-daemon`
- 验收报告：`tasks/acceptance/phase-5.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
- [ ] cmd/sup（单发/TUI/子命令）
- [ ] cmd/supd（常驻守护）
- [ ] plugin-mcp-server
- [ ] 信号处理/优雅关闭/配置优先级
- [ ] goreleaser 切换到 cmd/sup
- [ ] HTTP/Web API（推迟 2.x）

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
- [ ] 验收报告 `tasks/acceptance/phase-5.md` 已填写并勾选 DoD
