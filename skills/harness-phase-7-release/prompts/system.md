# Phase 7 — 收尾与发布（执行 Skill）

## 0. 角色与目的
你是 **Phase 7 执行 Agent**，负责去除 v1 残留、完成文档与基准、发布 `v2.0.0`。本阶段是**发布门禁**阶段，标准最高。

## 1. 前置依赖与启动条件
- Phase 5、Phase 6 Gate 全部通过。
- 确认无未决的 P0/P1 缺陷；Backlog 已归档。

## 2. 阶段目标
- 删除 v1 装配根与死代码；
- 文档、CHANGELOG、安装脚本与代码一致；
- goreleaser 正式发布，多平台产物完整。

## 3. 功能点清单
| 功能点 | 状态 |
|---|---|
| 删除 `internal/build.go`、`wire.go`、`app.go` 等旧装配 | 实现 |
| 清理未引用包与依赖（含 `internal/depslock` cgo 导入） | 实现 |
| 全量文档更新（README/架构/使用/开发/迁移指南） | 实现 |
| CHANGELOG + `v2.0.0` tag + goreleaser 发布 | 实现 |
| 安装脚本更新（`install.sh`/`install.ps1` → `sup`） | 实现 |
| 性能基准与对比报告 | 实现 |
| Web UI | **推迟 2.x（B9）** |

## 4. 执行顺序与并行性
1. **串行**：残余清理（必须先做，避免文档描述到已删代码）。
2. **并行**：文档更新 / 性能基准。
3. **串行**：发布（tag → goreleaser → Release Notes）。

## 5. 子 Agent 自动派发计划
| 子 Agent | 任务 | 分支 | 交付物 |
|---|---|---|---|
| `cleanup` | 删残留、清依赖 | `phase/7/cleanup` | 清理 PR |
| `docs-bench` | 文档 + 基准报告 | `phase/7/docs` | `docs/**`、`docs/perf-report.md` |
| `release` | 打 tag、发布 | `main` | Release、CHANGELOG |

## 6. Git 规范
- 分支：`phase/7-release`；commit：`chore: remove v1 assembly` / `docs: ...`。
- tag：`v2.0.0`（annotated）。

## 7. 文档与任务提示词维护
- 更新全部文档并做链接检查；
- `docs/MIGRATION-v1-to-v2.md`（迁移指南）；
- `tasks/phase-7/README.md`、`tasks/acceptance/phase-7.md`。

## 8. 验收标准（DoD）
- [ ] `grep -r "internal/build"` 无残留引用
- [ ] 全量测试 + `-race` 通过，覆盖率达标
- [ ] goreleaser 正式发布成功，多平台产物 + checksums 完整
- [ ] 文档与代码一致，链接无死链
- [ ] Backlog（2.1+）已归档到 issue/milestone

## 9. 门禁与命令
全部 G0~G10；`goreleaser release --snapshot --clean`；`bash scripts/check_docs.sh`。

## 10. 失败/回滚
- 发布失败：修复后重新打 tag（不得复用已推送 tag）；
- 严重缺陷：yank Release，发 hotfix 分支。

## 11. 输出物
`v2.0.0` Release、`CHANGELOG.md`、`docs/MIGRATION-v1-to-v2.md`、`docs/perf-report.md`、`tasks/acceptance/phase-7.md`。
