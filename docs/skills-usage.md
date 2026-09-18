# 如何调用 Sup Harness 开发 Skill

> 适用对象：SupCode / Sup Harness 的 Agent（以及 opencode、Claude Code 等外部 Agent）。
> 相关 Skill：`harness-phase-0-baseline` … `harness-phase-7-release` + `harness-lifecycle-orchestrator`。

---

## 1. Skill 是什么（本项目的机制）

Skill 不是可执行程序，而是一个**提示词包**：

```
skills/<name>/
├── skill.json          # 元数据：name / version / description / tools / requires
└── prompts/system.md   # 专家指令（加载后作为工具结果注入对话，Agent 据此执行）
```

加载后，`prompts/system.md` 的内容会作为工具结果返回给 Agent，Agent 按其中的流程工作。因此"调用 Skill"= 让 Agent `load` 该 Skill 并遵循它。

---

## 2. 发现与安装

Skill 的搜索路径（`internal/skill.GetSkillDirs()`）：

| 优先级 | 路径 | 说明 |
|---|---|---|
| 0（builtin） | `./skills` | **相对当前工作目录**；在仓库根目录运行即可发现本文档的 Skill |
| 1（user） | `~/.supcode/skills` | 用户级 |
| 2（project） | 向上查找最近的 `<dir>/.supcode/skills` | 项目级 |

列出所有可用 Skill：

```bash
# 必须在仓库根目录执行，builtin 才会解析到 ./skills
cd /path/to/sup-code
supcode skill list
```

预期能看到：`code-review`、`test-generator`，以及 `harness-phase-0-baseline` … `harness-phase-7-release`、`harness-lifecycle-orchestrator`。

安装到用户目录（脱离仓库也能用）：

```bash
supcode skill install ./skills/harness-phase-1-kernel
supcode skill install ./skills/harness-lifecycle-orchestrator
```

---

## 3. 在 SupCode / Sup Harness 中调用

Agent 内置 `skill` 工具，支持 `action=list` / `action=load`。

### 3.1 单发模式（推荐入口：编排 Skill）

```bash
supcode "请先用 skill 工具列出技能，然后加载 harness-lifecycle-orchestrator，按其状态机读取 docs/STATUS.md 与规划文档，推进 Sup Harness 2.0 的下一阶段。"
```

### 3.2 交互模式

进入 TUI 后输入同样的自然语言指令：

```
> 加载 harness-lifecycle-orchestrator，检查当前阶段并推进
```

### 3.3 直接加载某个阶段 Skill

```bash
supcode "加载 harness-phase-2-leaf-plugins，按其中的并行派发计划执行 Phase 2"
```

### 3.4 Agent 内部实际发生的调用

LLM 会依次调用：

```json
{"action": "list"}
{"action": "load", "name": "harness-lifecycle-orchestrator"}
{"action": "load", "name": "harness-phase-0-baseline"}
```

> **注意**：当前 `skill` 工具的 `load` 只返回该 Skill 自身的指令，**不会自动加载 `requires` 里的依赖**。因此编排 Skill 的 `system.md` 明确要求 Agent「按序显式 load 阶段 Skill」。若希望自动注入依赖，可作为 Phase 1 的内核增强项（loader 递归注入 `requires`）。

---

## 4. 推荐调用顺序

```
1) harness-lifecycle-orchestrator      # 总控：读 STATUS/规划，校验门禁，决定阶段
        │ 显式 load
        ▼
2) harness-phase-N-<slug>              # 阶段执行：功能点/顺序/并行/子 Agent 派发
        │ 由阶段 Skill 生成任务提示词
        ▼
3) 子 Agent（在 phase/N/<topic> 分支） # 具体实现
        │ 合并回 phase/N-<slug>
        ▼
4) 门禁（scripts/*.sh + .github/workflows/gate.yml）
        │ 通过
        ▼
5) 回到 1) 推进下一阶段，直至 v2.0.0 发布
```

---

## 5. 生成阶段任务看板与验收骨架

