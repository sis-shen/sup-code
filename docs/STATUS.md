# Sup Harness 2.0 开发状态看板

> 由编排 Skill `harness-lifecycle-orchestrator` 维护；每次阶段推进后更新。
> 生成/更新时间：2026-09-18

## 当前状态

| 字段 | 值 |
|---|---|
| 状态 | **PASSED（Phase 2 完成；可启动 Phase 3）** |
| 当前阶段 | Phase 3 — 状态与交互（NOT STARTED） |
| 已通过阶段 | Phase 0、Phase 1、Phase 2 |
| 当前版本 | v2.0.0-alpha.2 |
| 最近 tag | `v2.0.0-alpha.2` |
| 阻塞项 | 无 |

## 阶段进度

| 阶段 | 状态 | 门禁 | 版本 | 任务看板 | 验收报告 |
|---|---|---|---|---|---|
| P0 工程地基/基线冻结 | **PASSED** | 本地 + CI 全绿 + 已发布 | v2.0.0-alpha.0 | `tasks/phase-0/README.md` | `tasks/acceptance/phase-0.md` |
| P1 Cordis 内核 core | **PASSED** | 本地 + CI 全绿 + 已发布（core 覆盖率 94.0%） | v2.0.0-alpha.1 | `tasks/phase-1/README.md` | `tasks/acceptance/phase-1.md` |
| P2 叶子插件化 | **PASSED** | 本地 + CI 全绿 + 已发布（8 插件覆盖 ≥82.6%） | v2.0.0-alpha.2 | `tasks/phase-2/README.md` | `tasks/acceptance/phase-2.md` |
| P3 状态与交互 | NOT STARTED | — | v2.0.0-alpha.3 | `tasks/phase-3/README.md` | `tasks/acceptance/phase-3.md` |
| P4 心脏迁移（Agent） | NOT STARTED | — | v2.0.0-alpha.4 | `tasks/phase-4/README.md` | `tasks/acceptance/phase-4.md` |
| P5 双子进程与远程 | NOT STARTED | — | v2.0.0-beta.1 | `tasks/phase-5/README.md` | `tasks/acceptance/phase-5.md` |
| P6 dsh 生态兼容 | NOT STARTED | — | v2.0.0-beta.2 | `tasks/phase-6/README.md` | `tasks/acceptance/phase-6.md` |
| P7 收尾与发布 | NOT STARTED | — | v2.0.0 | `tasks/phase-7/README.md` | `tasks/acceptance/phase-7.md` |

状态取值：`NOT STARTED` / `IN PROGRESS` / `GATE` / `PASSED` / `BLOCKED`。

## 已知基线失败（回归判据，见 `docs/baseline-v1.md`）

| 用例 | 根因 | 判据 |
|---|---|---|
| TestDefaultValues / TestConfigGetExistingKey | 宿主机 `~/.supcode/config.yaml` provider=deepseek | 仅宿主机复现；干净 CI 通过；原因不得变化 |

> 原列为失败的 `TestMissingAPIKey` / `TestNewSupCode_MissingAPIKey` 经复核为**陈旧测试**（API key 校验已按设计推迟到 Agent 层），已更正，见 `docs/baseline-v1.md` §3。

## 阻塞与风险

| 项 | 影响阶段 | 处理 | 状态 |
|---|---|---|---|
| CI 首次全绿 | P0 收尾 | PR #1 全部 required checks 通过（lint/build/test/arch/skills/diff-coverage） | RESOLVED |
| 本机 golangci-lint 与 CI 配置版本 | P0 | 已迁移 `.golangci.yml` 至 v2；CI 用 action v9 + v2.13.2 | RESOLVED |
| G3 竞态从未实测 | P0 | 本地与 CI 全量 `-race` 执行并修复 4 处 data race | RESOLVED |
| `depslock` cgo sqlite 空白导入 | P2/P7 | Phase 7 清理或改纯 Go sqlite | DEFERRED |

## Backlog（2.1+）

| # | 项 | 目标版本 |
|---|---|---|
| B1 | Agent 并行工具调用 | 2.1 |
| B2 | 流式输出打通到 TUI | 2.1 |
| B3 | TUI Ctrl+C → ctx 取消 | 2.1 |
| B4 | WASM 插件运行时（wazero） | 2.1 |
| B5 | 动态 `.so` 插件后端 | 2.1 |
| B6 | 新 LLM/Embedding 适配器 | 2.1 |
| B7 | Agent invariants 运行时校验 | 2.1 |
| B8 | dsh 事件面全量覆盖 | 2.x |
| B9 | Web UI / HTTP server 插件 | 2.x |
| B10 | 会话模型向"仅追加事件流"演进 | 2.x |

## 最近更新

| 日期 | 事件 | 备注 |
|---|---|---|
| 2026-09-13 | Phase 0 执行 | 基线冻结、目录骨架、CI/门禁脚本、模板、任务看板；本地 G0/G5/G6/G7 通过 |
| 2026-09-18 | Phase 0 门禁补救（FIX_LOOP） | G1 464→0（`.golangci.yml` v2）；G3 修复 3 处 data race；G6/G5 脚本修复；ADR-0000 |
| 2026-09-18 | Phase 0 CI 首轮修复 + 全绿 | PR #1：修复 lint action、mcp 竞态、Windows mock、未提交 fixture、陈旧测试、diff-coverage 工具；所有 required checks 通过 |
| 2026-09-18 | Phase 0 合并与发布 | PR #1 squash 合入 main（`804f8f6`）；tag `v2.0.0-alpha.0`；goreleaser 发布多平台产物（预发布） |
| 2026-09-18 | Phase 1 执行 | Cordis 内核 `core` 落地：Context/Service、事件五模式、Scope/Effect、Loader/拓扑/配置树；`plugin/demo`；core 覆盖 94.0%；ADR-0001 |
| 2026-09-18 | Phase 1 CI 全绿 | PR #2：lint/build×3/test/arch/skills/diff-coverage/phase-gate 全部通过 |
| 2026-09-18 | Phase 1 合并与发布 | PR #2 squash 合入 main（`d3a145f`）；tag `v2.0.0-alpha.1`；goreleaser 预发布多平台产物 |
| 2026-09-18 | Phase 2 执行 | 8 叶子插件迁移（llm/tools/permission/hooks/mcp/skill/memory/worktree）+ 工具流水线事件桥；`pkg` 服务键/事件契约；ADR-0002 |
| 2026-09-18 | Phase 2 CI 全绿 + 发布 | PR #3 squash 合入 main（`9234ce3`）；tag `v2.0.0-alpha.2`；goreleaser 预发布多平台产物 |
