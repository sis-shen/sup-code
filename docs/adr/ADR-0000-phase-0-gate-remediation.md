# ADR-0000：Phase 0 门禁补救决策

- 状态：Accepted
- 日期：2026-09-18
- 决策者：编排 Agent（用户确认）
- 关联阶段：Phase 0

## 背景

Phase 0 的验收报告最初标注为「PASSED（本地）· CI 待验证」，但实际复核（本次编排 PRECHECK）发现本地门禁并未真正全绿，且已提交的规划假设与仓库实况不符：

1. **G1 静态检查**：v1 代码从未在 CI 上通过过。`ci.yml` 的 lint job 使用 `golangci-lint-action@v6` + `version: latest`（即 v2），而仓库 `.golangci.yml` 仍是 v1 配置格式，CI 会因配置版本不兼容而直接失败。本机实测（v2.13.2）共有 **464** 条现存问题。
2. **G3 竞态**：原验收报告写「待 gate CI」，从未本地执行。本次实测发现 **3 处 data race**（`internal/skill` 生产代码 1 处、`internal/contextmgr` 与 `internal/hooks/builtin` 测试各 1 处）。
3. **G6 Skill 校验**：`scripts/check_skills.sh` 依赖 `python3`，在 Windows 上 `python3` 解析到 Microsoft Store 占位程序（非交互调用返回非零），导致 11 个合法 manifest 全部误报 FAIL。
4. **G5 架构**：`scripts/check_arch.sh` 只有 v1 的 6 条规则，缺少架构方案 §6 要求的 v2 规则（core/plugin/pkg 隔离、插件互不 import）。

## 候选方案

| 方案 | 优点 | 缺点 |
|---|---|---|
| A. 保持 v1 原样，把 G1/G3 继续甩给 CI | 不动 v1 代码 | CI 必然红；门禁形同虚设；违反「门禁优先」 |
| B. 只修复新增代码，用 baseline diff 放宽 G1 | 改动小 | 与「lint 0 issue」冲突；CI 若按全量 lint 仍红；隐性债务 |
| C. 全量修复 464 条 lint + 3 处竞态，门禁真正本地可复现 | 门禁可信、CI 可绿、长期收益 | 触碰全部 v1 包，diff 较大 |

## 决策

采用 **方案 C**（经用户确认）：

1. `.golangci.yml` 迁移到 v2 格式（`golangci-lint migrate`），与 CI 的 `version: latest` 对齐；删除迁移备份 `.golangci.bck.yml`。
2. 修复全部 **464** 条 lint 问题至 **0**（含 gofmt/goimports 自动修复与 errcheck/unused/unparam/staticcheck/ineffassign 手工修复；测试内 errcheck 用 `require.NoError`，生产代码用传播错误/日志）。
3. 本地执行 G3 并修复 3 处 data race：
   - `internal/skill/loader.go`：缓存 map 增加 `sync.RWMutex`；
   - `internal/skill/watcher.go`：`Start/Close` 增加 `started atomic.Bool` + `done chan`，先停事件循环再关 `Changes`，消除 send-on-closed/close 竞态；
   - `internal/hooks/builtin/audit_test.go`：mock 增加互斥与快照访问；`internal/contextmgr/compressor_test.go`：`callCount` 改 `atomic.Int32`。
4. `scripts/check_skills.sh` 增加 Python 解释器探测（`python3` → `python` → `py -3`），Windows 兼容。
5. `scripts/check_arch.sh` 增加 v2 规则 7–11（`core` 不 import `plugin/`、`internal/`；`pkg` 不 import `core/`、`plugin/`；`plugin/X` 不 import 其他 `plugin/Y`）。
6. 4 项宿主机环境导致的测试失败（`TestDefaultValues`、`TestConfigGetExistingKey`、`TestMissingAPIKey`、`TestNewSupCode_MissingAPIKey`）**维持为已知基线失败**，判据不变。

## 后果

- 正面：
  - G0/G1/G3/G5/G6/G7 均可本地复现且全绿，CI 具备首次全绿的条件；
  - 修复了 `SkillLoader` 缓存与 `Watcher` 生命周期的真实并发缺陷；
  - 架构门禁从此覆盖 v2 约束，后续 Phase 2+ 可自动拦截越界 import。
- 负面/代价：
  - 违背了规划中 Phase 0「旧 `internal/*` 保持不动」的表述；本阶段对 v1 代码做了 lint/并发修复，diff 涉及 100+ 文件（其中大部分是 gofmt/goimports 纯格式变更）。
  - 后续「搬运优先」审计需以本次提交为新基线。
- 对 dsh/Cordis 兼容性的影响：无（不涉及服务键/事件面）。

## 后续

- B1（Backlog）：Phase 7 复核 lint 与覆盖率门禁。
- Phase 1 起，新增代码必须保持 lint 0；CI `lint` job 作为 required check 固化。

## 补充（2026-09-18，CI PR #1 首轮）

首轮 CI 证明 v1 在干净环境**从未全绿**，暴露并修复以下问题：

1. **lint action 版本**：`golangci-lint-action@v6` 的 `version: latest` 解析为 v1.64.8，无法分析 go1.25。升级为 `@v9` + `version: v2.13.2`，与本地一致。
2. **`internal/mcp` 竞态**：`stdioTransport` 在进程重启路径写 `cmd/stdin/stdout/stderr` 与 `Send/Close` 读并发；Windows 上未触发、Linux CI 触发。已将所有字段读写纳入 `t.mu`。
3. **测试可移植性**：`installer_mock_test.go` 使用 Windows 专属 `cmd /c`，Linux 必失败；改为 `runtime.GOOS` 分支。
4. **未提交的测试 fixture**：`tests/fixturess/` 被 `.gitignore` 忽略，导致 `tests/integration` 在 CI 全部失败。移除忽略项并提交 fixture（删除内层 `.git`）。
5. **陈旧测试更正**：`TestMissingAPIKey`、`TestNewSupCode_MissingAPIKey` 假设 `config.Load`/`NewSupCode` 校验 API key，但代码已按设计将校验推迟到 Agent 层（`internal/config/config.go:99`）。改为断言延迟校验成功；基线文档 §3 相应更正（真正的环境性失败由 4 项降为 2 项，且在干净 CI 上通过）。

教训：Phase 0 的「基线冻结」必须包含**在干净环境执行 G1/G3/G9**，不能把未执行的门禁记为通过；本次已把该原则固化到流程。
