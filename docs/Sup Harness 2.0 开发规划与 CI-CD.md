# Sup Harness 2.0 开发规划与 CI/CD 设计

> 副标题：阶段划分 · 功能点 · 验收标准 · 门禁 · GitHub Actions CI/CD · Skill 流程控制
> 文档版本：v1.0（规划稿，不含业务代码改动）
> 上游文档：[《Sup Harness 2.0 架构重整方案》](Sup%20Harness%202.0%20架构重整方案.md)
> 适用仓库：`github.com/supcode/supcode`（Go 1.25）
> 目标：把架构方案落成**可执行、可验收、可自动化**的开发流水线，并用 Skill 把流程固化为 Agent 可执行的指令。

---

## 0. 执行摘要

| 项目 | 结论 |
|---|---|
| 阶段数 | Phase 0 ~ Phase 7，共 8 个阶段；推迟项统一进 2.1+ Backlog |
| 并行策略 | Phase 0/1 内部并行；**Phase 2 可大规模并行**（8 个叶子插件自动派发子 Agent）；Phase 3/4/5/6 的关键路径串行、支线并行 |
| 集成方式 | 分支 `phase/N-*` 集成，PR 门禁通过后合入 `main`；主干始终可编译 |
| CI/CD | GitHub Actions：`ci` / `gate` / `release` / `nightly` 四类工作流 |
| 门禁 | G0~G10 共 11 项，本地脚本与 CI required checks 一一对应 |
| 流程控制 | 8 个阶段 Skill + 1 个编排 Skill；阶段 Skill 管"一阶段做什么"，编排 Skill 管"全版本流程状态机" |
| 质量保障 | 提交规范、阶段验收报告、ADR、任务提示词模板、覆盖率 diff 门禁、架构依赖检查 |

---

## 1. 开发总原则

1. **主干可编译**：任何阶段合并后，`main` 必须 `go build ./... && go vet ./...` 通过。
2. **搬运优先**：Phase 2 的叶子插件要求"逻辑零改动"，用 `git diff --stat` 与 `git log` 审计；只允许新增适配层。
3. **门禁即契约**：每个阶段有明确的 Gate 列表，未过门禁不得进入下一阶段；门禁既能在本地跑，也是 CI required check。
4. **文档与代码同 PR**：架构、使用、阶段报告、任务清单必须与代码在同一 PR 更新，否则门禁 G7 失败。
5. **提交可追溯**：Conventional Commits + 阶段前缀；一个提交只做一件逻辑完整的事。
6. **可回滚**：每阶段单独分支、单独 tag（预发布）；失败可回退到上一阶段 tag。
7. **推迟显式化**：本期不做的事情必须写入 Backlog 并在阶段报告标注，禁止"悄悄的 TODO"。

### 1.1 分支策略

| 分支 | 用途 | 保护 |
|---|---|---|
| `main` | 稳定主干，始终可发布 | 受保护，required checks + review |
| `phase/N-<slug>` | 阶段集成分支，承接该阶段所有 topic PR | 半保护，阶段 gate 通过后合 `main` |
| `phase/N/<topic>` | 单个功能点/插件分支 | 无 |
| `release/vX.Y.Z` | 发布冻结分支（可选） | 受保护 |
| `hotfix/<issue>` | 线上修复 | 受保护 |

流程：`phase/N/<topic>` → PR → CI → `phase/N-<slug>`（阶段回归 + Gate）→ PR → `main`（发布门禁）→ tag `vX.Y.Z`。

### 1.2 提交规范（Conventional Commits）

```
<type>(<scope>): <subject>
```
- type：`feat|fix|refactor|perf|test|docs|ci|chore|build`
- scope：`kernel|plugin-<name>|agent|ci|docs|…`
- 示例：`refactor(plugin-llm): move llm factory into cordis plugin` / `ci: add diff-coverage gate`
- 阶段前缀写入 PR 标题与阶段报告，不强制写入 commit（避免污染语义）。

### 1.3 版本策略

- 2.0 开发期：`v2.0.0-alpha.N`（每阶段一个）、`v2.0.0-beta.N`（Phase 5 后）。
- 正式发布：`v2.0.0`。
- Backlog 项 → `v2.1.0`。

---

## 2. 阶段总览

### 2.1 阶段与依赖 DAG

```
Phase 0 工程地基/基线冻结
   │
   ▼
Phase 1 Cordis 内核 core
   │
   ├──────────────────────────────┐
   ▼                              ▼
Phase 2 叶子插件化(可大规模并行)   Phase 3 状态与交互(依赖 1，部分依赖 2)
   │                              │
   └──────────────┬───────────────┘
                  ▼
            Phase 4 心脏迁移(Aget)
                  │
        ┌─────────┴──────────┐
        ▼                    ▼
Phase 5 双子进程/远程   Phase 6 dsh 生态兼容(可与 5 并行)
        └─────────┬──────────┘
                  ▼
            Phase 7 收尾与发布
```

