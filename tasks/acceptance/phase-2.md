# Phase 2 — 叶子插件化 验收报告（v2.0.0-alpha.2）

> 状态：**PASSED（本地门禁）· CI 待验证**
> 执行 Skill：`harness-phase-2-leaf-plugins`
> 分支：`phase/2-plugins`
> 生成时间：2026-09-18
> 契约：`docs/adr/ADR-0002-plugin-contract.md`

## 1. 交付物清单

| 交付物 | 路径 | 是否存在 |
|---|---|---|
| plugin-llm（seam + 4 适配器） | `plugin/llm/**` | ✅ |
| plugin-tools + 6 工具 + 流水线事件桥 | `plugin/tools/**` | ✅ |
| plugin-permission | `plugin/permission/**` | ✅ |
| plugin-hooks（audit/git） | `plugin/hooks/**` | ✅ |
| plugin-mcp | `plugin/mcp/**` | ✅ |
| plugin-skill + tool-skill | `plugin/skill/**` | ✅ |
| plugin-memory | `plugin/memory/**` | ✅ |
| plugin-worktree | `plugin/worktree/**` | ✅ |
| 共享契约 | `pkg/servicekeys.go`、`pkg/pipeline.go`、`docs/adr/ADR-0002-plugin-contract.md` | ✅ |
| 派发提示词 | `tasks/phase-2/{llm,tools,mcp,skill,permission,memory,worktree,hooks}.md` | ✅ |
| 集成测试 | `tests/integration/plugin_chain_test.go` | ✅ |

## 2. 门禁结果（本地实测）

| 门禁 | 命令 | 结果 |
|---|---|---|
| G0 编译 | `go build ./... && go vet ./...` | ✅ exit 0 |
| G1 静态检查 | `golangci-lint run ./...` | ✅ 0 issues |
| G2 单元测试 | `go test ./...` | ✅ 通过（宿主机 2 项已知环境失败除外） |
| G3 竞态 | `go test -race ./plugin/... ./tests/integration/...` | ✅ PASS |
| G4 覆盖率 | 插件包覆盖 | ✅ 全部 ≥82.6%（llm/mcp/worktree 100%） |
| G5 架构 | `bash scripts/check_arch.sh` | ✅ PASS（规则 11 无跨插件 import） |
| G6 Skill | `bash scripts/check_skills.sh` | ✅ validated 11 |
| G7 文档/任务 | `bash scripts/check_task_checkboxes.sh` | ✅ OK |
| G8/CI | PR 门禁 | ⏳ 待推送验证 |

## 3. 验收标准（DoD）

- [x] 每个插件单测通过（`-race`），v1 模块测试保持通过
- [x] 业务逻辑未重写：仅新增适配/装配/事件桥，`internal/*` 只做两处经授权的契约补充
- [x] 每个插件由 `sup.plugin.json` 声明，`Manifest()` 与之一致，可经 `inject` 拓扑装载
- [x] 无对 `internal/*` 的新反向依赖；无 `plugin/X → plugin/Y` import（G5 规则 11）
- [x] 集成测试：6 内置工具 + skill 工具同池可用，且经工具流水线执行
- [x] G4 覆盖率门禁（插件均 ≥80%）

## 4. 证据

- 覆盖率：`go test -cover ./plugin/...`
  - llm 100.0%、mcp 100.0%、worktree 100.0%、demo 94.7%、tools 93.1%、memory 92.3%、permission 90.9%、skill 87.5%、hooks 82.6%
- 架构：`check_arch.sh` → `PASS`（规则 1–11）
- 集成：`go test -race ./tests/integration/ -run TestPluginPool` → `ok`
- 拓扑装载：`LoadPlugins` 以 `Provides→key` 映射顺序装载 permission → tools → skill（测试乱序传入）
- 工具流水线：`plugin/tools` 的装饰器按 `pre-execute → execute → post-execute(→result)` 分发事件；`plugin/permission` 在 pre-execute 短路
- 契约来源：`pkg/servicekeys.go`、`pkg/pipeline.go`、`docs/adr/ADR-0002-plugin-contract.md`

## 5. 推迟 / Backlog

| 项 | 原因 | 目标版本 |
|---|---|---|
| 新 LLM/Embedding 适配器（Gemini/Azure 等） | 规划推迟（B6） | 2.1 |
| WASM / 动态 `.so` 后端 | 规划推迟（B4/B5） | 2.1 |
| `plugin-config` 与各插件从"可选配置"改为 `inject config` | 属 Phase 3 | Phase 3 |
| `plugin-tools` 的 `tools/execute` 目前为观察/改写包装，未接入沙箱/超时 | 属后续增强 | 2.x |

## 6. 结论

- 状态：**PASSED（本地）· PENDING CI**
- 推进条件：CI 全绿后合并 `main` 并打 tag `v2.0.0-alpha.2`。
- 签署：opencode Phase 2 编排 Agent + 8 子 Agent（2026-09-18）
