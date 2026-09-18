# Phase 0 — 工程地基与基线冻结（执行 Skill）

## 0. 角色与目的
你是 Sup Harness 2.0 的 **Phase 0 执行 Agent**。本阶段不实现任何 2.0 业务能力，只做两件事：
1. 冻结 v1 行为基线，作为后续所有阶段的回归对照组；
2. 搭好 2.0 的目录、CI、规范地基，让后续阶段"有据可依、有门可过"。

## 1. 前置依赖与启动条件
- 架构方案（`docs/Sup Harness 2.0 架构重整方案.md`）与本规划文档已存在。
- 无上一阶段 Gate。
- 启动即表示：允许创建分支、目录、CI 文件，但**不得改动 v1 业务代码**。

## 2. 阶段目标
- 产出可复现的 v1 基线报告。
- 产出能跑的 `ci.yml` 与门禁脚本。
- 产出统一的提交/分支/文档/任务提示词规范。

## 3. 功能点清单
| 功能点 | 状态 |
|---|---|
| 跑 `go build/vet/test` 并记录 v1 基线（含已知失败项与原因） | 实现 |
| 建立 2.0 目录骨架 `core/` `plugin/` `cmd/sup/` `cmd/supd/`（占位 doc.go） | 实现 |
| `.github/workflows/ci.yml` 骨架 | 实现 |
| `scripts/check_skills.sh` / `scripts/check_task_checkboxes.sh` | 实现 |
| `.gitattributes`（`*.sh` 强制 LF）、PR 模板、CODEOWNERS | 实现 |
| 分支保护与 required checks 配置说明 | 实现 |
| 阶段报告/任务提示词/ADR 模板 | 实现 |
| 移除 `internal/depslock` 对 cgo sqlite 的空白导入评估 | 实现（可延后 P1） |

## 4. 执行顺序与并行性
1. **串行（先做）**：基线测量（必须得到干净的对照数据）。
2. **并行**：目录骨架 / CI 骨架 / 规范模板 三组互不冲突，可同时进行。
3. **串行（收尾）**：本地跑一遍全部门禁命令，确认 CI 与本机一致。

## 5. 子 Agent 自动派发计划
| 子 Agent | 任务 | 分支 | 交付物 |
|---|---|---|---|
| `baseline-runner` | 跑基线并写报告 | `phase/0/baseline` | `docs/baseline-v1.md` |
| `repo-scaffolder` | 建目录骨架 | `phase/0/scaffold` | `core/doc.go` 等占位 |
| `ci-builder` | 写 CI 与门禁脚本 | `phase/0/ci` | `.github/workflows/ci.yml`、`scripts/*.sh` |
| `docs-templater` | 模板与规范 | `phase/0/templates` | `docs/templates/*`、PR 模板 |

**合并策略**：四个子 Agent 完成后，由阶段 Agent 串行合并到 `phase/0-engineering`，再统一跑门禁。

## 6. Git 规范
- 分支：`phase/0-engineering`（集成），子任务 `phase/0/<task>`。
- commit：`chore(phase0): ...` / `ci: ...` / `docs: ...`。
- 禁止直接 push `main`。

## 7. 文档与任务提示词维护
- 必须产出：`docs/baseline-v1.md`、`docs/templates/{phase-report,task-prompt,adr}.md`、`tasks/phase-0/README.md`、`tasks/acceptance/phase-0.md`。
- 每个子 Agent 派发前生成 `tasks/phase-0/<task>.md`（用 task-prompt 模板）。

## 8. 验收标准（DoD）
- [ ] `go build ./... && go vet ./...` 通过
- [ ] v1 基线报告完整，已知失败项逐一列明原因
- [ ] CI 在 PR 上全绿（lint/build/test/race/arch/skills）
- [ ] 目录骨架存在且可编译
- [ ] 规范模板齐全
- [ ] `tasks/acceptance/phase-0.md` 已产出并含验收证据

## 9. 门禁与命令
| 门禁 | 命令 |
|---|---|
| G0/G2 | `go build ./... && go vet ./... && go test ./...` |
| G1 | `bash scripts/lint.sh` |
| G5 | `bash scripts/check_arch.sh` |
| G6 | `bash scripts/check_skills.sh` |
| G7 | `bash scripts/check_task_checkboxes.sh` |

## 10. 失败/回滚
- 基线中出现 v1 既有失败：记录为"已知基线失败"，不视为本阶段失败；
- CI 与本机结果不一致：先修到一致再宣布完成；
- 回滚：删除新增骨架/CI 文件即可，不触碰 v1。

## 11. 输出物
`docs/baseline-v1.md`、`.github/workflows/ci.yml`、`scripts/check_*.sh`、`docs/templates/*`、`tasks/acceptance/phase-0.md`、tag `v2.0.0-alpha.0`。
