# SupCode v1 基线报告（Phase 0 回归对照）

> 用途：作为 Sup Harness 2.0 后续所有阶段的回归对照组。
> 生成：2026-09-13；执行 Skill：`harness-phase-0-baseline`
> 仓库：`github.com/supcode/supcode`；分支：`main`；提交：`362974b`
> 环境：Windows / amd64，`go version go1.25.0 windows/amd64`

---

## 1. 基线命令与结果

| 门禁 | 命令 | 结果 | 说明 |
|---|---|---|---|
| G0 编译 | `go build ./...` | **PASS**（exit 0） | 无输出 |
| G0 静态 | `go vet ./...` | **PASS**（exit 0） | 无输出 |
| G2 单元测试 | `go test ./...` | **4 失败 / 其余通过** | 全部为环境相关，见 §3 |
| G3 竞态 | `go test -race ./...` | 未执行 | 交由 `gate` CI 执行 |
| G1 静态检查 | `golangci-lint run ./...` | 未执行 | 本机未安装，交由 `ci` CI 执行 |
| G4 覆盖率 | diff-coverage | Phase 0 不适用 | 后续阶段启用 |
| G5 架构 | `bash scripts/check_arch.sh` | **PASS** | 6 条规则全部满足 |
| G6 Skill | `bash scripts/check_skills.sh` | **PASS** | validated 11 skill manifest(s) |
| G7 文档/任务 | `bash scripts/check_task_checkboxes.sh` | **PASS** | task checklists OK (8 board(s)) |

---

## 2. `go test ./...` 逐包结果

**通过（ok）**
```
internal/agent            internal/contextmgr      internal/hooks/builtin
internal/llm              internal/llm/anthropic    internal/llm/deepseek
internal/llm/ollama       internal/llm/openai       internal/mcp
internal/memory           internal/permission       internal/skill
internal/subagent         internal/tools            internal/tools/bash
internal/tools/editfile   internal/tools/glob       internal/tools/grep
internal/tools/readfile   internal/tools/skilltool  internal/tools/writefile
internal/tui              internal/worktree         pkg
tests/integration
```

**无测试文件**
```
cmd/supcode               internal/depslock
tests/fixturess/test_repo/src
```

**失败（FAIL）— 4 项**
```
--- FAIL: TestNewSupCode_MissingAPIKey   (internal/wire_test.go:37)
--- FAIL: TestConfigGetExistingKey       (internal/cli/cli_test.go:81)
--- FAIL: TestDefaultValues              (internal/config/config_test.go:32)
--- FAIL: TestMissingAPIKey              (internal/config/config_test.go:113)
```

---

## 3. 已知失败项根因（环境性，非代码回归）

| 用例 | 现象 | 根因 |
|---|---|---|
| `TestDefaultValues` | `Get("llm.provider") = deepseek, want openai` | 宿主机 `~/.supcode/config.yaml` 设置了 `llm.provider: deepseek`，覆盖了测试期望的默认值 |
| `TestConfigGetExistingKey` | `"deepseek-v4-flash" does not contain "gpt-4o"` | 同上：宿主机用户配置的 model/provider 覆盖测试期望 |
| `TestMissingAPIKey` | `expected error ... got nil` | 宿主机 `SUPCODE_LLM_API_KEY` 已设置（且用户配置含 key），测试无法构造"缺 key"场景 |
| `TestNewSupCode_MissingAPIKey` | `expected error for missing API key` | 同上（环境变量/用户配置提供了 key） |

**证据**：
```
# 宿主机用户配置
C:\Users\19049\.supcode\config.yaml
    model: deepseek-v4-flash
    provider: deepseek

# 环境
SUPCODE_LLM_API_KEY is SET
```

**说明**：
- 清除环境变量后测试仍失败，因为失败并非仅由环境变量引起，而是**宿主机用户配置文件**污染了默认值断言；需在隔离 HOME / 指定临时 `--config` 的环境下才会通过。
- 这 4 项在 2.0 开发全程作为"已知基线失败"接受，判据为：**失败原因不得变化，且不得新增失败项**。

