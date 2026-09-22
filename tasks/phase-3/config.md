# Phase 3 / config 派发提示词

## 背景与目标
把 v1 模块 `internal/config/*` 迁移为可装载的 Cordis 插件，业务逻辑零改动（仅适配/装配）。
契约见 `docs/adr/ADR-0003-state-interaction.md`、`pkg/servicekeys.go`、`pkg/commands.go`。

## 交付物
- `plugin/config/**/*.go`
- `plugin/config/**/*_test.go`
- `plugin/config/**/sup.plugin.json`

## 实现要求
Options{Path string}；Apply: config.NewManager()→(Path!='' 时 SetConfigFile)→Load()→Provide。兼容 ~/.supcode/config.yaml、项目级、SUPCODE_* 环境变量。Inject 空。

## 验收命令
```bash
go build ./...
go test -race ./plugin/config/...
golangci-lint run ./plugin/config/...
```

## 硬约束
- 只改 `plugin/config/`；禁止修改 `core/`、`internal/`、`pkg/`。
- 插件间禁止直接 import，只能经服务键/事件。
- 不运行 git 命令。