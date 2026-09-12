# Tool Pool 改造 — Phase 0 基线

日期：2026-09-10
命令：`go test ./...`（环境变量 `SUPCODE_LLM_API_KEY` 已清空）
完整日志：`tasks/acceptance/toolpool-baseline.log`

## `go build ./...`
通过（无报错）。

## 每包结果

| 包 | 结果 |
|---|---|
| github.com/supcode/supcode/internal | **FAIL** |
| internal/agent | ok |
| internal/cli | **FAIL** |
| internal/config | **FAIL** |
| internal/contextmgr | ok |
| internal/hooks | ok |
| internal/hooks/builtin | ok |
| internal/llm | ok |
| internal/llm/anthropic | ok |
| internal/llm/deepseek | ok |
| internal/llm/ollama | ok |
| internal/llm/openai | ok |
| internal/mcp | ok |
| internal/memory | ok |
| internal/permission | ok |
| internal/skill | ok |
| internal/subagent | ok |
| internal/tools | ok |
| internal/tools/bash | ok |
| internal/tools/editfile | ok |
| internal/tools/glob | ok |
| internal/tools/grep | ok |
| internal/tools/readfile | ok |
| internal/tools/writefile | ok |
| internal/tui | ok |
| internal/worktree | ok |
| pkg | ok |
| tests/integration | ok |

## 已知基线失败（与本任务无关，Phase 5 只需保证"失败原因不变差"）

1. `internal` / `TestNewSupCode_MissingAPIKey`
   - 原因：**测试已过时**。`internal/config/config.go:98-101` 明确注释 "API key validation is deferred to Agent layer"，`Load()` 不再因缺 key 报错，但测试仍断言必须报错。
2. `internal/config` / `TestMissingAPIKey`
   - 原因：同上（同一过时假设）。
3. `internal/config` / `TestDefaultValues/llm.provider`
   - 原因：**本机环境**。存在真实用户配置 `C:\Users\19049\.supcode\config.yaml`（`llm.provider: deepseek`），污染了默认值断言（期望 `openai`）。用临时 `USERPROFILE` 运行时可消除。
4. `internal/cli` / `TestConfigGetExistingKey`
   - 原因：同上，受真实用户配置文件影响。用临时 `USERPROFILE` 运行时可消除。

> 注：`tests/integration` 首次运行曾因等待本地 mock LLM server 超时（131s）出现一次 `TestAgentLoop_ReadFile` 失败，本次基线重跑已通过。属于时序/环境抖动，非确定性失败。

## 结论
- 除上述 4 个已知失败外，其余包全部通过。
- 本任务的新增改动不得引入任何新的失败包/用例。
