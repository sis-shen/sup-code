# Phase 3 / cli 派发提示词

## 背景与目标
把 v1 模块 `internal/cli/* + internal/tui/completions.go` 迁移为可装载的 Cordis 插件，业务逻辑零改动（仅适配/装配）。
契约见 `docs/adr/ADR-0003-state-interaction.md`、`pkg/servicekeys.go`、`pkg/commands.go`。

## 交付物
- `plugin/cli/**/*.go`
- `plugin/cli/**/*_test.go`
- `plugin/cli/**/sup.plugin.json`

## 实现要求
plugin/command 实现内存 CommandRegistry 并预注册 help/version；plugin/cli 提供 headless InteractionService（注入 sessions+commands），并导出 SingleShot(ctx, llm, ctxMgr, sessionMgr, sessionID, input) 用 mock LLM 演示单发查询。禁止 plugin/cli import plugin/command，经服务键交互。

## 验收命令
```bash
go build ./...
go test -race ./plugin/cli/...
golangci-lint run ./plugin/cli/...
```

## 硬约束
- 只改 `plugin/cli, command/`；禁止修改 `core/`、`internal/`、`pkg/`。
- 插件间禁止直接 import，只能经服务键/事件。
- 不运行 git 命令。