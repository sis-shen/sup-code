# Phase 0 — 工程地基与基线冻结 验收报告（v2.0.0-alpha.0）

> 状态：**PASSED（本地门禁）· CI 待验证**
> 执行 Skill：`harness-phase-0-baseline`
> 分支：`main`（本地执行）；集成/PR 分支：`phase/0-engineering`
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
| G2 单元测试 | `go test ./...` | ⚠️ 4 项已知环境失败（原因不变），无新增 |
| G3 竞态 | `go test -race -count=1 ./...` | ✅ 无 DATA RACE；仅 4 项已知环境失败 |
| G4 覆盖率 | diff-cover | ✅ N/A（无新增业务代码） |
| G5 架构 | `bash scripts/check_arch.sh` | ✅ PASS（11 规则） |
| G6 Skill | `bash scripts/check_skills.sh` | ✅ validated 11 |
| G7 文档/任务 | `bash scripts/check_task_checkboxes.sh` | ✅ 8 board(s) |
| G8/CI | 推送 `phase/0-engineering` → CI | ⏳ 待推送验证 |

## 3. 验收标准（DoD）

- [x] `go build ./... && go vet ./...` 通过
- [x] v1 基线报告完整，已知失败项逐一列明原因（`docs/baseline-v1.md` §3）
- [ ] CI 在 PR 上全绿 —— **待推送/PR 后验证**
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
- 测试基线：25 包 `ok`；4 项失败（`TestNewSupCode_MissingAPIKey`、`TestConfigGetExistingKey`、`TestDefaultValues`、`TestMissingAPIKey`）根因为宿主机 `~/.supcode/config.yaml`（provider=deepseek）与 `SUPCODE_LLM_API_KEY`。
- 竞态：修复 3 处 data race（`internal/skill` 生产代码，`internal/contextmgr`、`internal/hooks/builtin` 测试）。
- 参考：`docs/baseline-v1.md`、`docs/adr/ADR-0000-phase-0-gate-remediation.md`

## 5. 推迟 / Backlog

| 项 | 原因 | 目标版本 |
|---|---|---|
| 移除 `internal/depslock` 的 cgo sqlite 空白导入 | 非阻塞，属清理项 | Phase 7 |
| `.so`/WASM 插件后端 | 架构规划推迟项 | 2.1 |
| 4 项宿主机环境测试失败改为隔离 HOME 断言 | 非本次范围，属测试健壮性改进 | 2.1（可选） |

## 6. 结论

- 状态：**PASSED（本地）· PENDING CI**
- 推进条件：推送 `phase/0-engineering` 并开启 PR，确认 `ci` 的 `lint/build/test/arch/skills` 全绿（`test` 允许既有 4 项失败），随后打 tag `v2.0.0-alpha.0`。
- 签署：opencode Phase 0 执行/复核 Agent（2026-09-13 / 2026-09-18）
