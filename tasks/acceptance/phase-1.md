# Phase 1 — Cordis 内核 core 验收报告（v2.0.0-alpha.1）

> 状态：**PASSED（本地门禁 + CI 全绿）**
> 执行 Skill：`harness-phase-1-kernel`
> 分支：`phase/1-kernel`（PR #2）
> 生成时间：2026-09-18
> 相关决策：`docs/adr/ADR-0001-cordis-kernel.md`

## 1. 交付物清单

| 交付物 | 路径 | 是否存在 |
|---|---|---|
| 内核：Context/Service | `core/context.go`、`core/service.go` | ✅ |
| 内核：事件五模式 | `core/event.go` | ✅ |
| 内核：Scope/Effect | `core/context.go`（Effect/Dispose）、`core/effect.go` | ✅ |
| 内核：Loader/Plugin/Config | `core/loader.go`、`core/plugin.go` | ✅ |
| 内核测试 | `core/*_test.go` | ✅ |
| Demo 插件 | `plugin/demo/demo.go`、`plugin/demo/demo_test.go` | ✅ |
| ADR | `docs/adr/ADR-0001-cordis-kernel.md` | ✅ |
| API 文档 | `docs/cordis-go-api.md` | ✅ |

## 2. 门禁结果（本地实测）

| 门禁 | 命令 | 结果 |
|---|---|---|
| G0 编译 | `go build ./... && go vet ./...` | ✅ exit 0 |
| G1 静态检查 | `golangci-lint run ./...` | ✅ 0 issues |
| G2 单元测试 | `go test ./...` | ✅ 通过 |
| G3 竞态 | `go test -race ./core/... ./plugin/demo/...` | ✅ PASS |
| G4 覆盖率 | `core` 94.0% / `plugin/demo` 94.7% | ✅ ≥85% |
| G5 架构 | `bash scripts/check_arch.sh` | ✅ PASS（规则 7/8 保证 core 零业务依赖） |
| G6 Skill | `bash scripts/check_skills.sh` | ✅ validated 11 |
| G7 文档/任务 | `bash scripts/check_task_checkboxes.sh` | ✅ OK |
| G8/CI | PR #2 required checks | ✅ 全绿（lint/build×3/test/arch/skills/diff-coverage/phase-gate） |

## 3. 验收标准（DoD）

- [x] 五模式事件均有测试；waterfall 可短路与包装
- [x] `inject` 缺失依赖时报错明确；拓扑加载顺序正确（`Provides`→key + 直接插件名）
- [x] `Effect` 卸载逆序回放；`Dispose` 幂等并聚合错误
- [x] `Scope` 隔离测试 `-race` 通过（`Fork`/`Isolate`、隔离屏障）
- [x] `core` 不 import `plugin/*` 或 `internal/*`（G5）
- [x] 内核核心包覆盖率 ≥ 85%（`core` 94.0%）
- [x] Demo 插件完整演示 Provide/inject/event/effect/scope
- [x] CI 全绿（PR #2）

## 4. 证据

- 构建：`go build ./...` / `go vet ./...` → exit 0
- 测试：`go test -race -count=1 ./core/... ./plugin/demo/...`
  - `ok github.com/supcode/supcode/core  coverage: 94.0% of statements`
  - `ok github.com/supcode/supcode/plugin/demo  coverage: 94.7% of statements`
- 架构：`check_arch.sh` → `PASS: All architecture rules satisfied`（规则 1–11）
- 事件语义测试：`TestWaterfallWraps`（包装）、`TestWaterfallShortCircuit`（短路）、`TestParallelCollectsErrors`、`TestSerialFeedsPayloadAndReturnsLast`、`TestBailReturnsFirstDecision`
- 生命周期测试：`TestReloadReplaysEffects`、`TestDisposeAggregatesErrorsAndIsIdempotent`、`TestLoadApplyErrorDisposesScope`、`TestStartAndShutdown`
- 拓扑测试：`TestLoadPluginsTopology`、`TestLoadPluginsDirectNameDependency`、`TestLoadPluginsCycle`、`TestLoadPluginsDuplicate`

## 5. 推迟 / Backlog

| 项 | 原因 | 目标版本 |
|---|---|---|
| WASM 插件运行时（wazero） | 架构规划推迟（B4） | 2.1 |
| 动态 `.so` 插件后端 | 架构规划推迟（B5） | 2.1 |
| 插件清单文件加载（`sup.plugin.json`）与文件监听热重载 | 属 Phase 2+ 接入 | Phase 2/6 |

## 6. 结论

- 状态：**PASSED（本地 + CI）**
- 后续：合并 `phase/1-kernel` → `main`，打 tag `v2.0.0-alpha.1`。
- 签署：opencode Phase 1 执行 Agent（2026-09-18）
