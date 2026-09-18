# Sup Harness 2.0 开发状态看板

> 由编排 Skill `harness-lifecycle-orchestrator` 维护；每次阶段推进后更新。
> 生成/更新时间：2026-09-13

## 当前状态

| 字段 | 值 |
|---|---|
| 状态 | **PHASE_RUNNING（P0 本地门禁全绿，CI 待验证）** |
| 当前阶段 | Phase 0 — 工程地基与基线冻结 |
| 下一阶段 | Phase 1 — Cordis 内核 core |
| 当前版本 | v2.0.0-alpha.0（待 tag） |
| 最近 tag | — |
| 阻塞项 | GitHub CI 首次全绿需推送 `phase/0-engineering` 分支/开 PR |

## 阶段进度

| 阶段 | 状态 | 门禁 | 版本 | 任务看板 | 验收报告 |
|---|---|---|---|---|---|
| P0 工程地基/基线冻结 | GATE（本地 PASS，CI 待验证） | G0/G1/G3/G5/G6/G7 通过；G2 4 已知环境失败 | v2.0.0-alpha.0 | `tasks/phase-0/README.md` | `tasks/acceptance/phase-0.md` |
| P1 Cordis 内核 core | NOT STARTED | — | v2.0.0-alpha.1 | `tasks/phase-1/README.md` | `tasks/acceptance/phase-1.md` |
| P2 叶子插件化 | NOT STARTED | — | v2.0.0-alpha.2 | `tasks/phase-2/README.md` | `tasks/acceptance/phase-2.md` |
| P3 状态与交互 | NOT STARTED | — | v2.0.0-alpha.3 | `tasks/phase-3/README.md` | `tasks/acceptance/phase-3.md` |
| P4 心脏迁移（Agent） | NOT STARTED | — | v2.0.0-alpha.4 | `tasks/phase-4/README.md` | `tasks/acceptance/phase-4.md` |
| P5 双子进程与远程 | NOT STARTED | — | v2.0.0-beta.1 | `tasks/phase-5/README.md` | `tasks/acceptance/phase-5.md` |
| P6 dsh 生态兼容 | NOT STARTED | — | v2.0.0-beta.2 | `tasks/phase-6/README.md` | `tasks/acceptance/phase-6.md` |
| P7 收尾与发布 | NOT STARTED | — | v2.0.0 | `tasks/phase-7/README.md` | `tasks/acceptance/phase-7.md` |

状态取值：`NOT STARTED` / `IN PROGRESS` / `GATE` / `PASSED` / `BLOCKED`。

## 已知基线失败（回归判据，见 `docs/baseline-v1.md`）

| 用例 | 根因 | 判据 |
|---|---|---|
| TestDefaultValues / TestConfigGetExistingKey | 宿主机 `~/.supcode/config.yaml` provider=deepseek | 原因不得变化 |
| TestMissingAPIKey / TestNewSupCode_MissingAPIKey | 宿主机 `SUPCODE_LLM_API_KEY` 已设置 | 原因不得变化、不得新增 |

## 阻塞与风险

| 项 | 影响阶段 | 处理 | 状态 |
|---|---|---|---|
| CI 未在 GitHub 上首次运行 | P0 收尾 | 推送 `phase/0-engineering` 并开 PR，确认 required checks 全绿 | OPEN |
| 本机 golangci-lint 与 CI 配置版本 | P0 | 已迁移 `.golangci.yml` 至 v2 并对齐 `version: latest`；本地 0 issues | RESOLVED |
| G3 竞态从未实测 | P0 | 已本地 `-race` 全量执行并修复 3 处 data race | RESOLVED |
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
