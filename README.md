 # SupCode
 
 [![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
 [![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
 
 **终端原生的 AI 编程助手。** 用自然语言写代码，Agent 自己读文件、改代码、跑命令。
 
 ```
 $ supcode "把 utils.ts 里的所有 any 改成具体类型"
 
 Thinking... 正在分析 utils.ts
 读取文件... 找到 3 处 any 类型声明
 正在替换... 完成
 运行 tsc --noEmit... 0 errors
 
 已将 3 处 any 替换为具体类型，类型检查通过。
 ```
 
 ---
 
 ## 特性
 
 - **模型无关** —— OpenAI / Claude / DeepSeek / 本地 Ollama，随意切换
 - **终端原生** —— 纯 CLI + TUI，SSH 到服务器也能用
 - **自主执行** —— Agent 自己规划任务、选择工具、修正错误、交付结果
 - **安全可控** —— 五层权限防御，危险操作必须确认
 - **生态扩展** —— MCP 协议接入外部工具，Skill 技能包复用专业能力
 - **并行协作** —— SubAgent 拆分大任务并行执行
 
 ---
 
 ## 快速开始
 
 ```bash
 # 安装
 go install github.com/supcode/supcode/cmd/supcode@latest
 
 # 配置
 supcode config set llm.provider openai
 supcode config set llm.api_key sk-your-key-here
 
 # 开始
 supcode "解释这个项目"
 ```
 
 更多安装方式见 [产品使用说明书](docs/产品使用说明书.md#2-安装)。
 
 ---
 
 ## 文档
 
 | 文档 | 说明 |
 |------|------|
 | [产品使用说明书](docs/产品使用说明书.md) | 安装、配置、交互模式、Slash 命令、Skills、MCP |
 | [产品设计说明书](docs/产品设计说明书.md) | 产品定位、用户画像、核心价值主张、竞争分析 |
 | [软件需求规格说明书](docs/软件需求规格说明书.md) | 功能需求、非功能需求、接口定义、分阶段计划 |
 | [系统架构设计说明书](docs/系统架构设计说明书.md) | 五层架构模型、层间数据流、分阶段演进、Go 包映射 |
 | [接口契约设计说明书](docs/接口契约设计说明书.md) | 所有层 interface 定义、Mock 契约、错误类型、协作指南 |
 | [本地自测说明书](docs/本地自测说明书.md) | 自测命令、覆盖率门禁、Mock 策略、提交流程 |
 
 ---
 
 ## 架构
 
 ```
 第1层 交互层  ── TUI / Slash Command / Skill 加载 / 多轮对话
 第2层 引擎层  ── Agent Loop / LLM Client / SubAgent / Plan Mode
 第3层 工具层  ── 六大内置工具 / MCP 外部工具 / Hook 钩子
 第4层 记忆层  ── 上下文压缩 / Token 管理 / 长期记忆
 第5层 安全层  ── 五层权限防御 / Worktree 隔离 / 审计日志
 ```
 
 详细见 [系统架构设计说明书](docs/系统架构设计说明书.md)。
 
 ---
 
 ## 开发
 
 ```bash
 git clone https://github.com/supcode/supcode.git
 cd supcode
 
 # 编译
 go build ./cmd/supcode
 
 # 运行测试
 go test ./... -count=1 -race
 
 # 自测
 bash scripts/test.sh
 bash scripts/lint.sh
 bash scripts/check_arch.sh
 ```
 
 任务分发给子 Agent 的开发模式见 `tasks/` 目录。
 
 ---
 
 ## License
 
 [MIT](LICENSE)
