# Phase 7 — 收尾与发布 任务看板

- 执行 Skill：`harness-phase-7-release`
- 版本：v2.0.0
- 前置阶段：P5+P6
- 集成分支：`phase/7-release`
- 验收报告：`tasks/acceptance/phase-7.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
- [ ] 删除 v1 装配根（build/wire/app）
- [ ] 清理未引用依赖（含 cgo sqlite）
- [ ] 全量文档更新 + 迁移指南
- [ ] CHANGELOG + tag + goreleaser 发布
- [ ] 安装脚本更新
- [ ] 性能基准报告

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
- [ ] 验收报告 `tasks/acceptance/phase-7.md` 已填写并勾选 DoD