### 2.2 总览表

| 阶段 | 名称 | 关键路径 | 并行度 | 子 Agent 派发 | 版本标记 |
|---|---|---|---|---|---|
| P0 | 工程地基与基线冻结 | 串行 | 中 | 4 个子 Agent | alpha.0 |
| P1 | Cordis 内核 core | 串行 | 高 | 4 个子 Agent | alpha.1 |
| P2 | 叶子插件化 | 串行入口 | **极高** | 8 个子 Agent 并行 | alpha.2 |
| P3 | 状态与交互 | 串行 | 中 | 3 个子 Agent | alpha.3 |
| P4 | 心脏迁移（Agent） | 串行 | 中 | 3 个子 Agent | alpha.4 |
| P5 | 双子进程与远程 | 串行 | 中 | 2 个子 Agent | beta.1 |
| P6 | dsh 生态兼容 | 可与 P5 并行 | 高 | 3 个子 Agent | beta.2 |
| P7 | 收尾与发布 | 串行 | 低 | 2 个子 Agent | v2.0.0 |

### 2.3 可并行/自动派发规则（供编排 Skill 使用）

- **可并行条件**：任务之间无共享可写文件、无服务依赖（或依赖已就绪）、可独立测试。
- **必须串行条件**：修改同一核心文件、存在 `inject` 依赖链、需要前一任务产物作为输入。
- **派发协议**：编排 Skill 为每个并行任务生成独立任务提示词（`tasks/phase-N/<task>.md`），交给子 Agent；子 Agent 在自己的分支 `phase/N/<task>` 工作，提交后由编排 Skill 统一合入 `phase/N-<slug>` 并跑门禁。

---

## 3. 阶段详细设计

> 每个阶段包含：目标 / 功能点（实现 or 推迟）/ 执行顺序与并行 / 子 Agent 派发 / 交付物 / 验收标准 / 门禁。

### Phase 0 — 工程地基与基线冻结

**目标**：固化 v1 行为基线，搭好 2.0 目录骨架与 CI 骨架，让后续阶段有"对照组"。

| 功能点 | 状态 |
|---|---|
| 跑通 `go build/vet/test`，记录 v1 基线（含已知失败项） | 实现 |
| 建立 2.0 目录骨架 `core/` `plugin/` `cmd/sup` `cmd/supd/` | 实现 |
| `ci.yml` 骨架（lint/build/test/race/arch/skill 校验） | 实现 |
| 提交规范、PR 模板、CODEOWNERS、分支保护 | 实现 |
| 阶段报告模板、任务提示词模板、ADR 模板 | 实现 |
| 旧 `internal/*` 保持不动（后续阶段搬运） | — |

**执行顺序**：基线（串行，必须先做）→ 目录骨架 / CI 骨架 / 规范模板（三者并行）。

**子 Agent 派发**：
1. `baseline-runner`：跑基线并产出 `docs/baseline-v1.md`
2. `repo-scaffolder`：建目录骨架 + 占位 `doc.go`
3. `ci-builder`：写 `.github/workflows/ci.yml` + 扩展 `scripts/check_arch.sh`
4. `docs-templater`：模板与规范

**交付物**：`docs/baseline-v1.md`、`.github/workflows/ci.yml`、目录骨架、`docs/templates/*`、`tasks/acceptance/phase-0.md`。

**验收标准**：
- [ ] v1 基线记录完整，失败项与原因清单化
- [ ] 目录骨架存在且 `go build ./...` 通过
- [ ] CI 在 PR 上全部绿灯
- [ ] 模板与规范文件齐全

**门禁**：G0、G1、G2、G7、G8。

---

### Phase 1 — Cordis 内核 core

**目标**：实现与 Cordis 语义等价的 Go 微内核，不接任何业务代码。

| 功能点 | 状态 |
|---|---|
| `Context` + 服务注册表（`Provide/Use`） | 实现 |
| 事件总线五模式（emit/waterfall/parallel/serial/bail） | 实现 |
| `Scope/Fork/Isolate` + 引用计数 | 实现 |
| `Effect` 可逆副作用 + 逆序 Disposer | 实现 |
| `Loader`：`inject` 拓扑、生命周期、Unload/Reload | 实现 |
| `Plugin` 清单与配置树（每插件 config） | 实现 |
| 内核自测 + Demo 插件（验证全部原语） | 实现 |
| WASM 插件运行时 | **推迟 2.1** |
| 动态 `.so` 插件后端 | **推迟 2.1** |

