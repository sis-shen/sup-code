# Phase 2 / plugin-hooks 派发提示词

## 背景与目标
把 v1 模块 `internal/hooks/builtin/*` 迁移为可装载的 Cordis 叶子插件 `plugin/hooks`，**业务逻辑零改动**（只加适配/装配/事件桥接）。

## 权威输入
- 契约：`docs/adr/ADR-0002-plugin-contract.md`、`pkg/servicekeys.go`、`pkg/pipeline.go`
- 内核：`core/`（**禁止修改**）；v1 来源：`internal/hooks/builtin/*`
- 阶段 Skill：`skills/harness-phase-2-leaf-plugins/prompts/system.md`

## 交付物
- `plugin/hooks/*.go`
- `plugin/hooks/plugin_test.go`
- `plugin/hooks/sup.plugin.json`

## 实现要求
- 导出 `Plugin(opts ...Option) core.Plugin` 与 `Manifest() core.Manifest`。
- 服务：`（不提供）` → `—`。
- Inject ['permission']；AuditPlugin 监听 pkg.EventToolsResult（Emit）复用 builtin.NewAuditHook(perm).AfterTool；GitPlugin(enabled) 监听 pkg.EventToolsPostExecute（Waterfall）复用 builtin.NewGitCommitHook(enabled).AfterTool。

## 验收命令
```bash
go build ./...
go test -race ./plugin/hooks/...
golangci-lint run ./plugin/hooks/...
```

## 完成定义
- [ ] 提供 `（不提供）` 服务，`Manifest()` 与 `sup.plugin.json` 一致
- [ ] 单测通过（`-race`）
- [ ] 未修改 `core/`；无 `plugin/X → plugin/Y` import