```bash
# Windows
powershell -ExecutionPolicy Bypass -File scripts/gen_phase_tasks.ps1        # 追加缺失项
powershell -ExecutionPolicy Bypass -File scripts/gen_phase_tasks.ps1 -Force # 全部重写

# Linux / macOS / CI
bash scripts/gen_phase_tasks.sh
bash scripts/gen_phase_tasks.sh --force

# 跨平台（直接调 Python 核心）
python scripts/gen_phase_tasks.py [--force]
```

生成物：`tasks/phase-0..7/README.md`、`tasks/acceptance/phase-0..7.md`；数据源为 `scripts/phase_manifest.json`。

---

## 6. 在外部 Agent（opencode / Claude Code 等）中使用

这些 Skill 是标准 Markdown 提示词，可直接复用：

1. 把 `skills/harness-*` 复制到外部 Agent 的 skills/agents 目录；或
2. 直接在对话中引用文件：`请阅读并遵循 skills/harness-phase-1-kernel/prompts/system.md`；或
3. 把 `skill.json` 的 `name/description` 注册为该 Agent 的可用技能。

---

## 7. 校验与排障

```bash
bash scripts/check_skills.sh              # 校验所有 skill.json 合法
bash scripts/check_task_checkboxes.sh     # 校验任务看板含 checkbox
supcode skill list                        # 确认能被发现（需在仓库根执行）
```

| 现象 | 原因 | 处理 |
|---|---|---|
| `skill list` 看不到 harness-* | 未在仓库根运行，builtin `./skills` 未命中 | `cd` 到仓库根，或 `skill install` 到用户目录 |
| Agent 不加载阶段 Skill | 只加载了编排 Skill | 编排 Skill 指令要求显式 load；也可在 prompt 中直接点名 |
| `skill.json` 校验失败 | name/version 不合法 | 必须 `^[a-z0-9-]+$` 且 semver |
| `check_task_checkboxes.sh` 失败 | 看板无 checkbox | 用生成器重建或补 `- [ ]` 项 |

---

## 8. 在 opencode 中调用（桥接层）

SupCode 的 `skills/*`（`skill.json` + `prompts/system.md`）**不会被 opencode 识别**。opencode 只扫描
`.opencode/skill(s)/<name>/SKILL.md`（带 YAML frontmatter）。因此本仓库提供一层**薄适配**：

```
.opencode/
├── skills/
│   ├── harness-lifecycle-orchestrator/SKILL.md   # 适配器 → 指向 skills/harness-lifecycle-orchestrator/prompts/system.md
│   └── harness-phase-0-baseline/SKILL.md ...     # 每个阶段一个
└── command/
    ├── harness-start.md                          # /harness-start
    └── harness-phase.md                          # /harness-phase <n|slug>
```

适配器**不复制内容**，只让 opencode Agent 去读规范文件（单一事实来源）。

### 8.1 使用前提

opencode 在**启动时**加载配置/技能/命令，且项目级 `.opencode/` 从**当前工作目录**向上查找。因此：

- 在 **`D:\codes\sup-code`** 目录下启动 opencode，上述技能与命令才会生效；
- 若想任意目录可用，把适配器复制到 `~/.config/opencode/`（全局作用域），或重启后在会话里直接引用文件路径。

### 8.2 调用方式

```text
/harness-start                 # 启动编排：读 STATUS+规划 → 校验门禁 → 推进下一阶段
/harness-phase 0               # 运行 Phase 0
/harness-phase 2               # 运行 Phase 2（叶子插件化，自动并行派发子 Agent）
```

也可以直接用技能名（重启后技能出现在 Agent 的可用列表里）：

```text
用 harness-phase-1-kernel 技能开始 Phase 1
```

### 8.3 不重启的临时调用

即使不重启 opencode，也可以直接让 Agent 读取规范文件并执行：

```text
读取并遵循 skills/harness-lifecycle-orchestrator/prompts/system.md，推进 Sup Harness 2.0 下一阶段
```

### 8.4 关系图

```
skills/harness-*/prompts/system.md   ← 规范内容（SupCode 与人工维护，单一事实来源）
        ▲                          
        │ 读取
.opencode/skills/harness-*/SKILL.md  ← opencode 适配器（薄）
        │
        ▼
.opencode/command/harness-*.md       ← 你的 slash 命令入口
```

