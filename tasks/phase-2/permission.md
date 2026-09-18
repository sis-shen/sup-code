# Phase 2 / plugin-permission 派发提示词

## 背景与目标
把 v1 模块 `internal/permission/*` 迁移为可装载的 Cordis 叶子插件 `plugin/permission`，**业务逻辑零改动**（只加适配/装配/事件桥接）。

## 权威输入
- 契约：`docs/adr/ADR-0002-plugin-contract.md`、`pkg/servicekeys.go`、`pkg/pipeline.go`
- 内核：`core/`（**禁止修改**）；v1 来源：`internal/permission/*`
- 阶段 Skill：`skills/harness-phase-2-leaf-plugins/prompts/system.md`

## 交付物
- `plugin/permission/*.go`
- `plugin/permission/plugin_test.go`
- `plugin/permission/sup.plugin.json`

## 实现要求
- 导出 `Plugin(opts ...Option) core.Plugin` 与 `Manifest() core.Manifest`。
- 服务：`pkg.ServicePermission` → `pkg.PermissionEngine`。
- Provide internal/permission.New()；监听 pkg.EventToolsPreExecute（Waterfall），对 ToolInvocation 构造 pkg.Action 调 Check；Deny 返回 pkg.ErrPermissionDenied 短路；调 LogAction。

## 验收命令
```bash
go build ./...
go test -race ./plugin/permission/...
golangci-lint run ./plugin/permission/...
```

## 完成定义
- [ ] 提供 `pkg.ServicePermission` 服务，`Manifest()` 与 `sup.plugin.json` 一致
- [ ] 单测通过（`-race`）
- [ ] 未修改 `core/`；无 `plugin/X → plugin/Y` import