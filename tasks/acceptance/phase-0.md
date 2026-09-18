# Phase 0 — 工程地基与基线冻结 验收报告（v2.0.0-alpha.0）

> 状态：**PASSED（本地门禁 + CI 全绿）**
> 执行 Skill：`harness-phase-0-baseline`
> 分支：`main`（本地执行）；集成/PR 分支：`phase/0-engineering`（PR #1）
> 生成时间：2026-09-13；补录/复核：2026-09-18（编排 PRECHECK + FIX_LOOP）
> 相关决策：`docs/adr/ADR-0000-phase-0-gate-remediation.md`

## 1. 交付物清单

| 交付物 | 路径 | 是否存在 |
|---|---|---|
| v1 基线报告 | `docs/baseline-v1.md`（§7 门禁补录） | ✅ |
| 目录骨架 | `core/doc.go`、`plugin/doc.go`、`cmd/sup/main.go`、`cmd/supd/main.go` | ✅ |
| CI 工作流 | `.github/workflows/{ci,gate,release,nightly}.yml` | ✅ |
| 门禁脚本 | `scripts/check_skills.sh`、`scripts/check_task_checkboxes.sh`、`scripts/check_arch.sh`（v2 规则） | ✅ |
| 任务生成器 | `scripts/gen_phase_tasks.{py,ps1,sh}`、`scripts/phase_manifest.json` | ✅ |
| 规范模板 | `docs/templates/{phase-report,task-prompt,adr}.md`、`.github/PULL_REQUEST_TEMPLATE.md`、`.github/CODEOWNERS`、`.gitattributes` | ✅ |
| 状态看板 | `docs/STATUS.md` | ✅ |
| 任务看板 | `tasks/phase-0..7/README.md`、`tasks/acceptance/phase-0..7.md` | ✅ |
| Lint 配置 | `.golangci.yml`（v2 格式） | ✅ |
| ADR | `docs/adr/ADR-0000-phase-0-gate-remediation.md` | ✅ |

## 2. 门禁结果（2026-09-18 本地实测）

| 门禁 | 命令 | 结果 |
|---|---|---|
| G0 编译 | `go build ./... && go vet ./...` | ✅ exit 0 |
| G1 静态检查 | `golangci-lint run ./...`（v2.13.2） | ✅ **0 issues** |
| G2 单元测试 | `go test ./...` | ⚠️ 宿主机 2 项已知环境失败（`TestDefaultValues`、`TestConfigGetExistingKey`）；干净 CI 通过 |
| G3 竞态 | `go test -race -count=1 ./...` | ✅ 无 DATA RACE；仅 2 项已知环境失败 |
| G4 覆盖率 | diff-cover（harness 代码 core/plugin） | ✅ N/A（Phase 0 无 harness 代码；工具修复+门禁限定） |
| G5 架构 | `bash scripts/check_arch.sh` | ✅ PASS（11 规则） |
| G6 Skill | `bash scripts/check_skills.sh` | ✅ validated 11 |
| G7 文档/任务 | `bash scripts/check_task_checkboxes.sh` | ✅ 8 board(s) |
| G8/CI | PR #1 required checks | ✅ 全绿（lint/build×3/test/arch/skills/diff-coverage） |
| G9 集成 | `go test ./tests/integration/...` | ✅ PASS（fixture 已提交） |

## 3. 验收标准（DoD）

- [x] `go build ./... && go vet ./...` 通过
- [x] v1 基线报告完整，已知失败项逐一列明原因（`docs/baseline-v1.md` §3）
- [x] CI 在 PR 上全绿（PR #1，所有 required checks 通过）
- [x] 目录骨架存在且可编译
- [x] 规范模板齐全
- [x] G1/G3 本地门禁真正可复现且通过（本次补录）
- [x] `tasks/acceptance/phase-0.md` 已产出并含验收证据

## 4. 证据

- 构建：`go build ./...` / `go vet ./...` → exit 0
- 包存在：`go list ./core/... ./plugin/... ./cmd/sup/... ./cmd/supd/...` → 4 包
- 门禁脚本输出：
  - `check_arch.sh` → `PASS: All architecture rules satisfied`（规则 1–11）
  - `check_skills.sh` → `validated 11 skill manifest(s)`（exit 0）
  - `check_task_checkboxes.sh` → `task checklists OK (8 board(s))`
- Lint：`golangci-lint run ./...` → `0 issues.`（exit 0）；修复前 464 issues（见 §7 ADR）
- 测试基线：25 包 `ok`；宿主机 2 项失败（`TestDefaultValues`、`TestConfigGetExistingKey`）根因为宿主机 `~/.supcode/config.yaml`（provider=deepseek），在干净 CI 上通过。原基线报告的另 2 项（`TestMissingAPIKey`、`TestNewSupCode_MissingAPIKey`）经复核为**陈旧测试**（API key 校验已按设计推迟到 Agent 层），已更正。
- 竞态：修复 `internal/skill`（生产）、`internal/contextmgr`、`internal/hooks/builtin`（测试）共 3 处 data race；CI 首轮追加修复 `internal/mcp/stdio.go`（Linux 触发）。
- CI 首轮（PR #1）：build/arch/skills 通过；追加修复 lint action 版本、mcp 竞态、Windows 专属 `cmd` mock、未提交的 `tests/fixturess` fixture。
- 参考：`docs/baseline-v1.md`、`docs/adr/ADR-0000-phase-0-gate-remediation.md`

## 5. 推迟 / Backlog

| 项 | 原因 | 目标版本 |
|---|---|---|
| 移除 `internal/depslock` 的 cgo sqlite 空白导入 | 非阻塞，属清理项 | Phase 7 |
| `.so`/WASM 插件后端 | 架构规划推迟项 | 2.1 |
| 2 项宿主机环境测试失败改为隔离 HOME 断言 | 非本次范围，属测试健壮性改进 | 2.1（可选） |

## 6. 结论

- 状态：**PASSED（本地 + CI）**
- 合并与发布：将 `phase/0-engineering` 合并到 `main`，打 tag `v2.0.0-alpha.0`。
- 签署：opencode Phase 0 执行/复核 Agent（2026-09-13 / 2026-09-18）