**执行顺序**：Context/Service（串行基础）→ 事件 / Scope+Effect / Loader（可并行）→ 集成自测。

**子 Agent 派发**：`kernel-context`、`kernel-event`、`kernel-scope-effect`、`kernel-loader`（4 并行，契约先冻结接口）。

**交付物**：`core/*.go`、`core/*_test.go`、`plugin/demo/`、`docs/adr/ADR-0001-cordis-kernel.md`、`tasks/acceptance/phase-1.md`。

**验收标准**：
- [ ] 五模式事件各有测试，waterfall 可短路/包装
- [ ] `inject` 依赖拓扑加载正确，缺依赖时报错明确
- [ ] `Effect` 卸载时逆序回放，无资源泄漏（循环 reload 压测）
- [ ] `Scope` 隔离通过 `-race`
- [ ] 内核不 import 任何 `plugin/*` 或业务包（G5）
- [ ] 内核核心包覆盖率 ≥ 85%

**门禁**：G0~G6、G8。

---

### Phase 2 — 叶子插件化（可大规模并行）

**目标**：把无状态/低依赖的 v1 模块**逻辑零改动**地搬成插件。

| 插件 | v1 来源 | 状态 |
|---|---|---|
| `plugin-llm` + 4 适配器 | `internal/llm/*` | 实现 |
| `plugin-tools` + 6 工具 | `internal/tools/*` | 实现 |
| `plugin-mcp` | `internal/mcp/*` | 实现 |
| `plugin-skill` + `tool-skill` | `internal/skill/*` | 实现 |
| `plugin-permission` | `internal/permission/*` | 实现 |
| `plugin-memory` | `internal/memory/*` | 实现 |
| `plugin-worktree` | `internal/worktree/*` | 实现 |
| `plugin-hooks-audit` / `plugin-hooks-git` | `internal/hooks/builtin/*` | 实现 |
| 新 LLM 适配器（Gemini/Azure 等） | — | **推迟 2.1** |

**执行顺序**：先冻结插件契约与服务键（串行）→ 8 个插件并行迁移 → 统一集成回归。

**子 Agent 派发**：**8 个并行子 Agent**，每个负责一个插件：搬运代码、补适配层、写插件单测、产出 `tasks/phase-2/<plugin>.md`。合并由编排 Skill 串行完成（避免冲突）。

**交付物**：`plugin/<name>/*`、对应 `*_test.go`、`tasks/acceptance/phase-2.md`、`git diff --stat` 审计记录。

**验收标准**：
- [ ] 每个插件单测通过，且原模块测试可复用部分全部通过
- [ ] `git diff` 证明业务逻辑未被重写（仅 import/装配变化）
- [ ] 每个插件可通过 `sup.plugin.json` 声明并被内核按 `inject` 装载
- [ ] 未引入对 `internal/*` 的反向依赖
- [ ] 集成测试：工具池同时挂载 6 内置工具 + skill 工具

**门禁**：G0~G6、G8、G9。

---

### Phase 3 — 状态与交互

**目标**：迁移会话、上下文、TUI、CLI/命令，恢复"可用产品"形态。

| 功能点 | 状态 |
|---|---|
| `plugin-session`（SQLite 会话，兼容 v1） | 实现 |
| `plugin-context`（上下文压缩/Token） | 实现 |
| `plugin-tui`（interaction seam，订阅事件渲染） | 实现 |
| `plugin-cli` + `plugin-command`（Slash 命令） | 实现 |
| `plugin-config` 旧配置兼容层 | 实现 |
| 流式输出打通到 TUI | **推迟 2.1**（架构方案已列 Backlog） |
| TUI Ctrl+C → ctx 取消 | **推迟 2.1** |
| Web UI / HTTP server | **推迟 2.x** |

**执行顺序**：session / context（并行）→ tui（依赖 session）→ cli/command（依赖 command + agent 桩）。可先用 mock agent 跑通 TUI。

**子 Agent 派发**：`session`、`context`、`tui`（3 并行；`cli/command` 随后串行）。

**交付物**：`plugin/session|context|tui|cli|command/*`、旧配置迁移测试、`tasks/acceptance/phase-3.md`。

**验收标准**：
- [ ] 单发查询与 TUI 交互在 mock LLM 下可用
- [ ] 旧 `~/.supcode/config.yaml` 可无缝加载
- [ ] 会话创建/切换/持久化/恢复闭环
- [ ] 上下文超限触发压缩，Token 统计正确

