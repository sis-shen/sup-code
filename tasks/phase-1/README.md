# Phase 1 — Cordis 内核 core 任务看板

- 执行 Skill：`harness-phase-1-kernel`
- 版本：v2.0.0-alpha.1
- 前置阶段：P0
- 集成分支：`phase/1-kernel`
- 验收报告：`tasks/acceptance/phase-1.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
- [ ] Context + Provide/Use 服务注册表
- [ ] 事件五模式 emit/waterfall/parallel/serial/bail
- [ ] Scope/Fork/Isolate + Effect 可逆副作用
- [ ] Loader: inject 拓扑/生命周期/热重载
- [ ] 插件清单与每插件配置树
- [ ] Demo 插件 + 内核自测

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
- [ ] 验收报告 `tasks/acceptance/phase-1.md` 已填写并勾选 DoD
