# Phase 1 — Cordis 内核 core 任务看板

- 执行 Skill：`harness-phase-1-kernel`
- 版本：v2.0.0-alpha.1
- 前置阶段：P0
- 集成分支：`phase/1-kernel`
- 验收报告：`tasks/acceptance/phase-1.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
- [x] Context + Provide/Use 服务注册表（含 Fork/Isolate、内核级注册表）
- [x] 事件五模式 emit/waterfall/parallel/serial/bail（+ Once）
- [x] Scope/Fork/Isolate + Effect 可逆副作用（LIFO 回放）
- [x] Loader: inject 拓扑/生命周期/热重载（Load/Unload/Reload/Shutdown）
- [x] 插件清单（Plugin/Manifest）与每插件配置树
- [x] Demo 插件 + 内核自测

## 子 Agent 派发

| 子 Agent | 分支 | 状态 | 交付物 |
|---|---|---|---|
| kernel-context | `phase/1-kernel` | ✅ | `core/context.go`、`core/service.go` |
| kernel-event | `phase/1-kernel` | ✅ | `core/event.go` |
| kernel-scope-effect | `phase/1-kernel` | ✅ | `core/effect.go`（并入 context.go） |
| kernel-loader | `phase/1-kernel` | ✅ | `core/loader.go`、`core/plugin.go` |

> 说明：事件/Scope/Loader 共享 `Context` 内部结构，按 Orchestrator 裁决规则「同一核心文件禁止并行」，本阶段由阶段 Agent 串行实现以先冻结契约；测试与 Demo 独立成文件。

## 门禁
- [x] G0 编译 `go build ./... && go vet ./...` → exit 0
- [x] G1 静态检查 `golangci-lint run ./...` → 0 issues
- [x] G2 单元测试 `go test ./...` → 通过（宿主机 2 项已知环境失败除外）
- [x] G3 竞态 `go test -race ./core/... ./plugin/demo/...` → PASS
- [x] G4 覆盖率 `core` 94.0% / `plugin/demo` 94.7%（≥85%）
- [x] G5 架构 `bash scripts/check_arch.sh` → PASS（含 core 隔离规则）
- [x] G6 Skill `bash scripts/check_skills.sh` → validated 11
- [x] G7 文档/任务 `bash scripts/check_task_checkboxes.sh` → OK

## 文档维护
- [x] `docs/adr/ADR-0001-cordis-kernel.md`
- [x] `docs/cordis-go-api.md`
- [x] 验收报告 `tasks/acceptance/phase-1.md` 已填写并勾选 DoD