**门禁**：G0~G6、G8、G9。

---

### Phase 4 — 心脏迁移（Agent）

**目标**：把 ReAct 循环迁移为 `plugin-agent`，接入完整工具流水线，能力矩阵全绿。

| 功能点 | 状态 |
|---|---|
| `plugin-agent`：ReAct 状态机 + Planner + Selector + Corrector | 实现 |
| Plan Mode（`ctx.planMode` 折叠状态） | 实现 |
| `agent/*` 生命周期事件、`llm/chunk` 事件 | 实现 |
| 工具流水线完整接入：`tools/pre-execute → guard → execute → post-execute → result` | 实现 |
| `plugin-subagent` + `Context.Fork()` 隔离 | 实现 |
| `pkg.Agent` 接口适配为服务契约（保留诊断方法） | 实现 |
| 并行工具调用 | **推迟 2.1** |
| Agent 级 invariants 运行时校验 | **推迟 2.1** |

**执行顺序**：agent 循环（串行核心）→ 事件与流水线（并行）→ subagent（依赖 agent）→ v1/v2 双跑对账。

**子 Agent 派发**：`agent-loop`、`agent-pipeline-events`、`subagent-scope`（3 并行，接口先冻结）。

**交付物**：`plugin/agent/*`、`plugin/subagent/*`、能力矩阵勾选表、双跑对账报告、`tasks/acceptance/phase-4.md`。

**验收标准**：
- [ ] 架构方案 §5 的 C1~C25 能力矩阵全部勾选
- [ ] ReAct 端到端场景（规划→选择→执行→观察→修正→交付）通过
- [ ] 权限在 `pre-execute` 短路，审计/Git 在 `post-execute` 生效
- [ ] v1 与 v2 对同一批集成用例结果一致
- [ ] `-race` 通过

**门禁**：G0~G6、G8、G9。

---

### Phase 5 — 双子进程与远程

**目标**：提供用户入口与常驻/远程形态。

| 功能点 | 状态 |
|---|---|
| `cmd/sup`（单发 / TUI / 子命令） | 实现 |
| `cmd/supd`（常驻守护，多会话） | 实现 |
| `plugin-mcp-server`（把 Harness 暴露为 MCP server） | 实现 |
| 信号处理、优雅关闭、配置加载 | 实现 |
| goreleaser 配置切换到 `cmd/sup` | 实现 |
| HTTP/Web API | **推迟 2.x** |

**执行顺序**：`sup`（依赖 Phase 3/4）与 `supd` 可并行；`mcp-server` 随后。

**子 Agent 派发**：`cmd-sup`、`cmd-supd`（2 并行）→ `mcp-server`。

**交付物**：`cmd/sup/*`、`cmd/supd/*`、`.goreleaser.yaml` 更新、`tasks/acceptance/phase-5.md`。

**验收标准**：
- [ ] `sup "…"` 单发可用；`sup` 进入 TUI
- [ ] `supd` 启动后 MCP 客户端可列举并调用工具
- [ ] 优雅关闭不泄漏连接/进程
- [ ] goreleaser `--snapshot` 本地成功

**门禁**：G0~G6、G8、G9、G10（快照）。

---

### Phase 6 — dsh 生态兼容（可与 Phase 5 并行）

**目标**：打通与 dsh/Cordis 生态的插件互通。

| 功能点 | 状态 |
|---|---|
| Sup Plugin Protocol（SPP）规范与实现 | 实现 |
| `plugin-dsh-bridge`（Node 侧车） | 实现 |
| dsh 工具 → `ctx.tools` 映射 | 实现 |
| dsh LLM provider → `ctx.llm` 映射 | 实现 |
| `sup.plugin.json` 规范 + dsh `package.json` 识别 | 实现 |
| 至少 1 个真实 dsh 插件端到端验证 | 实现 |
| dsh 事件面全量覆盖 | **推迟 2.x** |
| WASM 沙箱插件 | **推迟 2.1** |

**执行顺序**：SPP 协议冻结（串行）→ 侧车/工具映射/LLM 映射（并行）→ 端到端验证。

**子 Agent 派发**：`spp-spec`、`dsh-sidecar`、`dsh-tool-llm-mapping`（3 并行）。

**交付物**：`docs/spp-v1.md`、`plugin/dshbridge/*`、`tests/compat/dsh/*`、`tasks/acceptance/phase-6.md`。

**验收标准**：
- [ ] dsh 插件注册的工具能在 Sup 中被 Agent 调用
- [ ] Sup 作为 MCP server 能被 dsh 调用
- [ ] 协议往返测试（waterfall 往返）通过
- [ ] Node 缺失时优雅降级（不阻塞内置功能）

