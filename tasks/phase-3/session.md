# Phase 3 / session 派发提示词

## 背景与目标
把 v1 模块 `internal/tui/session.go` 迁移为可装载的 Cordis 插件，业务逻辑零改动（仅适配/装配）。
契约见 `docs/adr/ADR-0003-state-interaction.md`、`pkg/servicekeys.go`、`pkg/commands.go`。

## 交付物
- `plugin/session/**/*.go`
- `plugin/session/**/*_test.go`
- `plugin/session/**/sup.plugin.json`

## 实现要求
Options{DBPath string}；Apply: tui.NewSessionManager(DBPath)→Provide→ctx.Effect(CloseAll)。Inject 空。测试覆盖创建/获取/列表/追加/持久化恢复。

## 验收命令
```bash
go build ./...
go test -race ./plugin/session/...
golangci-lint run ./plugin/session/...
```

## 硬约束
- 只改 `plugin/session/`；禁止修改 `core/`、`internal/`、`pkg/`。
- 插件间禁止直接 import，只能经服务键/事件。
- 不运行 git 命令。