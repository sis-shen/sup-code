# Phase 2 / plugin-worktree 派发提示词

## 背景与目标
把 v1 模块 `internal/worktree/*` 迁移为可装载的 Cordis 叶子插件 `plugin/worktree`，**业务逻辑零改动**（只加适配/装配/事件桥接）。

## 权威输入
- 契约：`docs/adr/ADR-0002-plugin-contract.md`、`pkg/servicekeys.go`、`pkg/pipeline.go`
- 内核：`core/`（**禁止修改**）；v1 来源：`internal/worktree/*`
- 阶段 Skill：`skills/harness-phase-2-leaf-plugins/prompts/system.md`

## 交付物
- `plugin/worktree/*.go`
- `plugin/worktree/plugin_test.go`
- `plugin/worktree/sup.plugin.json`

## 实现要求
- 导出 `Plugin(opts ...Option) core.Plugin` 与 `Manifest() core.Manifest`。
- 服务：`pkg.ServiceWorktree` → `pkg.WorktreeManager`。
- Provide worktree.NewManager(opts.RepoPath)。

## 验收命令
```bash
go build ./...
go test -race ./plugin/worktree/...
golangci-lint run ./plugin/worktree/...
```

## 完成定义
- [ ] 提供 `pkg.ServiceWorktree` 服务，`Manifest()` 与 `sup.plugin.json` 一致
- [ ] 单测通过（`-race`）
- [ ] 未修改 `core/`；无 `plugin/X → plugin/Y` import