# Phase 2 / plugin-llm 派发提示词

## 背景与目标
把 v1 模块 `internal/llm/*（factory.go + openai/anthropic/deepseek/ollama）` 迁移为可装载的 Cordis 叶子插件 `plugin/llm`，**业务逻辑零改动**（只加适配/装配/事件桥接）。

## 权威输入
- 契约：`docs/adr/ADR-0002-plugin-contract.md`、`pkg/servicekeys.go`、`pkg/pipeline.go`
- 内核：`core/`（**禁止修改**）；v1 来源：`internal/llm/*（factory.go + openai/anthropic/deepseek/ollama）`
- 阶段 Skill：`skills/harness-phase-2-leaf-plugins/prompts/system.md`

## 交付物
- `plugin/llm/*.go`
- `plugin/llm/plugin_test.go`
- `plugin/llm/sup.plugin.json`

## 实现要求
- 导出 `Plugin(opts ...Option) core.Plugin` 与 `Manifest() core.Manifest`。
- 服务：`pkg.ServiceLLM` → `pkg.LLMClient`。
- Plugin(opts) 用 llm.NewLLMClient(llm.Config{...}) 构造并 Provide；Options 允许直接传 pkg.LLMClient；可选读取 pkg.ServiceConfig。Inject 为空。

## 验收命令
```bash
go build ./...
go test -race ./plugin/llm/...
golangci-lint run ./plugin/llm/...
```

## 完成定义
- [ ] 提供 `pkg.ServiceLLM` 服务，`Manifest()` 与 `sup.plugin.json` 一致
- [ ] 单测通过（`-race`）
- [ ] 未修改 `core/`；无 `plugin/X → plugin/Y` import