**门禁**：G0~G6、G8、G9、G-compat。

---

### Phase 7 — 收尾与发布

**目标**：去除 v1 残留，完成文档与发布。

| 功能点 | 状态 |
|---|---|
| 删除 `internal/build.go`、`wire.go`、`app.go` 等旧装配 | 实现 |
| 全量文档更新（README/架构/使用/开发/迁移指南） | 实现 |
| CHANGELOG + `v2.0.0` tag + goreleaser 发布 | 实现 |
| 安装脚本更新（`install.sh`/`install.ps1` → `sup`） | 实现 |
| 性能基准（benchmark）与对比报告 | 实现 |
| 覆盖率总览达标 | 实现 |
| Web UI | **推迟 2.x** |

**执行顺序**：残余清理（串行）→ 文档 / 基准（并行）→ 发布（串行）。

**子 Agent 派发**：`cleanup`、`docs-bench`（2 并行）→ 发布 Agent。

**交付物**：发布产物、`CHANGELOG.md`、迁移指南、`tasks/acceptance/phase-7.md`。

**验收标准**：
- [ ] `grep -r "internal/build"` 无残留引用
- [ ] 全量测试 + `-race` + 覆盖率达标
- [ ] goreleaser 正式发布成功，多平台产物完整
- [ ] 文档与代码一致

**门禁**：G0~G10 全部。

---

## 4. 推迟到下一版本（2.1+ Backlog）

| # | 项 | 来源 | 目标版本 |
|---|---|---|---|
| B1 | Agent 并行工具调用 | 架构方案 §1.4 | 2.1 |
| B2 | 流式输出打通到 TUI | 架构方案 §1.4 | 2.1 |
| B3 | TUI Ctrl+C → ctx 取消 | 架构方案 §1.4 | 2.1 |
| B4 | WASM 插件运行时（wazero） | 架构方案 §6.3 | 2.1 |
| B5 | 动态 `.so` 插件后端 | 架构方案 §6.3 | 2.1 |
| B6 | 新 LLM/Embedding 适配器 | 本规划 P2 | 2.1 |
| B7 | Agent invariants 运行时校验 | 架构方案 §9 R7 | 2.1 |
| B8 | dsh 事件面全量覆盖 | 本规划 P6 | 2.x |
| B9 | Web UI / HTTP server 插件 | 架构方案 §1.4 | 2.x |
| B10 | 会话模型向"仅追加事件流"演进 | 架构方案 §6.5 | 2.x |

---

## 5. GitHub Actions CI/CD 设计

### 5.1 工作流清单

| 工作流 | 触发 | 作用 | 对应门禁 |
|---|---|---|---|
| `.github/workflows/ci.yml` | push / PR | lint、build 矩阵、test+race、覆盖率、架构检查、skill 校验 | G0~G6 |
| `.github/workflows/gate.yml` | PR labeled `phase-gate` / 手动 | 阶段门禁：验收报告、任务清单、阶段回归、人工审批 | G7 + 阶段 |
| `.github/workflows/release.yml` | tag `v*` | goreleaser 发布、校验和、Release Notes | G10 |
| `.github/workflows/nightly.yml` | schedule | 全量 race + e2e + 覆盖率趋势 + flaky 检测 | 趋势 |
| `.github/workflows/compat.yml` | PR（Phase 6 起） | dsh 侧车互操作测试 | G-compat |
| `.github/workflows/docs.yml` | PR `docs/**` | markdown lint + 链接检查 | G7 |

设计要点：
- Go 1.25；`actions/setup-go` 缓存 `go.sum`；矩阵 `os ∈ {ubuntu, windows, macos}` × `go ∈ {1.25.x}`（矩阵以 ubuntu 为主，跨平台编译在 build job）。
- `concurrency` 按分支取消旧运行，节省额度。
- 覆盖率用 `go test -coverprofile` + `diff-cover` 做"新增代码覆盖率"门禁。
- 架构门禁调用仓库自带 `scripts/check_arch.sh`，并扩展 v2 规则。
- Skill 校验用 `scripts/check_skills.sh` 遍历 `skills/*/skill.json` 做 JSON + 名称 + semver 校验；阶段门禁用 `scripts/check_task_checkboxes.sh` 校验任务看板。
- **CGO 注意**：`internal/depslock/lock.go` 空白导入了 cgo 版 `mattn/go-sqlite3`，因此 build job 采用**各 OS 原生构建**而非 `CGO_ENABLED=0` 交叉编译；goreleaser 只编译 `cmd/*` 可达依赖图（不含 depslock），故 `CGO_ENABLED=0` 发布不受影响。Phase 0 可评估移除该空白导入、统一改用纯 Go 的 `modernc.org/sqlite`。

