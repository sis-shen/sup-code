# Phase 0 — 工程地基与基线冻结 任务看板

- 执行 Skill：`harness-phase-0-baseline`
- 版本：v2.0.0-alpha.0
- 前置阶段：—
- 集成分支：`phase/0-engineering`
- 验收报告：`tasks/acceptance/phase-0.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
- [x] 跑 go build/vet/test 并记录 v1 基线 → `docs/baseline-v1.md`
- [x] 建立 2.0 目录骨架 core/ plugin/ cmd/sup cmd/supd
- [x] ci.yml 与门禁脚本
- [x] 提交/分支/文档规范模板（含 PR 模板、CODEOWNERS、.gitattributes）
- [x] 阶段报告与任务提示词模板
- [x] 门禁补救：G1 全量 lint 清零 + `.golangci.yml` 迁移 v2（ADR-0000）
- [x] 门禁补救：G3 修复 3 处 data race（skill loader/watcher + 测试 mock）
- [x] 门禁补救：G6 `check_skills.sh` Python 解释器探测（Windows 兼容）
- [x] 门禁补救：G5 `check_arch.sh` 增加 v2 规则 7–11

## 子 Agent 派发

| 子 Agent | 分支 | 状态 | 交付物 |
|---|---|---|---|
| baseline-runner | —（由阶段 Agent 直接执行） | ✅ | `docs/baseline-v1.md` |
| repo-scaffolder | — | ✅ | `core/`、`plugin/`、`cmd/sup`、`cmd/supd` |
| ci-builder | — | ✅ | `.github/workflows/*`、`scripts/check_*.sh` |
| docs-templater | — | ✅ | `docs/templates/*`、`.github/PULL_REQUEST_TEMPLATE.md` |
| lint-fix（8 组并行） | `phase/0/lint-*`（并入 phase/0-engineering） | ✅ | 464→0 lint |
| race-fix | — | ✅ | `internal/skill/{loader,watcher}.go`、测试 mock |

> 说明：本阶段为工程地基，编辑面互不冲突，由阶段 Agent 直接串行/并行执行；lint 补救按包边界拆分为 8 个互不重叠的子 Agent 并行完成。

## 门禁
- [x] G0 编译 `go build ./... && go vet ./...` → exit 0
- [x] G1 静态检查 `bash scripts/lint.sh` → `0 issues`（v2.13.2）
- [x] G2 单元测试 `go test ./...` → 宿主机 2 项已知环境失败（见 `docs/baseline-v1.md` §3），干净 CI 通过
- [x] G3 竞态 `bash scripts/test.sh` → 无 DATA RACE，仅 2 项已知环境失败
- [x] G4 覆盖率（Phase 0 无新增业务代码，N/A）
- [x] G5 架构 `bash scripts/check_arch.sh` → PASS（11 规则）
- [x] G6 Skill `bash scripts/check_skills.sh` → validated 11
- [x] G7 文档/任务 `bash scripts/check_task_checkboxes.sh` → OK
- [x] G8/CI：PR #1 全部 required checks 通过（lint/build/test/arch/skills/diff-coverage）

## 文档维护
- [x] 相关 `docs/*` 已更新（baseline/STATUS/skills-usage/规划）
- [x] 关键决策记录 `docs/adr/ADR-0000-phase-0-gate-remediation.md`
- [x] 验收报告 `tasks/acceptance/phase-0.md` 已填写
