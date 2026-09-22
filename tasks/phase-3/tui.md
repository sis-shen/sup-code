# Phase 3 / tui 派发提示词

## 背景与目标
把 v1 模块 `internal/tui/service.go` 迁移为可装载的 Cordis 插件，业务逻辑零改动（仅适配/装配）。
契约见 `docs/adr/ADR-0003-state-interaction.md`、`pkg/servicekeys.go`、`pkg/commands.go`。

## 交付物
- `plugin/tui/**/*.go`
- `plugin/tui/**/*_test.go`
- `plugin/tui/**/sup.plugin.json`

## 实现要求
Inject [pkg.ServiceSessions]；Apply: sm:=Use[pkg.SessionManager]；svc:=tui.NewService(sm)；可选 MaybeUse agent→SetAgent；Provide interaction。覆盖 StreamResponse/RequestConfirmation/ReadInput/Notify/HandleCommand（不启动 Bubble Tea 即可测的部分）。

## 验收命令
```bash
go build ./...
go test -race ./plugin/tui/...
golangci-lint run ./plugin/tui/...
```

## 硬约束
- 只改 `plugin/tui/`；禁止修改 `core/`、`internal/`、`pkg/`。
- 插件间禁止直接 import，只能经服务键/事件。
- 不运行 git 命令。