### 5.2 `ci.yml`（核心）

```yaml
name: ci
on:
  push:
    branches: [main, 'phase/**']
  pull_request:
    branches: [main, 'phase/**']

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

env:
  GO_VERSION: '1.25.x'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '${{ env.GO_VERSION }}', cache: true }
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with: { version: latest, args: --timeout=5m }

  build:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, windows-latest, macos-latest]   # 原生构建，规避 CGO 交叉编译问题
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '${{ env.GO_VERSION }}', cache: true }
      - run: go build ./...
      - run: go vet ./...

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '${{ env.GO_VERSION }}', cache: true }
      - run: go test -race -count=1 -coverprofile=coverage.out ./...
      - name: coverage summary
        run: go tool cover -func=coverage.out | tail -n 1
      - uses: actions/upload-artifact@v4
        with: { name: coverage, path: coverage.out }

  diff-coverage:
    runs-on: ubuntu-latest
    needs: test
    if: github.event_name == 'pull_request'
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '${{ env.GO_VERSION }}', cache: true }
      - run: go test -count=1 -coverprofile=coverage.out ./...
      - run: pip install diff-cover
      - name: new-code coverage >= 80%
        run: diff-cover coverage.out --compare-branch=origin/${{ github.base_ref }} --fail-under=80

  arch:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: bash scripts/check_arch.sh

  skills:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: bash scripts/check_skills.sh
```

### 5.3 `gate.yml`（阶段门禁）

```yaml
name: gate
on:
  pull_request:
    types: [labeled, synchronize, reopened]
  workflow_dispatch:

jobs:
  phase-gate:
    if: contains(github.event.pull_request.labels.*.name, 'phase-gate')
    runs-on: ubuntu-latest
    environment: phase-gate            # 配置人工审批
    steps:
      - uses: actions/checkout@v4
      - name: verify acceptance report
        run: |
          phase=$(grep -oP 'phase/\K[0-9]+' <<< "${GITHUB_HEAD_REF}" | head -1)
          test -f "tasks/acceptance/phase-${phase}.md" || (echo "missing phase-${phase}.md"; exit 1)
      - name: verify task checklist
        run: bash scripts/check_task_checkboxes.sh
      - uses: actions/setup-go@v5
        with: { go-version: '1.25.x', cache: true }
      - run: bash scripts/test.sh
      - run: bash scripts/check_arch.sh
```

### 5.4 `release.yml`（CD）

```yaml
name: release
on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { fetch-depth: 0 }
      - uses: actions/setup-go@v5
        with: { go-version: '1.25.x', cache: true }
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### 5.5 `nightly.yml`

```yaml
name: nightly
on:
  schedule: [{ cron: '0 18 * * *' }]
  workflow_dispatch:

jobs:
  full:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.25.x', cache: true }
      - run: go test -race -count=3 ./...        # 重复跑，探测 flaky
      - run: go test -count=1 -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out | tail -n 1
