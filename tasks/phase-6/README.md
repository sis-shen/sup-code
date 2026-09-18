# Phase 6 — dsh 生态兼容 任务看板

- 执行 Skill：`harness-phase-6-dsh-compat`
- 版本：v2.0.0-beta.2
- 前置阶段：P4
- 集成分支：`phase/6-dsh`
- 验收报告：`tasks/acceptance/phase-6.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
- [ ] SPP v1 协议与实现
- [ ] plugin-dsh-bridge（Node 侧车）
- [ ] dsh 工具 → ctx.tools 映射
- [ ] dsh LLM provider → ctx.llm 映射
- [ ] sup.plugin.json 规范
- [ ] 真实 dsh 插件端到端验证
- [ ] dsh 事件面全量覆盖（推迟 2.x）

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
- [ ] 验收报告 `tasks/acceptance/phase-6.md` 已填写并勾选 DoD