---

## 4. 2.0 目录骨架（本阶段新增）

```
core/doc.go          package core   （Phase 1 实现内核）
plugin/doc.go        package plugin （Phase 2/4 添加插件）
cmd/sup/main.go      sup CLI 占位   （Phase 5 实现）
cmd/supd/main.go     supd 守护占位  （Phase 5 实现）
```
`go list` 验证：`core`、`plugin`、`cmd/sup`、`cmd/supd` 均可编译（`go build ./...` exit 0）。

## 5. 本阶段新增的工程文件

| 文件 | 作用 |
|---|---|
| `.github/workflows/{ci,gate,release,nightly}.yml` | CI/CD |
| `.github/PULL_REQUEST_TEMPLATE.md` | PR 门禁自检 |
| `.github/CODEOWNERS` | 关键路径评审归属（占位待替换） |
| `.gitattributes` | 强制 `*.sh`/`*.yml` 使用 LF |
| `scripts/check_skills.sh` / `check_task_checkboxes.sh` | G6/G7 门禁脚本 |
| `scripts/gen_phase_tasks.*` + `scripts/phase_manifest.json` | 任务看板/验收骨架生成 |
| `docs/templates/{phase-report,task-prompt,adr}.md` | 过程文档模板 |
| `docs/STATUS.md` | 全版本状态看板 |

---

## 6. 结论

- v1 基线已冻结：**G0/G5/G6/G7 通过；G2 存在 4 项已知环境失败；G1/G3 交由 CI**。
- 后续阶段的回归判据：**不得新增失败、既有 4 项失败原因不得变化**。
- 目录与 CI 骨架就绪，Phase 0 DoD 的本地部分完成；**CI 首次全绿**需推送 PR 后确认。

---

## 7. Phase 0 门禁补录（2026-09-18，编排 PRECHECK 实测）

> 背景与决策见 [`adr/ADR-0000-phase-0-gate-remediation.md`](adr/ADR-0000-phase-0-gate-remediation.md)。本节记录对 §1 中「未执行」门禁的实测结论与据此对 v1 代码的必要修复。

### 7.1 G1 静态检查（首次实测）

- 工具：`golangci-lint v2.13.2`（与 CI `version: latest` 对齐）；配置由 v1 迁移为 v2。
- 结果：初次 **464 issues** → 修复后 **0 issues**。
- 分类（初始）：gofmt 132、errcheck 245、goimports 34、unused 16、unparam 17、staticcheck 13、misspell 4、ineffassign 3。
- 影响：`internal/*`、`pkg/*`、`cmd/supcode` 大量格式与 errcheck/unused 清理；行为不变。

### 7.2 G3 竞态（首次实测）

- 命令：`go test -race -count=1 ./...`。
- 初次结果：3 处 DATA RACE。
  | 包 | 位置 | 性质 |
  |---|---|---|
  | `internal/skill` | `loader.go` 缓存 map / `watcher.go` 生命周期 | **生产代码** |
  | `internal/contextmgr` | `compressor_test.go` 共享 `callCount` | 测试 |
  | `internal/hooks/builtin` | `audit_test.go` mock 无锁 | 测试 |
- 修复后：无 DATA RACE；仅剩 §3 的 4 项已知环境失败。

### 7.3 G5/G6

- `check_arch.sh`：新增 v2 规则 7–11，PASS。
- `check_skills.sh`：Python 解释器探测（`python3`/`python`/`py -3`），`validated 11 skill manifest(s)`，PASS。

### 7.4 冻结判据更新

- 原判据（§6）继续有效：**不得新增失败、既有 4 项失败原因不得变化**。
- 新增：以 Phase 0 补救提交为 v2.0 迁移的**新基线**；此后 `golangci-lint run ./...` 必须保持 0 issues。