```

### 5.6 分支保护（Required checks）

`main` 与 `phase/N-*` 分支要求以下检查通过后方可合并：

`lint` · `build (linux/amd64)` · `build (windows/amd64)` · `build (darwin/arm64)` · `test` · `diff-coverage` · `arch` · `skills`

加上：至少 1 位 reviewer、禁止直推、要求分支最新、建议 Squash 合并。

---

## 6. 门禁（Gates）定义

| 门禁 | 含义 | 本地命令 | CI 对应 |
|---|---|---|---|
| G0 编译 | `build` + `vet` 通过 | `go build ./... && go vet ./...` | build job |
| G1 静态检查 | lint 0 issue | `bash scripts/lint.sh` | lint job |
| G2 单元测试 | 全量通过，无新增失败 | `go test ./...` | test job |
| G3 竞态 | `-race` 通过 | `bash scripts/test.sh` | test job |
| G4 覆盖率 | 新增代码 ≥80%，整体不下降 >1% | `diff-cover … --fail-under=80` | diff-coverage job |
| G5 架构约束 | 依赖方向正确，内核不依赖业务 | `bash scripts/check_arch.sh` | arch job |
| G6 Skill 规范 | 所有 `skill.json` 合法 | `bash scripts/check_skills.sh` | skills job |
| G7 文档/任务 | 阶段报告 + 任务清单 + 文档更新 | `bash scripts/check_docs.sh` | gate job |
| G8 提交规范 | Conventional Commits，无直推 | commitlint / 分支保护 | 分支保护 |
| G9 集成 | mock LLM 端到端通过 | `go test ./tests/integration/...` | gate job |
| G10 发布 | tag 可构建、goreleaser 成功、CHANGELOG | `goreleaser release --snapshot` | release job |
| G-compat | dsh 侧车互操作（Phase 6 起） | `bash scripts/test_compat.sh` | compat job |

> v2 新增/扩展的架构规则（写入 `check_arch.sh`）：
> 1. `core/` 不得 import `plugin/*`、`internal/*`；
> 2. `plugin/X` 不得直接 import `plugin/Y`（只能经服务键/事件）；
> 3. `pkg/` 不得 import `core/`、`plugin/`、`internal/`；
> 4. 迁移期允许 `plugin/*` import `internal/*`，Phase 7 后禁止（渐进收紧）。

---

## 7. 文档、提交与任务提示词维护规范

| 对象 | 何时更新 | 位置 | 要求 |
|---|---|---|---|
| 架构方案 | 架构决策变化 | `docs/Sup Harness 2.0 架构重整方案.md` | 同 PR 更新 |
| 开发规划 | 阶段范围/门禁变化 | `docs/Sup Harness 2.0 开发规划与 CI-CD.md` | 同 PR 更新 |
| ADR | 关键决策 | `docs/adr/ADR-XXXX-*.md` | 每决策一篇 |
| 阶段验收报告 | 每阶段结束 | `tasks/acceptance/phase-N.md` | Gate 依赖存在性 |
| 任务提示词 | 派发子 Agent 前 | `tasks/phase-N/<task>.md` | 模板化，含验收命令 |
| 任务总清单 | 阶段内实时 | `tasks/phase-N/README.md` | checkbox 驱动 |
| CHANGELOG | 每次合并到 main | `CHANGELOG.md` | goreleaser 生成 |
| 使用/开发文档 | 行为变化 | `docs/*.md`、`README.md` | 同 PR 更新 |

任务提示词模板（`docs/templates/task-prompt.md`）：

```md
# Task: <phase-N / 功能点名>
## 背景与目标
## 输入（依赖产物/接口契约）
## 交付物（文件清单）
## 实现要求（禁止重写逻辑/接口约束）
## 验收命令（本地可复现）
## Git 规范（分支/commit 前缀）
## 文档维护（需更新的文档）
## 完成定义（DoD checklist）
```

### 7.1 一键生成任务看板与验收骨架

- 数据源：`scripts/phase_manifest.json`（阶段元数据与功能点清单）
- 生成器：`scripts/gen_phase_tasks.py`（核心，UTF-8 安全）；`scripts/gen_phase_tasks.ps1`（Windows 包装）、`scripts/gen_phase_tasks.sh`（Linux/macOS 包装）
- 生成物：`tasks/phase-N/README.md`、`tasks/acceptance/phase-N.md`
- 用法与调用说明见 [`docs/skills-usage.md`](skills-usage.md)

---

## 8. Skill 体系设计

> 目标：把本规划固化为 Agent 可执行的 Skill。共 **9 个 Skill**：8 个阶段 Skill + 1 个编排 Skill。
> 目录规范沿用仓库现有格式：`skills/<name>/skill.json` + `skills/<name>/prompts/system.md`。

### 8.1 阶段 Skill（一阶段一个）

| Skill 名 | 覆盖阶段 | 核心作用 |
|---|---|---|
| `harness-phase-0-baseline` | P0 | 基线冻结、地基搭建 |
| `harness-phase-1-kernel` | P1 | Cordis 内核实现 |
| `harness-phase-2-leaf-plugins` | P2 | 叶子插件并行迁移 |
| `harness-phase-3-state-interaction` | P3 | 状态与交互迁移 |
| `harness-phase-4-agent-core` | P4 | Agent 心脏迁移 |
| `harness-phase-5-daemon-remote` | P5 | 双子进程与远程 |
| `harness-phase-6-dsh-compat` | P6 | dsh 生态兼容 |
| `harness-phase-7-release` | P7 | 收尾与发布 |
| `harness-lifecycle-orchestrator` | 全局 | 全版本流程编排 |

**每个阶段 Skill 的 system.md 统一包含：**
1. 阶段目标与前置依赖（上一阶段 Gate 是否满足）；
2. 功能点清单（实现 / 推迟，并链接 Backlog）；
3. 执行顺序与并行性判定；
4. **子 Agent 自动派发计划**（任务切分、输入输出、合并策略）；
5. Git 规范（分支名、commit 前缀、PR 目标）；
6. 文档与任务提示词维护清单；
7. 验收标准 checklist；
8. 门禁命令（本地）与 CI 对应关系；
9. 失败/回滚处理；
10. 输出物：`tasks/acceptance/phase-N.md`。

### 8.2 编排 Skill：`harness-lifecycle-orchestrator`

- **角色**：Sup Harness 项目的总控 Agent（Tech Lead / Release Manager）。
- **职责**：
  1. 读取本规划与架构方案，确定当前阶段与版本；
  2. 校验前置 Gate，决定"启动 / 阻塞 / 回滚"；
  3. 调用（或派发）对应阶段 Skill；
  4. 执行并行任务的自动派发与合并；
  5. 维护阶段 tag、CHANGELOG、状态看板；
  6. 管理 Backlog 与版本迁移；
  7. 在阶段结束时汇总验收报告并推进版本号。
- **状态机**：

```
IDLE → PRECHECK → PHASE_RUNNING → PHASE_GATE → (PASS → NEXT_PHASE | FAIL → FIX_LOOP)
      → ... → RELEASE_GATE → RELEASED → BACKLOG_TRIAGE → IDLE
```

- **决策规则**：Gate 未过禁止推进；并行冲突禁止并行；Backlog 只增不减需评审；发行冻结后进入 hotfix 模式。
- **输出物**：`docs/STATUS.md`（当前阶段/版本/门禁状态）、`CHANGELOG.md`、阶段 tag。

### 8.3 Skill 与仓库现有 Skill 系统的一致性

- 位置：仓库根 `skills/`（`GetSkillDirs()` 的 builtin 目录），可被 SupCode/Sup Harness 通过 `skill list` / `skill load` 发现与加载。
- 清单字段：`name` / `version` / `description` / `author` / `tools` / `requires`。
- 阶段 Skill 之间通过 `requires` 表达顺序（如 `harness-phase-2-leaf-plugins` requires `harness-phase-1-kernel`）。
- 编排 Skill 的 `requires` 列出全部阶段 Skill。
- 名称必须匹配 `^[a-z0-9-]+$`，版本遵循 semver。

---

## 9. 风险与回滚

| 风险 | 对策 |
|---|---|
| 并行子 Agent 冲突 | 冻结接口契约；按文件/包切分；编排 Skill 串行合并 |
| 阶段推进过快导致返工 | Gate 强制；每阶段独立 tag，可回退 |
| CI 额度消耗 | 缓存 + concurrency 取消 + nightly 仅在主干 |
| 覆盖率门禁误伤迁移代码 | 迁移代码不计入 diff-coverage，或按"逻辑零改动"白名单 |
| dsh 兼容不及预期 | 限定 MVP（工具/LLM seam），其余进 Backlog |
| Go 动态插件限制 | 主推进程外 SPP 插件，`.so`/WASM 进 Backlog |

**回滚策略**：阶段失败 → 回退到 `phase/N-1` 的 tag；`main` 出问题 → revert PR + 从最近 tag 重建；发布问题 → yank Release + 发 hotfix。

---

## 附录 A：阶段 ↔ Skill ↔ 门禁映射

| 阶段 | Skill | 主要门禁 | 阶段产物 |
|---|---|---|---|
| P0 | `harness-phase-0-baseline` | G0/G1/G2/G7/G8 | baseline、ci.yml、骨架 |
| P1 | `harness-phase-1-kernel` | G0~G6 | core、ADR-0001 |
| P2 | `harness-phase-2-leaf-plugins` | G0~G6/G8/G9 | 8 插件 |
| P3 | `harness-phase-3-state-interaction` | G0~G6/G8/G9 | session/context/tui/cli |
| P4 | `harness-phase-4-agent-core` | G0~G6/G8/G9 | agent/subagent、能力矩阵 |
| P5 | `harness-phase-5-daemon-remote` | G0~G6/G8/G9/G10 | sup/supd/mcp-server |
| P6 | `harness-phase-6-dsh-compat` | G0~G6/G8/G9/G-compat | SPP、dsh-bridge |
| P7 | `harness-phase-7-release` | G0~G10 | v2.0.0 发布 |
| 全局 | `harness-lifecycle-orchestrator` | 全部 | STATUS.md、tag、CHANGELOG |

## 附录 B：角色与职责（RACI 简表）

| 活动 | 编排 Agent | 阶段 Agent | 子 Agent | 人工 |
|---|---|---|---|---|
| 阶段启动/推进 | A/R | C | I | C |
| 功能点实现 | C | A | R | I |
| 并行派发与合并 | A/R | C | R | I |
| 门禁执行 | A | R | C | C（审批） |
| 文档/任务提示词 | C | A/R | R | I |
| 发布 | A/R | C | C | C（审批） |

> R=负责执行，A=最终问责，C=咨询，I=知会。
