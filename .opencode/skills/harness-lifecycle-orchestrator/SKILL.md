---
name: harness-lifecycle-orchestrator
description: Use when coordinating or advancing the Sup Harness 2.0 multi-phase development lifecycle, checking gates, choosing the next phase, dispatching subagents, or releasing versions. SupCode-format canonical skill lives at skills/harness-lifecycle-orchestrator/prompts/system.md. Triggers: sup harness, harness 2.0, 编排, 开发流程, lifecycle, next phase, 阶段推进.
---

# Sup Harness lifecycle orchestrator (opencode adapter)

This repo's canonical skill is stored in the SupCode skill format (single source of truth):

- `skills/harness-lifecycle-orchestrator/prompts/system.md`
- Metadata: `skills/harness-lifecycle-orchestrator/skill.json`

## Steps
1. Read `skills/harness-lifecycle-orchestrator/prompts/system.md` with the Read tool.
2. Read the authoritative inputs it names:
   - `docs/Sup Harness 2.0 架构重整方案.md`
   - `docs/Sup Harness 2.0 开发规划与 CI-CD.md`
   - `docs/STATUS.md`
   - `tasks/acceptance/phase-*.md`
3. Determine the current state, run the precheck, then load the matching phase skill
   (`.opencode/skills/harness-phase-<n>-<slug>/SKILL.md` → canonical `skills/harness-phase-<n>-<slug>/prompts/system.md`).
4. Enforce gates, dispatch parallel subagents, update `docs/STATUS.md` and the phase acceptance report, and tag versions per the plan.

Do NOT invent phases or skip gates; follow the canonical file exactly.
