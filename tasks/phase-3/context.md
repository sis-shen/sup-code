# Phase 3 / context 派发提示词

## 背景与目标
把 v1 模块 `internal/contextmgr/*` 迁移为可装载的 Cordis 插件，业务逻辑零改动（仅适配/装配）。
契约见 `docs/adr/ADR-0003-state-interaction.md`、`pkg/servicekeys.go`、`pkg/commands.go`。

## 交付物
- `plugin/context/**/*.go`
- `plugin/context/**/*_test.go`
- `plugin/context/**/sup.plugin.json`

## 实现要求
Options{Threshold int; LLM pkg.LLMClient}；有 LLM（或 MaybeUse pkg.ServiceLLM）用 NewManagerWithCompression，否则按 Threshold 用 NewManager/NewManagerWithThreshold→Provide。Inject 空。

## 验收命令
```bash
go build ./...
go test -race ./plugin/context/...
golangci-lint run ./plugin/context/...
```

## 硬约束
- 只改 `plugin/context/`；禁止修改 `core/`、`internal/`、`pkg/`。
- 插件间禁止直接 import，只能经服务键/事件。
- 不运行 git 命令。