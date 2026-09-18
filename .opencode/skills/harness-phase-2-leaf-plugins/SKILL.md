---
name: harness-phase-2-leaf-plugins
description: Use when executing Sup Harness 2.0 Phase 2 (叶子插件化 / leaf plugin migration, high parallelism). Canonical skill at skills/harness-phase-2-leaf-plugins/prompts/system.md. Triggers: phase 2, 叶子插件, leaf plugins, plugin migration.
---

# Sup Harness Phase 2 (opencode adapter)

Canonical instructions: `skills/harness-phase-2-leaf-plugins/prompts/system.md`

## Steps
1. Read the canonical file with the Read tool.
2. Read `docs/Sup Harness 2.0 开发规划与 CI-CD.md` Phase 2 and `tasks/phase-2/README.md`.
3. Freeze service keys/events, then migrate the 8 leaf plugins with logic-zero-change; dispatch one subagent per plugin in parallel and merge serially.
4. Audit `git diff --stat`, fill `tasks/acceptance/phase-2.md`, update `docs/STATUS.md`, run gates.

Follow the canonical file exactly; never rewrite business logic.
